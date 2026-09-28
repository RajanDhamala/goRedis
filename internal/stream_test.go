package internal

import (
	"bytes"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	r "github.com/rajandhamala/goRedis/helpers"
	"github.com/rajandhamala/goRedis/snapshot"
	"github.com/rajandhamala/goRedis/src"
)

func cleanupStreamKeys(t *testing.T, keys ...string) {
	t.Helper()
	t.Cleanup(func() {
		src.CommandMu.Lock()
		defer src.CommandMu.Unlock()
		for _, key := range keys {
			src.DeleteKey(key)
		}
	})
}

func TestStreamRESPWire(t *testing.T) {
	s := transactionServer(t)
	server, peer := net.Pipe()
	done := make(chan struct{})
	go func() {
		s.HandleConnection(server)
		close(done)
	}()
	defer peer.Close()
	if err := peer.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	key := "stream-wire-test"
	cleanupStreamKeys(t, key)
	commands := [][]string{
		{"AUTH", "transaction-test-password"},
		{"XADD", key, "1-0", "field", "value"},
		{"XLEN", key},
		{"XRANGE", key, "-", "+"},
		{"XREAD", "STREAMS", key, "0-0"},
		{"XTRIM", key, "MAXLEN", "0"},
		{"XLEN", key},
		{"QUIT"},
	}
	var input []byte
	for _, command := range commands {
		input = append(input, r.Strings(command)...)
	}
	if _, err := peer.Write(input); err != nil {
		t.Fatal(err)
	}
	entry := r.Array(r.Bulk("1-0"), r.Strings([]string{"field", "value"}))
	want := append([]byte{}, r.Simple("OK")...)
	want = append(want, r.Bulk("1-0")...)
	want = append(want, r.Integer(1)...)
	want = append(want, r.Array(entry)...)
	want = append(want, r.Array(r.Array(r.Bulk(key), r.Array(entry)))...)
	want = append(want, r.Integer(1)...)
	want = append(want, r.Integer(0)...)
	want = append(want, r.Simple("OK")...)
	got := make([]byte, len(want))
	if _, err := io.ReadFull(peer, got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("RESP stream wire reply = %q, want %q", got, want)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("connection did not close after QUIT")
	}
}

func TestStreamRESPAndKeyspace(t *testing.T) {
	s := transactionServer(t)
	c := transactionClient(t)
	key := "stream-resp-test"
	cleanupStreamKeys(t, key)
	transactExpect(t, s, c, ":0\r\n", "XLEN", key)
	transactExpect(t, s, c, "*0\r\n", "XRANGE", key, "-", "+")
	transactExpect(t, s, c, "$-1\r\n", "XREAD", "STREAMS", key, "0-0")
	transactExpect(t, s, c, "$3\r\n1-0\r\n", "XADD", key, "1-0", "a", "one", "b", "two")
	if len(snapshot.AofChan) != 1 {
		t.Fatal("successful XADD was not recorded")
	}
	firstRecord := <-snapshot.AofChan
	if !bytes.Equal(firstRecord, r.Strings([]string{"XADD", key, "1-0", "a", "one", "b", "two"})) {
		t.Fatalf("unexpected XADD AOF record: %q", firstRecord)
	}
	if got := transactCall(t, s, c, "XADD", key, "1-0", "a", "duplicate"); !strings.HasPrefix(got, "-ERR ") {
		t.Fatal(got)
	}
	if len(snapshot.AofChan) != 0 {
		t.Fatal("failed XADD was recorded")
	}
	transactExpect(t, s, c, ":1\r\n", "XLEN", key)
	transactExpect(t, s, c, "+stream\r\n", "TYPE", key)
	transactExpect(t, s, c, ":1\r\n", "EXISTS", key)
	transactExpect(t, s, c, "-WRONGTYPE Operation against a key holding the wrong kind of value\r\n", "GET", key)
	transactExpect(t, s, c, "$-1\r\n", "SET", key, "other", "NX")
	entry := r.Array(r.Bulk("1-0"), r.Strings([]string{"a", "one", "b", "two"}))
	transactExpect(t, s, c, string(r.Array(entry)), "XRANGE", key, "-", "+")
	transactExpect(t, s, c, string(r.Array(entry)), "XREVRANGE", key, "+", "-")
	transactExpect(t, s, c, string(r.Array(r.Array(r.Bulk(key), r.Array(entry)))), "XREAD", "STREAMS", key, "0-0")
	transactExpect(t, s, c, ":1\r\n", "XDEL", key, "1-0")
	if len(snapshot.AofChan) != 1 {
		t.Fatal("successful XDEL was not recorded")
	}
	<-snapshot.AofChan
	transactExpect(t, s, c, ":0\r\n", "XLEN", key)
	transactExpect(t, s, c, "+stream\r\n", "TYPE", key)
	transactExpect(t, s, c, ":0\r\n", "XDEL", key, "1-0")
	if len(snapshot.AofChan) != 0 {
		t.Fatal("no-op XDEL was recorded")
	}
	transactExpect(t, s, c, ":1\r\n", "DEL", key)
	transactExpect(t, s, c, ":0\r\n", "XLEN", key)
	transactExpect(t, s, c, "+none\r\n", "TYPE", key)
}

