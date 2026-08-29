package internal

import (
	"strings"

	"github.com/rajandhamala/goRedis/snapshot"
	"github.com/rajandhamala/goRedis/src"
)

var ActiveSubscribers = make(map[string]map[*src.Client]struct{})

func HandleMethods(msg []string, client *src.Client) {
	length := len(msg)

	if length < 1 {
		client.Send <- []byte("no command found\n")
		return
	}

	method := strings.ToUpper(msg[0])

	switch method {

	case "GET":
		if length != 2 {
			client.Send <- []byte("usage: GET key\n")
			return
		}

		data, err := src.GetKey(msg[1])
		if err != nil {
			client.Send <- []byte(err.Error() + "\n")
			return
		}
		client.Send <- []byte(data + "\n")

	case "SET":
		if length != 4 {
			client.Send <- []byte("usage: SET key value ttl\n")
			return
		}
		data := []byte(strings.Join(msg, " ") + "\n")
		snapshot.AofChan <- data

		_, err := src.AddKey(msg[1], msg[2], msg[3])
		if err != nil {
			client.Send <- []byte(err.Error() + "\n")
			return
		}
		client.Send <- []byte("Key added successfully\n")

	case "DEL":
		if length != 2 {
			client.Send <- []byte("usage: DEL key\n")
			return
		}
		data := []byte(strings.Join(msg, " ") + "\n")
		snapshot.AofChan <- data

		_, err := src.DelKey(msg[1])
		if err != nil {

			client.Send <- []byte(err.Error() + "\n")
			return
		}

		client.Send <- []byte("Key deleted successfully\n")

	case "PUBLISH":
		resp, err := src.PublishEvent(msg)
		if err != nil {
			client.Send <- []byte("failed to publish event \n")
		}

		client.Send <- []byte(resp)

	case "SUBSCRIBE":
		resp, err := src.SubscribeEvent(msg, client)
		if err != nil {
			client.Send <- []byte("failed to subscribe event \n")
		}

		client.Send <- []byte(resp)

	case "UNSUBSCRIBE":
		resp, err := src.UnsubscribeEvent(msg, client)
		if err != nil {
			client.Send <- []byte("failed to unsubscribe event")
		}

		client.Send <- []byte(resp)

	default:
		client.Send <- []byte("Unsupported method\n")
	}
}
