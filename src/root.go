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
	Done          chan struct{}
	Closed        bool
	Name          string
	ID            int64
	Authenticated bool
	Transaction   *Transaction
}

// Transaction is connection-local state protected by CommandMu.
type Transaction struct {
	Commands [][]string
	Bytes    int
	Failed   bool
}

// CommandMu serializes prototype command execution, including collection access and AOF ordering.
var CommandMu sync.Mutex

var (
	ActiveKeys = make(map[string]*Entry)
	KeyMu      sync.RWMutex

	ActiveSubscribers = make(map[string]map[*Client]struct{})
	SubMu             sync.RWMutex
)
