package internal

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	r "github.com/rajandhamala/goRedis/helpers"
	"github.com/rajandhamala/goRedis/snapshot"
	"github.com/rajandhamala/goRedis/src"
)

func transactionServer(t *testing.T) *Server {
	t.Helper()
	src.CommandMu.Lock()
	clear(src.ActiveKeys)
	clear(src.GlobalHash)
	clear(src.GobalSet)
	clear(src.GlobalList)
	clear(src.GlobalZset)
	snapshot.AofChan = make(chan []byte, 4096)
	src.CommandMu.Unlock()
	server, err := NewServer("transaction-test-password")
	if err != nil {
		t.Fatal(err)
	}
	return server
}
func transactionClient(t *testing.T) *src.Client {
	t.Helper()
	client := src.NewClient(nil)
	client.Authenticated = true
	t.Cleanup(client.Disconnect)
	return client
}
func transactCall(t *testing.T, s *Server, c *src.Client, args ...string) string {
	t.Helper()
	s.HandleMethods(args, c)
	select {
	case reply := <-c.Send:
		return string(reply)
	case <-time.After(time.Second):
		t.Fatal("missing reply")
		return ""
	}
}
func transactExpect(t *testing.T, s *Server, c *src.Client, want string, args ...string) {
	t.Helper()
	if got := transactCall(t, s, c, args...); got != want {
		t.Fatalf("%v got %q want %q", args, got, want)
	}
}

const execAbort = "-EXECABORT Transaction discarded because of previous errors.\r\n"

func TestTransactionQueuesAndExecutesOnce(t *testing.T) {
	s := transactionServer(t)
	c := transactionClient(t)
	other := transactionClient(t)
	transactExpect(t, s, c, "+OK\r\n", "MULTI")
	transactExpect(t, s, c, "+QUEUED\r\n", "SET", "n", "4")
	transactExpect(t, s, c, "+QUEUED\r\n", "INCR", "n")
	transactExpect(t, s, c, "+QUEUED\r\n", "GET", "n")
	transactExpect(t, s, other, "$-1\r\n", "GET", "n")
	if len(snapshot.AofChan) != 0 {
		t.Fatal("queued writes reached AOF")
	}
	transactExpect(t, s, c, "*3\r\n+OK\r\n:5\r\n$1\r\n5\r\n", "EXEC")
	transactExpect(t, s, other, "$1\r\n5\r\n", "GET", "n")
	transactExpect(t, s, c, "-ERR EXEC without MULTI\r\n", "EXEC")
	if len(snapshot.AofChan) != 1 {
		t.Fatal("transaction was not one AOF batch")
	}
	batch := <-snapshot.AofChan
	if !bytes.HasPrefix(batch, r.Strings([]string{"MULTI"})) || !bytes.HasSuffix(batch, r.Strings([]string{"EXEC"})) {
		t.Fatalf("unframed AOF batch %q", batch)
	}
	src.CommandMu.Lock()
	src.DeleteKey("n")
	err := snapshot.Replay(bytes.NewReader(batch))
	src.CommandMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	transactExpect(t, s, c, "$1\r\n5\r\n", "GET", "n")
}

func TestTransactionDiscardEmptyAndNested(t *testing.T) {
	s := transactionServer(t)
	c := transactionClient(t)
	transactExpect(t, s, c, "-ERR DISCARD without MULTI\r\n", "DISCARD")
	transactExpect(t, s, c, "+OK\r\n", "MULTI")
	transactExpect(t, s, c, "-ERR MULTI calls can not be nested\r\n", "MULTI")
	transactExpect(t, s, c, "+QUEUED\r\n", "SET", "k", "v")
	transactExpect(t, s, c, "+OK\r\n", "DISCARD")
	transactExpect(t, s, c, "$-1\r\n", "GET", "k")
	transactExpect(t, s, c, "+OK\r\n", "MULTI")
	transactExpect(t, s, c, "*0\r\n", "EXEC")
	if len(snapshot.AofChan) != 0 {
		t.Fatal("discard or empty transaction persisted")
	}
}

func TestTransactionQueueErrorsAbortEverything(t *testing.T) {
	for _, bad := range [][]string{{"GET"}, {"unknown"}, {"HSET", "h", "f"}, {"SUBSCRIBE", "channel"}, {"UNSUBSCRIBE"}, {"EXEC", "extra"}} {
		t.Run(strings.Join(bad, "-"), func(t *testing.T) {
			s := transactionServer(t)
			c := transactionClient(t)
			transactExpect(t, s, c, "+OK\r\n", "MULTI")
			transactExpect(t, s, c, "+QUEUED\r\n", "SET", "k", "v")
			if got := transactCall(t, s, c, bad...); !strings.HasPrefix(got, "-ERR ") {
				t.Fatal(got)
			}
			transactExpect(t, s, c, "+QUEUED\r\n", "SET", "after-error", "v")
			transactExpect(t, s, c, execAbort, "EXEC")
			transactExpect(t, s, c, ":0\r\n", "EXISTS", "k", "after-error")
			transactExpect(t, s, c, "+PONG\r\n", "PING")
			if len(snapshot.AofChan) != 0 {
				t.Fatal("aborted writes persisted")
			}
		})
	}
}

