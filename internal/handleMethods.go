package internal

import (
	"net"
	"strings"

	"github.com/rajandhamala/goRedis/snapshot"
	"github.com/rajandhamala/goRedis/src"
)

type Subscriber struct {
	Conn net.Conn
	Send chan []byte
}

var ActiveSubscribers = make(map[string]map[*Subscriber]struct{})

func HandleMethods(msg []string, conn net.Conn) {
	length := len(msg)

	if length < 1 {
		_, _ = conn.Write([]byte("no command found\n"))
		return
	}

	method := strings.ToUpper(msg[0])

	switch method {

	case "GET":
		if length != 2 {
			_, _ = conn.Write([]byte("usage: GET key\n"))
			return
		}

		data, err := src.GetKey(msg[1])
		if err != nil {
			_, _ = conn.Write([]byte(err.Error() + "\n"))
			return
		}

		_, _ = conn.Write([]byte(data + "\n"))

	case "SET":
		if length != 4 {
			_, _ = conn.Write([]byte("usage: SET key value ttl\n"))
			return
		}
		data := []byte(strings.Join(msg, " ") + "\n")
		snapshot.AofChan <- data

		_, err := src.AddKey(msg[1], msg[2], msg[3])
		if err != nil {
			_, _ = conn.Write([]byte(err.Error() + "\n"))
			return
		}

		_, _ = conn.Write([]byte("Key added successfully\n"))

	case "DEL":
		if length != 2 {
			_, _ = conn.Write([]byte("usage: DEL key\n"))
			return
		}
		data := []byte(strings.Join(msg, " ") + "\n")
		snapshot.AofChan <- data

		_, err := src.DelKey(msg[1])
		if err != nil {
			_, _ = conn.Write([]byte(err.Error() + "\n"))
			return
		}

		_, _ = conn.Write([]byte("Key deleted successfully\n"))

	case "PUBLISH":
		resp, err := src.PublishEvent(msg)
		if err != nil {
			_, _ = conn.Write([]byte("failed to publish event"))
		}
		_, _ = conn.Write([]byte(resp))

	case "SUBSCRIBE":
		resp, err := src.SubscribeEvent(msg, conn)
		if err != nil {
			_, _ = conn.Write([]byte("failed to subscribe event"))
		}
		_, _ = conn.Write([]byte(resp))

	case "UNSUBSCRIBE":
		resp, err := src.UnsubscribeEvent(msg)
		if err != nil {
			_, _ = conn.Write([]byte("failed to unsubscribe event"))
		}
		_, _ = conn.Write([]byte(resp))

	default:
		_, _ = conn.Write([]byte("Unsupported method\n"))
	}
}
