package src

import (
	"time"
)

func CheckTTL(msg []string, client *Client) (int64, error) {
	key := msg[1]
	KeyMu.RLock()
	entry, ok := ActiveKeys[key]
	var copy Entry
	if ok {
		copy = *entry
	}
	data := &copy
	KeyMu.RUnlock()

	if !ok {
		return -2, nil
	}

	if data.TTL.IsZero() {
		return -1, nil
	}

	if time.Now().After(data.TTL) {
		KeyMu.Lock()
		// recheck in case another goroutine updated same key for concurrency safety likely a edge case

		data, ok = ActiveKeys[key]
		if ok && !data.TTL.IsZero() && time.Now().After(data.TTL) {
			delete(ActiveKeys, key)
		}

		KeyMu.Unlock()
		return -2, nil
	}

	return int64(time.Until(data.TTL).Seconds()), nil
}
