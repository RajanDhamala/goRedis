package snapshot

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/rajandhamala/goRedis/src"
)

func PlayAofShapshot() {
	fmt.Println("Playing shapshots from disk")

	file, err := os.Open("appendonly.aof")
	if err != nil {
		fmt.Println("error while opening file")
		panic(err)
	}

	defer file.Close()
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		msg := scanner.Text()
		final := strings.Fields(msg)

		if len(msg) == 0 {
			continue
		}
		ExecAofLog(final)
	}
	fmt.Println("Redis prev snapshot achieved")
	if err := scanner.Err(); err != nil {
		fmt.Println("error reading AOF:", err)
	}
}

func ExecAofLog(msg []string) {
	method := strings.ToUpper(msg[0])
	if method == "GET" {
		// no need to play the get comamnd we wont even log it for safely checking during devleopment
		return
	}

	switch method {

	case "SET":
		if len(msg) < 4 {
			fmt.Println("invalid SET entry in AOF:", msg)
			return
		}

		_, err := src.AddKey(msg[1], msg[2], msg[3])
		if err != nil {
			fmt.Println("failed to replay SET:", err)
		}

	case "DEL":
		if len(msg) < 2 {
			fmt.Println("invalid DEL entry in AOF:", msg)
			return
		}

		_, err := src.DelKey(msg[1])
		if err != nil {
			fmt.Println("failed to replay DEL:", err)
		}
	}
}
