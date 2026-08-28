package src

// root/warehouse for server status

import (
	"net"
	"sync"
	"time"
)

type Entry struct {
	Value string    `json:"key"`
	TTL   time.Time `json:"ttl"`
}

type Subscriber struct {
	Conn net.Conn
	Send chan []byte
}

var (
	ActiveKeys = make(map[string]*Entry)
	Mu         sync.RWMutex

	ActiveSubscribers = make(map[string]map[*Subscriber]struct{})
)
