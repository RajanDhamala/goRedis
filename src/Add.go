package src

import (
	"errors"
	"strconv"
	"time"
)

func AddKey(key string, value string, ttl string) (string, error) {
	data := Entry{
		Value: value,
	}

	ttlok, err := strconv.Atoi(ttl)
	if err != nil {
		return "", errors.New("invalid ttl")
	}
	data.TTL = time.Now().Add(
		time.Duration(ttlok) * time.Second,
	)

	Mu.Lock()
	ActiveKeys[key] = &data
	Mu.Unlock()

	return "key set successfully", nil
}
