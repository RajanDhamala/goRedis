package src

import (
	"errors"
)

func UnsubscribeEvent(msg []string, client *Client) (string, error) {
	name := msg[1]

	client.Mu.Lock()

	if client.Subscriptions == nil {
		client.Mu.Unlock()
		return "No channel joined", errors.New("no subscriptions\n")
	}

	_, ok := client.Subscriptions[name]
	if !ok {
		client.Mu.Unlock()
		return "You're not a subscriber",
			errors.New("has not subscribed yet\n")
	}

	delete(client.Subscriptions, name)

	client.Mu.Unlock()

	SubMu.Lock()

	channel, ok := ActiveSubscribers[name]
	if !ok {
		SubMu.Unlock()
		return "", errors.New("event does not exist\n")
	}

	delete(channel, client)

	SubMu.Unlock()

	return "Channel Unsubscribed Successfully\n", nil
}
