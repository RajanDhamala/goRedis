package main

import (
	"fmt"
	"log"
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

	server, err := internal.NewServer(os.Getenv("REDIS_PASSWORD"))
	if err != nil {
		log.Fatal(err)
	}
	host := os.Getenv("HOST")
	if host == "" {
		host = "127.0.0.1"
	}

	PORT := os.Getenv("PORT")

	if PORT == "" {
		PORT = "6379"
	}

	if err := snapshot.PlayAofShapshot(); err != nil {
		log.Fatal(err)
	}
	listener, err := net.Listen("tcp", net.JoinHostPort(host, PORT))
	if err != nil {
		fmt.Println("error while listing for TCP req", err)
		panic("error while listining on TCP Port")
	}
	fmt.Println("TCP server is listening on:", listener.Addr())

	go worker.FLushExpiredKeys()
	go snapshot.AofWoker()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("error while accepting req", err)
			continue
		}
		go server.HandleConnection(conn)
	}
}
