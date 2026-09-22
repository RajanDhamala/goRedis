package snapshot

import (
	"sort"
	"strconv"

	"github.com/rajandhamala/goRedis/helpers"
	"github.com/rajandhamala/goRedis/src"
)

// Recorder is used only while CommandMu is held. The zero value logs immediately.
// Transactions collect changed keys, then persist their final string state once.
// Collection persistence remains outside the prototype's current AOF support.
type Recorder struct {
	changed map[string]struct{}
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

func (r *Recorder) Commit() {
	if len(r.changed) == 0 {
		return
	}
	keys := make([]string, 0, len(r.changed))
	for key := range r.changed {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	batch := helpers.Strings([]string{"MULTI"})
	for _, key := range keys {
		batch = append(batch, stringRecord(key)...)
	}
	batch = append(batch, helpers.Strings([]string{"EXEC"})...)
	// One channel item keeps the worker from interleaving records from other commands.
	AofChan <- batch
	clear(r.changed)
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
