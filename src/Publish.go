package src

import (
	"errors"
	"fmt"
)

func PublishEvent(msg []string) (string, error) {
	// to keep track of all the clients subscrbed to channel we can leverage map as it is very fast for 10 20 subscribers on same channel
	// as we loop over the whole array and emit subscribe vent to all client subscribed to respective channel but issue arise during unsubscribe
	// cause we need to do unnecessary loops to find client and remove but adding unsubscribe channel itself seems unnecessary and adds overhead still we will use nested slice
	fmt.Println("Publish to channel invoked")

	name := msg[1]
	data := msg[2]
	// no checks during testing

	// get clients on channel
	// allow read only
	SubMu.RLock()
	resp, ok := ActiveSubscribers[name]
	if !ok {
		fmt.Println("no one found on the channel")
		SubMu.RUnlock()
		return "", errors.New("no one found on the channel")
	}

	clients := make([]*Client, 0, len(resp))
	for client := range resp {
		clients = append(clients, client)
	}
	SubMu.RUnlock()

	payload := []byte(name + " " + data + "\n")
	for _, user := range clients {
		user.Send <- payload
	}
	return "Event Published Succesfully\n", nil
}
