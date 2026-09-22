package internal

import (
	"strings"

	r "github.com/rajandhamala/goRedis/helpers"
	"github.com/rajandhamala/goRedis/snapshot"
	"github.com/rajandhamala/goRedis/src"
)

const maxTransactionCommands = 1024
const maxTransactionBytes = 64 * 1024 * 1024

// HandleMethods holds CommandMu for this whole call, including every EXEC command.
func (s *Server) dispatchCommand(msg []string, client *src.Client) []byte {
	if reply := validateCommand(msg, client); reply != nil {
		failTransaction(client)
		return reply
	}
	method := strings.ToUpper(msg[0])
	switch method {
	case "MULTI":
		if client.Transaction != nil {
			return r.Error("ERR MULTI calls can not be nested")
		}
		client.Transaction = &src.Transaction{}
		return r.Simple("OK")
	case "DISCARD":
		if client.Transaction == nil {
			return r.Error("ERR DISCARD without MULTI")
		}
		client.Transaction = nil
		return r.Simple("OK")
	case "EXEC":
		tx := client.Transaction
		if tx == nil {
			return r.Error("ERR EXEC without MULTI")
		}
		client.Transaction = nil
		if tx.Failed {
			return r.Error("EXECABORT Transaction discarded because of previous errors.")
		}
		journal := snapshot.NewTransactionRecorder()
		replies := make([][]byte, 0, len(tx.Commands))
		for _, command := range tx.Commands {
			replies = append(replies, s.executeCommand(command, client, journal))
		}
		journal.Commit()
		return r.Array(replies...)
	case "QUIT":
		client.Transaction = nil
		return r.Simple("OK")
	}
	if tx := client.Transaction; tx != nil {
		// Subscription commands emit pushes directly instead of one ordinary reply.
		// Keep them outside transactions until transactional subscription handling exists.
		if method == "SUBSCRIBE" || method == "UNSUBSCRIBE" {
			failTransaction(client)
			return r.Error("ERR subscription commands are not supported inside MULTI")
		}
		size := len(msg) * 16 // Include string-header overhead as well as argument bytes.
		for _, arg := range msg {
			size += len(arg)
		}
		if len(tx.Commands) >= maxTransactionCommands || size > maxTransactionBytes-tx.Bytes {
			failTransaction(client)
			return r.Error("ERR transaction queue limit exceeded")
		}
		if !tx.Failed {
			tx.Commands = append(tx.Commands, append([]string(nil), msg...))
			tx.Bytes += size
		}
		return r.Simple("QUEUED")
	}
	return s.executeCommand(msg, client, &snapshot.Recorder{})
}

func failTransaction(client *src.Client) {
	if tx := client.Transaction; tx != nil {
		tx.Failed = true
		tx.Commands = nil
		tx.Bytes = 0
	}
}
