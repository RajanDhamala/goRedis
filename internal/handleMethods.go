package internal

import (
	"net"
	"strings"

	"github.com/rajandhamala/goRedis/src"
)

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

		_, err := src.DelKey(msg[1])
		if err != nil {
			_, _ = conn.Write([]byte(err.Error() + "\n"))
			return
		}

		_, _ = conn.Write([]byte("Key deleted successfully\n"))

	default:
		_, _ = conn.Write([]byte("Unsupported method\n"))
	}
}
