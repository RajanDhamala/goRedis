package internal

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rajandhamala/goRedis/helpers"
)

type Entry struct {
	Value string    `json:"key"`
	TTL   time.Time `json:"ttl"`
}

var (
	ActiveKeys = make(map[string]*Entry)
	Mu         sync.RWMutex
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
		data := Entry{}

		if length == 0 {
			fmt.Println("no command found")
			_, _ = conn.Write([]byte("no command found\n"))
		}

		method := strings.ToUpper(msg[0])
		switch method {
		case "GET":
			Mu.RLocker()
			fmt.Println("Get method")
			if length >= 1 {
				println("Key:", msg[1])

				Mu.RLock()
				resp, ok := ActiveKeys[msg[1]]
				Mu.RUnlock()

				if !ok {
					fmt.Println("invalid key or expired ")
					_, _ = conn.Write([]byte("invalid key or expired \n"))
					continue
				}

				_, _ = conn.Write([]byte(resp.Value + "\n"))
			}

		case "SET":
			fmt.Println("set method")
			if length >= 1 {
				println("Key:", msg[1])
			}
			if length >= 3 {
				println("Value:", msg[2])
				data.Value = msg[2]
				ttl, err := strconv.Atoi(msg[3])
				if err != nil {
					fmt.Println("invalid ttl")
				}
				data.TTL = time.Now().Add(time.Duration(ttl) * time.Second)
				Mu.Lock()
				ActiveKeys[msg[1]] = &data
				Mu.Unlock()
				_, _ = conn.Write([]byte("key set succesfully\n"))
				continue
			}

		case "DEL":
			fmt.Println("del method")
			if length >= 1 {
				println("Key:", msg[1])
				Mu.RLock()
				_, exists := ActiveKeys[msg[1]]
				Mu.RUnlock()
				if exists {
					fmt.Println("Found and deleting:", msg[1])
					Mu.Lock()
					delete(ActiveKeys, msg[1])
					Mu.Unlock()
					_, err = conn.Write([]byte("key deleted succesfully\n"))
				} else {
					fmt.Println("Key not found")
					_, err = conn.Write([]byte("key not found\n"))
				}
			}

		}

		if err != nil {
			fmt.Println("error writing resposne")
			return
		}
	}
}
