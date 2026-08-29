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

type Client struct {
	Conn          net.Conn
	Send          chan []byte
	Subscriptions map[string]struct{}
	Mu            sync.Mutex
}

var (
	ActiveKeys = make(map[string]*Entry)
	KeyMu      sync.RWMutex

	ActiveSubscribers = make(map[string]map[*Client]struct{})
	SubMu             sync.RWMutex
)
