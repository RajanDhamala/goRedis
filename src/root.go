package src

// root/warehouse for server status

import (
	"sync"
	"time"
)

type Entry struct {
	Value string    `json:"key"`
	TTL   time.Time `json:"ttl"`
}

var (
	ActiveKeys = make(map[string]*Entry)
	Mu         sync.RWMutex
)
