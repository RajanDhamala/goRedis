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
	_, _ = conn.Write([]byte("hello from server\n"))

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
			_, _ = conn.Write([]byte("no command found\n"))
		}

		HandleMethods(msg, conn)
	}
}