func TestTransactionExecutionErrorsContinueWithoutRollback(t *testing.T) {
	s := transactionServer(t)
	c := transactionClient(t)
	transactExpect(t, s, c, "+OK\r\n", "MULTI")
	for _, cmd := range [][]string{{"SET", "k", "text"}, {"INCR", "k"}, {"LPUSH", "k", "item"}, {"SET", "bad-expiry", "v", "EX", "oops"}, {"SET", "after", "yes"}, {"GET", "after"}} {
		transactExpect(t, s, c, "+QUEUED\r\n", cmd...)
	}
	want := "*6\r\n+OK\r\n-ERR value is not an integer or out of range\r\n-WRONGTYPE Operation against a key holding the wrong kind of value\r\n-ERR value is not an integer or out of range\r\n+OK\r\n$3\r\nyes\r\n"
	transactExpect(t, s, c, want, "EXEC")
	transactExpect(t, s, c, "$4\r\ntext\r\n", "GET", "k")
	transactExpect(t, s, c, "$-1\r\n", "GET", "bad-expiry")
	batch := <-snapshot.AofChan
	src.CommandMu.Lock()
	src.DeleteKey("k")
	src.DeleteKey("after")
	err := snapshot.Replay(bytes.NewReader(batch))
	src.CommandMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	transactExpect(t, s, c, "$3\r\nyes\r\n", "GET", "after")
}

func TestTransactionTypesCheckedAtExecution(t *testing.T) {
	s := transactionServer(t)
	c := transactionClient(t)
	other := transactionClient(t)
	transactExpect(t, s, other, "+OK\r\n", "SET", "h", "old-string")
	transactExpect(t, s, c, "+OK\r\n", "MULTI")
	for _, cmd := range [][]string{{"DEL", "h"}, {"HSET", "h", "f", "value"}, {"HGETALL", "h"}} {
		transactExpect(t, s, c, "+QUEUED\r\n", cmd...)
	}
	transactExpect(t, s, c, "*3\r\n:1\r\n:1\r\n*2\r\n$1\r\nf\r\n$5\r\nvalue\r\n", "EXEC")
}

func TestTransactionCannotInterleaveWithOtherClients(t *testing.T) {
	s := transactionServer(t)
	c := transactionClient(t)
	other := transactionClient(t)
	transactExpect(t, s, c, "+OK\r\n", "MULTI")
	transactExpect(t, s, c, "+QUEUED\r\n", "SET", "n", "0")
	replies := [][]byte{r.Simple("OK")}
	for i := 1; i <= 100; i++ {
		transactExpect(t, s, c, "+QUEUED\r\n", "INCR", "n")
		replies = append(replies, r.Integer(int64(i)))
	}
	transactExpect(t, s, c, "+QUEUED\r\n", "GET", "n")
	replies = append(replies, r.Bulk("100"))
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 100; i++ {
			s.HandleMethods([]string{"SET", "n", "999"}, other)
			<-other.Send
		}
	}()
	close(start)
	got := transactCall(t, s, c, "EXEC")
	wg.Wait()
	if got != string(r.Array(replies...)) {
		t.Fatal("another client interleaved with EXEC", got)
	}
}

func TestTransactionLimitsAndAuthentication(t *testing.T) {
	s := transactionServer(t)
	c := transactionClient(t)
	c.Authenticated = false
	transactExpect(t, s, c, "-NOAUTH Authentication required.\r\n", "MULTI")
	transactExpect(t, s, c, "+OK\r\n", "AUTH", "transaction-test-password")
	transactExpect(t, s, c, "+OK\r\n", "MULTI")
	for i := 0; i < maxTransactionCommands; i++ {
		transactExpect(t, s, c, "+QUEUED\r\n", "PING")
	}
	transactExpect(t, s, c, "-ERR transaction queue limit exceeded\r\n", "PING")
	transactExpect(t, s, c, execAbort, "EXEC")
	transactExpect(t, s, c, "+OK\r\n", "MULTI")
	c.Transaction.Bytes = maxTransactionBytes - 1
	transactExpect(t, s, c, "-ERR transaction queue limit exceeded\r\n", "SET", "k", "v")
	transactExpect(t, s, c, "+OK\r\n", "DISCARD")
	if len(snapshot.AofChan) != 0 {
		t.Fatal("limited transaction wrote data")
	}
}

func TestDisconnectDiscardsTransaction(t *testing.T) {
	s := transactionServer(t)
	for _, quit := range []bool{false, true} {
		t.Run(fmt.Sprint(quit), func(t *testing.T) {
			server, peer := net.Pipe()
			done := make(chan struct{})
			go func() { defer close(done); s.HandleConnection(server) }()
			defer peer.Close()
			peer.SetDeadline(time.Now().Add(2 * time.Second))
			reader := bufio.NewReader(peer)
			for _, cmd := range [][]string{{"AUTH", "transaction-test-password"}, {"MULTI"}, {"SET", "disconnected", "v"}} {
				if _, err := peer.Write(r.Strings(cmd)); err != nil {
					t.Fatal(err)
				}
				if _, err := reader.ReadString('\n'); err != nil {
					t.Fatal(err)
				}
			}
			if quit {
				peer.Write(r.Strings([]string{"QUIT"}))
				if line, err := reader.ReadString('\n'); err != nil || line != "+OK\r\n" {
					t.Fatal(line, err)
				}
				if _, err := reader.ReadByte(); err != io.EOF {
					t.Fatal(err)
				}
			} else {
				peer.Close()
			}
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("connection did not exit")
			}
			other := transactionClient(t)
			transactExpect(t, s, other, "$-1\r\n", "GET", "disconnected")
			if len(snapshot.AofChan) != 0 {
				t.Fatal("disconnected transaction persisted")
			}
		})
	}
}
