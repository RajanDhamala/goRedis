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