func TestStreamRangeTrimAndReplay(t *testing.T) {
	s := transactionServer(t)
	c := transactionClient(t)
	key := "stream-replay-test"
	cleanupStreamKeys(t, key)
	for _, id := range []string{"10-0", "10-1", "11-0"} {
		transactExpect(t, s, c, string(r.Bulk(id)), "XADD", key, id, "f", id)
	}
	transactExpect(t, s, c, ":1\r\n", "XTRIM", key, "MAXLEN", "2")
	transactExpect(t, s, c, ":2\r\n", "XLEN", key)
	first := r.Array(r.Bulk("10-1"), r.Strings([]string{"f", "10-1"}))
	last := r.Array(r.Bulk("11-0"), r.Strings([]string{"f", "11-0"}))
	transactExpect(t, s, c, string(r.Array(last, first)), "XREVRANGE", key, "+", "-", "COUNT", "2")
	transactExpect(t, s, c, string(r.Array(last)), "XRANGE", key, "(10-1", "+", "COUNT", "1")
	transactExpect(t, s, c, "$4\r\n11-1\r\n", "XADD", key, "11-*", "f", "new")
	transactExpect(t, s, c, ":1\r\n", "XDEL", key, "10-1")
	var log []byte
	for len(snapshot.AofChan) > 0 {
		log = append(log, (<-snapshot.AofChan)...)
	}
	src.CommandMu.Lock()
	src.DeleteKey(key)
	err := snapshot.Replay(bytes.NewReader(log))
	src.CommandMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	transactExpect(t, s, c, ":2\r\n", "XLEN", key)
	transactExpect(t, s, c, "$4\r\n11-2\r\n", "XADD", key, "11-*", "f", "after-replay")
}

func TestStreamTransactionAndBlockingRead(t *testing.T) {
	s := transactionServer(t)
	c := transactionClient(t)
	reader := transactionClient(t)
	key := "stream-transaction-test"
	cleanupStreamKeys(t, key)
	transactExpect(t, s, c, "+OK\r\n", "SET", key, "old")
	<-snapshot.AofChan
	transactExpect(t, s, c, "+OK\r\n", "MULTI")
	for _, command := range [][]string{
		{"DEL", key},
		{"XADD", key, "1-0", "f", "first"},
		{"DEL", key},
		{"XADD", key, "2-0", "f", "second"},
	} {
		transactExpect(t, s, c, "+QUEUED\r\n", command...)
	}
	if len(snapshot.AofChan) != 0 {
		t.Fatal("queued commands reached AOF")
	}
	if got := transactCall(t, s, c, "EXEC"); !strings.HasPrefix(got, "*4\r\n") {
		t.Fatal(got)
	}
	if len(snapshot.AofChan) != 1 {
		t.Fatal("transaction did not produce one batch")
	}
	batch := <-snapshot.AofChan
	src.CommandMu.Lock()
	src.SetValue(key, "old", time.Time{})
	err := snapshot.Replay(bytes.NewReader(batch))
	src.CommandMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	transactExpect(t, s, c, "+stream\r\n", "TYPE", key)
	transactExpect(t, s, c, ":1\r\n", "XLEN", key)
	transactExpect(t, s, c, "$-1\r\n", "XREAD", "STREAMS", key, "$")

	done := make(chan struct{})
	go func() {
		s.HandleMethods([]string{"XREAD", "BLOCK", "1000", "STREAMS", key, "$"}, reader)
		close(done)
	}()
	time.Sleep(10 * time.Millisecond)
	transactExpect(t, s, c, "$3\r\n3-0\r\n", "XADD", key, "3-0", "f", "wake")
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("blocking XREAD did not wake")
	}
	select {
	case reply := <-reader.Send:
		entry := r.Array(r.Bulk("3-0"), r.Strings([]string{"f", "wake"}))
		want := r.Array(r.Array(r.Bulk(key), r.Array(entry)))
		if !bytes.Equal(reply, want) {
			t.Fatalf("blocking XREAD = %q, want %q", reply, want)
		}
	default:
		t.Fatal("blocking XREAD returned without a reply")
	}
}

