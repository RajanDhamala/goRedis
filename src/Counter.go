package src

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"time"
)

func Incr(msg []string) (int64, error) { return changeCounter(msg[1], 1, false) }
func Decr(msg []string) (int64, error) { return changeCounter(msg[1], 1, true) }

func CounterBy(msg []string) (int64, error) {
	delta, err := strconv.ParseInt(msg[2], 10, 64)
	if err != nil {
		return 0, errors.New("value is not an integer or out of range")
	}
	return changeCounter(msg[1], delta, strings.EqualFold(msg[0], "DECRBY"))
}

func changeCounter(key string, delta int64, subtract bool) (int64, error) {
	KeyMu.Lock()
	defer KeyMu.Unlock()
	entry, ok := ActiveKeys[key]
	if ok && !entry.TTL.IsZero() && !time.Now().Before(entry.TTL) {
		delete(ActiveKeys, key)
		ok = false
	}
	if !ok {
		entry = &Entry{Value: "0"}
	}
	value, err := strconv.ParseInt(entry.Value, 10, 64)
	if err != nil {
		return 0, errors.New("value is not an integer or out of range")
	}
	if subtract {
		if (delta > 0 && value < math.MinInt64+delta) || (delta < 0 && value > math.MaxInt64+delta) {
			return 0, errors.New("increment or decrement would overflow")
		}
		value -= delta
	} else {
		if (delta > 0 && value > math.MaxInt64-delta) || (delta < 0 && value < math.MinInt64-delta) {
			return 0, errors.New("increment or decrement would overflow")
		}
		value += delta
	}

	entry.Value = strconv.FormatInt(value, 10)
	ActiveKeys[key] = entry
	return value, nil
}
