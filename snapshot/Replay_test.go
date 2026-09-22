package snapshot

import (
	"bytes"
	"strconv"
	"testing"
	"time"

	"github.com/rajandhamala/goRedis/helpers"
	"github.com/rajandhamala/goRedis/src"
)

func TestBinaryRecordsAndAbsoluteExpiration(t *testing.T) {
	src.CommandMu.Lock()
	defer src.CommandMu.Unlock()
	clear(src.ActiveKeys)
	future := time.Now().Add(time.Minute).UnixMilli()
	past := time.Now().Add(-time.Minute).UnixMilli()
	data := []byte("SET legacy value 60\n")
	data = append(data, helpers.Strings([]string{"SET", "binary", "hello \r\n\x00雪", "PXAT", strconv.FormatInt(future, 10)})...)
	data = append(data, helpers.Strings([]string{"SET", "expired", "v", "PXAT", strconv.FormatInt(past, 10)})...)
	data = append(data, helpers.Strings([]string{"SET", "persistent", ""})...)
	if err := Replay(bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	if value, err := src.GetKey("binary"); err != nil || value != "hello \r\n\x00雪" {
		t.Fatal(value, err)
	}
	if src.Expiry("binary").UnixMilli() != future {
		t.Fatal("expiration shifted during replay")
	}
	if src.KeyType("expired") != "none" {
		t.Fatal("resurrected expired key")
	}
	if !src.Expiry("persistent").IsZero() {
		t.Fatal("persistent key got expiration")
	}
	if _, err := src.GetKey("legacy"); err != nil {
		t.Fatal("legacy replay failed", err)
	}
}
func TestTruncatedRecordReturnsError(t *testing.T) {
	if err := Replay(bytes.NewBufferString("*3\r\n$3\r\nSET\r\n$1\r\nk\r\n$5\r\nhi")); err == nil {
		t.Fatal("accepted truncated record")
	}
}

func TestTransactionReplayRequiresCompleteValidBatch(t *testing.T) {
	src.CommandMu.Lock()
	defer src.CommandMu.Unlock()
	validStart := helpers.Strings([]string{"MULTI"})
	write := helpers.Strings([]string{"SET", "tx-key", "new"})
	end := helpers.Strings([]string{"EXEC"})
	for name, suffix := range map[string][]byte{
		"missing-exec":   nil,
		"truncated-exec": []byte("*1\r\n$4\r\nEX"),
		"invalid-record": append(helpers.Strings([]string{"SET", "other", "v", "PXAT", "bad"}), end...),
		"nested-multi":   validStart,
	} {
		t.Run(name, func(t *testing.T) {
			src.SetValue("tx-key", "original", time.Time{})
			batch := append(append(append([]byte{}, validStart...), write...), suffix...)
			if err := Replay(bytes.NewReader(batch)); err == nil {
				t.Fatal("accepted incomplete or invalid transaction")
			}
			if value, _ := src.GetKey("tx-key"); value != "original" {
				t.Fatal("partially applied transaction", value)
			}
		})
	}
	batch := append(append(append([]byte{}, validStart...), write...), end...)
	if err := Replay(bytes.NewReader(batch)); err != nil {
		t.Fatal(err)
	}
	if value, _ := src.GetKey("tx-key"); value != "new" {
		t.Fatal(value)
	}
	if err := Replay(bytes.NewReader(end)); err == nil {
		t.Fatal("accepted EXEC without MULTI")
	}
}

func TestTransactionRecorderPreservesFinalStateAndExpiration(t *testing.T) {
	src.CommandMu.Lock()
	defer src.CommandMu.Unlock()
	AofChan = make(chan []byte, 4)
	recorder := NewTransactionRecorder()
	expiry := time.Now().Add(time.Hour).Truncate(time.Millisecond)
	src.SetValue("recorded", "first", expiry)
	recorder.String("recorded")
	src.SetValue("recorded", "last \r\n\x00", expiry)
	recorder.String("recorded")
	src.SetValue("deleted", "old", time.Time{})
	src.DeleteKey("deleted")
	recorder.Delete("deleted")
	recorder.Commit()
	if len(AofChan) != 1 {
		t.Fatal("expected one transaction batch")
	}
	batch := <-AofChan
	if bytes.Contains(batch, []byte("first")) {
		t.Fatal("recorded an intermediate value")
	}
	src.DeleteKey("recorded")
	src.SetValue("deleted", "restore-old", time.Time{})
	if err := Replay(bytes.NewReader(batch)); err != nil {
		t.Fatal(err)
	}
	if value, _ := src.GetKey("recorded"); value != "last \r\n\x00" {
		t.Fatal(value)
	}
	if !src.Expiry("recorded").Equal(expiry) {
		t.Fatal("expiration changed")
	}
	if src.KeyType("deleted") != "none" {
		t.Fatal("deletion not restored")
	}
}