func TestStreamOptionsAndNoopLogging(t *testing.T) {
	s := transactionServer(t)
	c := transactionClient(t)
	key := "stream-options-test"
	cleanupStreamKeys(t, key)
	transactExpect(t, s, c, "$-1\r\n", "XADD", key, "NOMKSTREAM", "*", "f", "v")
	if len(snapshot.AofChan) != 0 {
		t.Fatal("NOMKSTREAM no-op reached AOF")
	}
	transactExpect(t, s, c, "$4\r\n10-0\r\n", "XADD", key, "10-0", "f", "first")
	transactExpect(t, s, c, "$4\r\n11-0\r\n", "XADD", key, "MAXLEN", "~", "1", "11-0", "f", "second")
	transactExpect(t, s, c, ":1\r\n", "XLEN", key)
	transactExpect(t, s, c, ":0\r\n", "XTRIM", key, "MINID", "11-0")
	if got := transactCall(t, s, c, "XTRIM", key, "MAXLEN", "invalid"); !strings.HasPrefix(got, "-ERR ") {
		t.Fatal(got)
	}
	if len(snapshot.AofChan) != 2 {
		t.Fatal("only two successful XADD calls should be logged")
	}
	var log []byte
	for len(snapshot.AofChan) > 0 {
		log = append(log, (<-snapshot.AofChan)...)
	}
	src.CommandMu.Lock()
	src.DeleteKey(key)
	err := snapshot.Replay(bytes.NewReader(log))
	src.CommandMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	transactExpect(t, s, c, ":1\r\n", "XLEN", key)
	transactExpect(t, s, c, "$4\r\n11-1\r\n", "XADD", key, "11-*", "f", "after-replay")
	transactExpect(t, s, c, "$-1\r\n", "XREAD", "BLOCK", "15", "STREAMS", key, "$")
}

func TestStreamExpirationSurvivesReplay(t *testing.T) {
	s := transactionServer(t)
	c := transactionClient(t)
	key := "stream-expiry-test"
	cleanupStreamKeys(t, key)
	transactExpect(t, s, c, "$3\r\n1-0\r\n", "XADD", key, "1-0", "f", "v")
	transactExpect(t, s, c, ":1\r\n", "PEXPIRE", key, "60000")
	if len(snapshot.AofChan) != 2 {
		t.Fatal("stream write and expiration were not both recorded")
	}
	log := append([]byte{}, (<-snapshot.AofChan)...)
	log = append(log, (<-snapshot.AofChan)...)
	src.CommandMu.Lock()
	src.DeleteKey(key)
	err := snapshot.Replay(bytes.NewReader(log))
	src.CommandMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	transactExpect(t, s, c, "+stream\r\n", "TYPE", key)
	if got := transactCall(t, s, c, "PTTL", key); !strings.HasPrefix(got, ":") || got == ":-1\r\n" || got == ":-2\r\n" {
		t.Fatal("stream expiration was not restored", got)
	}
	transactExpect(t, s, c, ":1\r\n", "PEXPIRE", key, "0")
	if len(snapshot.AofChan) != 1 {
		t.Fatal("successful stream expiration removal was not logged")
	}
	transactExpect(t, s, c, "+none\r\n", "TYPE", key)
}

func TestMultiStreamReadAndMinID(t *testing.T) {
	s := transactionServer(t)
	c := transactionClient(t)
	left, right := "stream-read-left", "stream-read-right"
	cleanupStreamKeys(t, left, right)
	for _, command := range [][]string{
		{"XADD", left, "10-0", "f", "a"},
		{"XADD", left, "11-0", "f", "b"},
		{"XADD", right, "20-0", "f", "c"},
	} {
		if got := transactCall(t, s, c, command...); !strings.HasPrefix(got, "$") {
			t.Fatal(got)
		}
	}
	leftFirst := r.Array(r.Bulk("10-0"), r.Strings([]string{"f", "a"}))
	leftLast := r.Array(r.Bulk("11-0"), r.Strings([]string{"f", "b"}))
	rightFirst := r.Array(r.Bulk("20-0"), r.Strings([]string{"f", "c"}))
	want := r.Array(r.Array(r.Bulk(left), r.Array(leftFirst)), r.Array(r.Bulk(right), r.Array(rightFirst)))
	transactExpect(t, s, c, string(want), "XREAD", "COUNT", "1", "STREAMS", left, right, "0-0", "0-0")
	transactExpect(t, s, c, string(r.Array(r.Array(r.Bulk(left), r.Array(leftLast)))), "XREAD", "STREAMS", left, "+")
	transactExpect(t, s, c, ":1\r\n", "XTRIM", left, "MINID", "11-0")
	transactExpect(t, s, c, string(r.Array(leftLast)), "XRANGE", left, "-", "+")
}
