package internal

import (
	"bufio"
	"fmt"
	"net"

	"github.com/rajandhamala/goRedis/helpers"

	"github.com/rajandhamala/goRedis/src"
)

func HandleConnection(conn net.Conn) {
	client := src.Client{
		Conn: conn,
		Send: make(chan []byte, 100),
	}

	go func() {
		for msg := range client.Send {
			_, err := client.Conn.Write(msg)
			if err != nil {
				return
			}
		}
	}()
	defer func() {
		conn.Close()

		client.Mu.Lock()

		for name := range client.Subscriptions {
			src.SubMu.Lock()

			if subscribers, ok := src.ActiveSubscribers[name]; ok {
				delete(subscribers, &client)
			}

			src.SubMu.Unlock()
		}

		client.Mu.Unlock()

		close(client.Send)
	}()

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

		HandleMethods(msg, &client)
	}
}
