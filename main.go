package main

import (
	"fmt"
	"net"
	"os"

	"github.com/joho/godotenv"

	"github.com/rajandhamala/goRedis/internal"

	"github.com/rajandhamala/goRedis/worker"

	"github.com/rajandhamala/goRedis/snapshot"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	PORT := os.Getenv("PORT")

	if PORT == "" {
		fmt.Println("Missing PORT no")
		return
	}

	listener, err := net.Listen("tcp", ":"+PORT)
	if err != nil {
		fmt.Println("error while listing for TCP req", err)
	}
	fmt.Println("TCP server is listening on port:", PORT)

	snapshot.PlayAofShapshot()
	go worker.FLushExpiredKeys()
	go snapshot.AofWoker()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("error while accepting req", err)
		}
		go internal.HandleConnection(conn)
	}
}
