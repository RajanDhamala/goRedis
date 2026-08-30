package src

import (
	// "fmt"
	"errors"
	"time"
)

func CheckKeyExistance(msg []string, client *Client) (bool, error) {
	key := msg[1]
	// no validation during testing
	KeyMu.RLock()
	resp, ok := ActiveKeys[key]
	KeyMu.RUnlock()

	if !ok {
		return false, errors.New("key not found")
	}

	if !resp.TTL.IsZero() && time.Now().After(resp.TTL) {
		return false, errors.New("key expired")
	}

	return true, nil
}
