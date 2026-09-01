package src

import (
	"errors"
	"strconv"
)

func Incr(msg []string) (int, error) {
	key := msg[1]

	length := len(msg)
	value := 1

	if length >= 3 {
		temp, err := strconv.Atoi(msg[2])
		if err != nil {
			return 0, errors.New("unable to incr value")
		}
		value = temp
	}

	KeyMu.Lock()

	defer KeyMu.Unlock()
	data, ok := ActiveKeys[key]

	if !ok {
		return 0, errors.New("key not found")
	}

	intvalue, errr := strconv.Atoi(data.Value)

	if errr != nil {
		return 0, errors.New("failed to parse value")
	}
	intvalue += value

	data.Value = strconv.Itoa(intvalue)
	return intvalue, nil
}

func Decr(msg []string) (int, error) {
	key := msg[1]

	length := len(msg)
	value := 1

	if length >= 3 {
		temp, err := strconv.Atoi(msg[2])
		if err != nil {
			return 0, errors.New("unable to decr value")
		}
		value = temp
	}

	KeyMu.Lock()

	defer KeyMu.Unlock()
	data, ok := ActiveKeys[key]

	if !ok {
		return 0, errors.New("key not found")
	}

	intvalue, err := strconv.Atoi(data.Value)
	if err != nil {
		return 0, errors.New("failed to parse counter")
	}

	intvalue -= value
	data.Value = strconv.Itoa(intvalue)

	return intvalue, nil
}
