package worker

import (
	"fmt"
	"time"
)

// for deleting expired keys
func FLushExpiredKeys() {
	newticker := time.NewTicker(time.Second * 2)
	defer newticker.Stop()

	for tickTime := range newticker.C {
		fmt.Println("ticker just ticked", tickTime)
	}
}
