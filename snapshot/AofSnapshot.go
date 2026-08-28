package snapshot

import (
	"bufio"
	"log"
	"os"
	"time"
)

var AofChan = make(chan []byte, 50)

func AofWoker() {
	file, err := os.OpenFile("appendonly.aof", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Println("unable to open file", err.Error())
		panic(err)

	}
	defer file.Close()
	writer := bufio.NewWriterSize(file, 64*1024)

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case data := <-AofChan:
			_, _ = writer.Write(data)

			if writer.Buffered() >= 64*1024 {
				_ = writer.Flush()
			}

		case <-ticker.C:
			_ = writer.Flush()
		}
	}
}
