package internal

import (
	"bufio"
	"io"
	"net"
	"strings"
	"time"

	"github.com/rajandhamala/goRedis/helpers"
	"github.com/rajandhamala/goRedis/src"
)

func (s *Server) HandleConnection(conn net.Conn) {
	client := src.NewClient(conn)
	defer client.Disconnect()
	go func() {
		defer client.Disconnect()
		for {
			select {
			case <-client.Done:
				return
			case msg := <-client.Send:
				if msg == nil {
					return
				} // Drain preceding replies before QUIT/EOF.
				_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				for len(msg) > 0 {
					n, err := conn.Write(msg)
					if err != nil || n == 0 {
						return
					}
					msg = msg[n:]
				}
			}
		}
	}()
	finish := func() { client.TrySend(nil); <-client.Done }
	reader := bufio.NewReader(conn)
	for {
		msg, err := helpers.ReadCommand(reader)
		if err != nil {
			if err != io.EOF {
				client.TrySend(helpers.Error("ERR Protocol error: " + err.Error()))
			}
			finish()
			return
		}
		if len(msg) == 0 {
			continue
		}
		s.HandleMethods(msg, client)
		if strings.EqualFold(msg[0], "QUIT") && len(msg) == 1 {
			finish()
			return
		}
		select {
		case <-client.Done:
			return
		default:
		}
	}
}
