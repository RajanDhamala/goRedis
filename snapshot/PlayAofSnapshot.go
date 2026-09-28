package snapshot

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/rajandhamala/goRedis/helpers"
	"github.com/rajandhamala/goRedis/src"
)

var ErrIncompleteTransaction = errors.New("incomplete AOF transaction: missing EXEC; repair or restore the AOF before restarting")

func PlayAofShapshot() error {
	file, err := os.Open("appendonly.aof")
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("opening AOF: %w", err)
	}
	defer file.Close()
	if err := Replay(file); err != nil {
		return fmt.Errorf("replaying AOF: %w", err)
	}
	return nil
}

type aofOperation struct {
	key        string
	value      string
	expiry     time.Time
	remove     bool
	stream     []string
	expireOnly bool
}

// Read both binary-safe RESP and legacy inline records. Transaction operations
// are decoded and validated but never applied until a complete EXEC is present.
func Replay(input io.Reader) error {
	reader := bufio.NewReader(input)
	var pending []aofOperation
	inTransaction := false
	for {
		msg, err := helpers.ReadCommand(reader)
		if err == io.EOF {
			if inTransaction {
				return ErrIncompleteTransaction
			}
			return nil
		}
		if err != nil {
			return fmt.Errorf("invalid AOF record: %w", err)
		}
		if len(msg) == 0 {
			continue
		}
		switch strings.ToUpper(msg[0]) {
		case "MULTI":
			if len(msg) != 1 || inTransaction {
				return errors.New("invalid or nested MULTI in AOF")
			}
			inTransaction = true
			continue
		case "EXEC":
			if len(msg) != 1 || !inTransaction {
				return errors.New("invalid EXEC in AOF")
			}
			if err := validateOperations(pending); err != nil {
				return err
			}
			if err := applyOperations(pending); err != nil {
				return err
			}
			pending = nil
			inTransaction = false
			continue
		}
		operations, err := decodeRecord(msg)
		if err != nil {
			return err
		}
		if inTransaction {
			pending = append(pending, operations...)
		} else {
			if err := applyOperations(operations); err != nil {
				return err
			}
		}
	}
}

type replayKeyState struct {
	kind   string
	lastID src.StreamID
}

// Preflight state-dependent stream errors before applying any AOF transaction.
func validateOperations(operations []aofOperation) error {
	states := make(map[string]replayKeyState)
	stateFor := func(key string) replayKeyState {
		if state, ok := states[key]; ok {
			return state
		}
		state := replayKeyState{kind: src.KeyType(key)}
		if state.kind == "stream" {
			state.lastID, _ = src.StreamLastID(key)
		}
		return state
	}
	for _, op := range operations {
		if op.expireOnly {
			state := stateFor(op.key)
			if !time.Now().Before(op.expiry) {
				state = replayKeyState{kind: "none"}
			}
			states[op.key] = state
			continue
		}
		if len(op.stream) != 0 {
			key := op.stream[1]
			state := stateFor(key)
			switch strings.ToUpper(op.stream[0]) {
			case "XADD":
				if state.kind != "none" && state.kind != "stream" {
					return fmt.Errorf("invalid XADD type in AOF transaction for %q", key)
				}
				id, err := src.StreamRecordID(op.stream)
				if err != nil || (state.kind == "stream" && (id.Milliseconds < state.lastID.Milliseconds || (id.Milliseconds == state.lastID.Milliseconds && id.Sequence <= state.lastID.Sequence))) {
					return fmt.Errorf("invalid XADD ID order in AOF transaction for %q", key)
				}
				state.kind, state.lastID = "stream", id
			case "XDEL", "XTRIM":
				if state.kind != "stream" {
					return fmt.Errorf("invalid %s type in AOF transaction for %q", op.stream[0], key)
				}
			}
			states[key] = state
			continue
		}
		if op.remove || (!op.expiry.IsZero() && !time.Now().Before(op.expiry)) {
			states[op.key] = replayKeyState{kind: "none"}
		} else {
			states[op.key] = replayKeyState{kind: "string"}
		}
	}
	return nil
}

func ExecAofLog(msg []string) {
	operations, err := decodeRecord(msg)
	if err != nil {
		fmt.Println("error replaying AOF command:", err)
		return
	}
	if err := applyOperations(operations); err != nil {
		fmt.Println("error applying AOF command:", err)
	}
}

func decodeRecord(msg []string) ([]aofOperation, error) {
	if len(msg) == 0 {
		return nil, nil
	}
	switch strings.ToUpper(msg[0]) {
	case "SET":
		var expiry time.Time
		switch {
		case len(msg) == 4: // Legacy: SET key value relative-ttl.
			seconds, err := strconv.ParseInt(msg[3], 10, 64)
			if err != nil || seconds > int64((1<<63-1)/time.Second) || seconds < int64((-1<<63)/time.Second) {
				return nil, errors.New("invalid legacy SET expiration in AOF")
			}
			expiry = time.Now().Add(time.Duration(seconds) * time.Second)
		case len(msg) == 5 && strings.EqualFold(msg[3], "PXAT"):
			millis, err := strconv.ParseInt(msg[4], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid SET expiration in AOF: %w", err)
			}
			expiry = time.UnixMilli(millis)
		case len(msg) == 3:
		default:
			return nil, errors.New("invalid SET entry in AOF")
		}
		return []aofOperation{{key: msg[1], value: msg[2], expiry: expiry}}, nil
	case "DEL":
		if len(msg) < 2 {
			return nil, errors.New("invalid DEL entry in AOF")
		}
		operations := make([]aofOperation, 0, len(msg)-1)
		for _, key := range msg[1:] {
			operations = append(operations, aofOperation{key: key, remove: true})
		}
		return operations, nil
	case "GET":
		return nil, nil // Old development logs may contain reads.
	case "PEXPIREAT":
		if len(msg) != 3 {
			return nil, errors.New("invalid PEXPIREAT entry in AOF")
		}
		millis, err := strconv.ParseInt(msg[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid PEXPIREAT entry in AOF: %w", err)
		}
		return []aofOperation{{key: msg[1], expiry: time.UnixMilli(millis), expireOnly: true}}, nil
	case "XADD", "XDEL", "XTRIM":
		if err := src.ValidateStreamRecord(msg); err != nil {
			return nil, fmt.Errorf("invalid %s entry in AOF: %w", msg[0], err)
		}
		return []aofOperation{{stream: slices.Clone(msg)}}, nil
	default:
		return nil, fmt.Errorf("unsupported AOF command %q", msg[0])
	}
}

func applyOperations(operations []aofOperation) error {
	for _, op := range operations {
		if op.expireOnly {
			src.ExpireAt(op.key, op.expiry)
			continue
		}
		if len(op.stream) != 0 {
			var err error
			switch strings.ToUpper(op.stream[0]) {
			case "XADD":
				_, _, err = src.XADD(op.stream)
			case "XDEL":
				_, err = src.XDEL(op.stream)
			case "XTRIM":
				_, err = src.XTRIM(op.stream)
			}
			if err != nil {
				return fmt.Errorf("applying %s from AOF: %w", op.stream[0], err)
			}
			continue
		}
		if op.remove || (!op.expiry.IsZero() && !time.Now().Before(op.expiry)) {
			src.DeleteKey(op.key)
		} else {
			src.SetValue(op.key, op.value, op.expiry)
		}
	}
	return nil
}
