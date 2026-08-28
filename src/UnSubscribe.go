package src

import (
	"errors"
)

func UnsubscribeEvent(msg []string) (string, error) {
	// currenly not working
	name := msg[1]

	_, ok := ActiveSubscribers[name]
	if !ok {
		return "", errors.New("Was not Subscribed\n")
	}
	return "Channel Unsubscribed Succesfully\n", nil
}
