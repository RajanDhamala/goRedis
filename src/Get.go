package src

import (
	"errors"
	"time"
)

func GetKey(key string) (string, error) {
	Mu.RLock()
	resp, ok := ActiveKeys[key]
	Mu.RUnlock()

	if !ok {
		return "", errors.New("key not found")
	}

	if !resp.TTL.IsZero() && time.Now().After(resp.TTL) {
		return "", errors.New("key expired")
	}

	return resp.Value, nil
}
