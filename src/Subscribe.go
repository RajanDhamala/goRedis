package src

import (
	"fmt"
	"net"
)

func SubscribeEvent(msg []string, conn net.Conn) (string, error) {
	name := msg[1]

	fmt.Println("channel name:", name)
	userchannel := make(chan []byte, 50)
	user := Subscriber{
		Conn: conn,
		Send: userchannel,
	}
	if _, ok := ActiveSubscribers[name]; !ok {
		ActiveSubscribers[name] = make(map[*Subscriber]struct{})
	}

	ActiveSubscribers[name][&user] = struct{}{}

	_, _ = conn.Write([]byte("Subscribed to Channel Successfully\n"))

	go func() {
		for data := range userchannel {
			fmt.Println("we got message on user channel btw", data)
			_, _ = conn.Write(data)
		}
	}()
	// we can just close channel to stop writer goRoutine
	return "Event Subscribed Successfully\n", nil
}
