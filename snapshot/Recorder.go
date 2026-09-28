package snapshot

import (
	"sort"
	"strconv"

	"github.com/rajandhamala/goRedis/helpers"
	"github.com/rajandhamala/goRedis/src"
)

// Recorder is used only while CommandMu is held. The zero value logs immediately.
// Transactions collect changed string keys until a stream operation needs an
// ordering boundary, then persist one MULTI/EXEC batch.
// Other collection types remain outside the prototype's AOF support.
type Recorder struct {
	changed map[string]struct{}
	batch   []byte
}

func NewTransactionRecorder() *Recorder { return &Recorder{changed: make(map[string]struct{})} }

func (r *Recorder) String(key string) {
	if r.changed != nil {
		r.changed[key] = struct{}{}
		return
	}
	AofChan <- stringRecord(key)
}

func (r *Recorder) Delete(keys ...string) {
	if r.changed != nil {
		for _, key := range keys {
			r.changed[key] = struct{}{}
		}
		return
	}
	AofChan <- helpers.Strings(append([]string{"DEL"}, keys...))
}

func (r *Recorder) Stream(command []string) {
	if r.changed != nil {
		r.flushChanged()
		r.batch = append(r.batch, helpers.Strings(command)...)
		return
	}
	AofChan <- helpers.Strings(command)
}

func (r *Recorder) flushChanged() {
	if len(r.changed) == 0 {
		return
	}
	keys := make([]string, 0, len(r.changed))
	for key := range r.changed {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		r.batch = append(r.batch, stringRecord(key)...)
	}
	clear(r.changed)
}

func (r *Recorder) Commit() {
	r.flushChanged()
	if len(r.batch) == 0 {
		return
	}
	batch := helpers.Strings([]string{"MULTI"})
	batch = append(batch, r.batch...)
	batch = append(batch, helpers.Strings([]string{"EXEC"})...)
	// One channel item keeps the worker from interleaving records from other commands.
	AofChan <- batch
	r.batch = nil
}

func stringRecord(key string) []byte {
	if src.KeyType(key) != "string" {
		return helpers.Strings([]string{"DEL", key})
	}
	value, err := src.GetKey(key)
	if err != nil {
		return helpers.Strings([]string{"DEL", key})
	}
	args := []string{"SET", key, value}
	if expiry := src.Expiry(key); !expiry.IsZero() {
		args = append(args, "PXAT", strconv.FormatInt(expiry.UnixMilli(), 10))
	}
	return helpers.Strings(args)
}
