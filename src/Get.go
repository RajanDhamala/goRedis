package src

import (
	"errors"
	"time"
)

func GetKey(key string) (string, error) {
	KeyMu.RLock()
	resp, ok := ActiveKeys[key]
	KeyMu.RUnlock()

	if !ok {
		return "", errors.New("key not found")
	}

	if !resp.TTL.IsZero() && time.Now().After(resp.TTL) {
		return "", errors.New("key expired")
	}

	return resp.Value, nil
}
