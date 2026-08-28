package src

import (
	"errors"
)

func DelKey(key string) (string, error) {
	Mu.Lock()
	defer Mu.Unlock()

	_, exists := ActiveKeys[key]
	if !exists {
		return "", errors.New("key not found")
	}

	delete(ActiveKeys, key)

	return "key deleted successfully", nil
}
