package snapshot

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rajandhamala/goRedis/helpers"
	"github.com/rajandhamala/goRedis/src"
)

func PlayAofShapshot() {
	file, err := os.Open("appendonly.aof")
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		fmt.Println("error opening AOF:", err)
		return
	}
	defer file.Close()
	if err := Replay(file); err != nil {
		fmt.Println("error replaying AOF:", err)
	}
}

// Read both new binary-safe RESP records and the prototype's legacy inline records.
func Replay(input io.Reader) error {
	reader := bufio.NewReader(input)
	for {
		msg, err := helpers.ReadCommand(reader)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := replayCommand(msg); err != nil {
			return err
		}
	}
}

func ExecAofLog(msg []string) {
	if err := replayCommand(msg); err != nil {
		fmt.Println("error replaying AOF command:", err)
	}
}

func replayCommand(msg []string) error {
	if len(msg) == 0 {
		return nil
	}
	switch strings.ToUpper(msg[0]) {
	case "SET":
		if len(msg) == 4 { // Legacy: SET key value relative-ttl
			_, err := src.AddKey(msg[1], msg[2], msg[3])
			return err
		}
		var expiry time.Time
		if len(msg) == 5 && strings.EqualFold(msg[3], "PXAT") {
			millis, err := strconv.ParseInt(msg[4], 10, 64)
			if err != nil {
				return err
			}
			expiry = time.UnixMilli(millis)
		} else if len(msg) != 3 {
			return fmt.Errorf("invalid SET entry")
		}
		src.SetValue(msg[1], msg[2], expiry)
		if !expiry.IsZero() && !time.Now().Before(expiry) {
			src.DeleteKey(msg[1])
		}
	case "DEL":
		if len(msg) < 2 {
			return fmt.Errorf("invalid DEL entry")
		}
		for _, key := range msg[1:] {
			src.DeleteKey(key)
		}
	case "GET": // Old development logs may contain reads.
	default:
		return fmt.Errorf("unsupported AOF command %q", msg[0])
	}
	return nil
}

// Call while holding src.CommandMu so mutation and log order agree.
// The prototype persists string values; collections remain in-memory only.
func RecordString(key string) {
	if src.KeyType(key) != "string" {
		AofChan <- helpers.Strings([]string{"DEL", key})
		return
	}
	value, err := src.GetKey(key)
	if err != nil {
		AofChan <- helpers.Strings([]string{"DEL", key})
		return
	}
	args := []string{"SET", key, value}
	if expiry := src.Expiry(key); !expiry.IsZero() {
		args = append(args, "PXAT", strconv.FormatInt(expiry.UnixMilli(), 10))
	}
	AofChan <- helpers.Strings(args)
}
