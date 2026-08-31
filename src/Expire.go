package src

import (
	"fmt"
	"strconv"
	"time"
)

func ExpireKey(msg []string, client *Client) (string, error) {
	key := msg[1]
	seconds, err := strconv.Atoi(msg[2])
	if err != nil {
		return "invalid ttl", err
	}

	expiry := time.Now().Add(time.Duration(seconds) * time.Second)

	KeyMu.Lock()
	defer KeyMu.Unlock()

	result, ok := ActiveKeys[key]
	if !ok {
		return "key not found", nil
	}
	if !result.TTL.IsZero() && time.Now().After(result.TTL) {
		fmt.Println("key has been expired")
		delete(ActiveKeys, key)
	}

	result.TTL = expiry
	return "", nil
}
