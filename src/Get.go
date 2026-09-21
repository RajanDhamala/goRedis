package src

import (
	"errors"
	"time"
)

func GetKey(key string) (string, error) {
	KeyMu.RLock()
	defer KeyMu.RUnlock()
	resp, ok := ActiveKeys[key]

	if !ok {
		return "", errors.New("key not found")
	}

	if !resp.TTL.IsZero() && time.Now().After(resp.TTL) {
		return "", errors.New("key expired")
	}

	return resp.Value, nil
}
