package internal

import (
	"bufio"
	"fmt"
	"net"

	"github.com/rajandhamala/goRedis/helpers"
)

func HandleConnection(conn net.Conn) {
	defer conn.Close()

	fmt.Println("client conncected", conn.RemoteAddr())

	reader := bufio.NewReader(conn)
	for {
		msg, err := helpers.ReadCommand(reader)
		if err != nil {
			fmt.Println("error wile reading buffer", err)
			return
		}

		length := len(msg)

		if length == 0 {
			fmt.Println("no command found")
		} else if length >= 1 {
			println("Mehod:", msg[0])
		}
		if length >= 2 {
			println("Key:", msg[1])
		}
		if length >= 3 {
			println("Value:", msg[2])
		}

		_, err = conn.Write([]byte("hello from server\n"))
		if err != nil {
			fmt.Println("error writing resposne")
			return
		}
	}
}
