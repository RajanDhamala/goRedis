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

	// get client on channel
	resp, ok := ActiveSubscribers[name]
	if !ok {
		fmt.Println("no one found on the channel")
		return "", errors.New("no one found on the channel")
	}
	payload := []byte(name + " " + data + "\n")
	for clients := range resp {
		clients.Send <- payload
	}
	return "Event Published Succesfully\n", nil
}
