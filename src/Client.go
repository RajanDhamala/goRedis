package src

import (
	"net"
	"sync/atomic"
)

var nextClientID atomic.Int64

func NewClient(conn net.Conn) *Client {
	return &Client{
		Conn:          conn,
		Send:          make(chan []byte, 1024),
		Done:          make(chan struct{}),
		Subscriptions: make(map[string]struct{}),
		ID:            nextClientID.Add(1),
	}
}

// TrySend disconnects a slow client rather than holding up other commands.
// Send is never closed: publishers may still hold a reference to this client.
func (c *Client) TrySend(payload []byte) bool {
	c.Mu.Lock()
	if c.Closed {
		c.Mu.Unlock()
		return false
	}
	select {
	case c.Send <- payload:
		c.Mu.Unlock()
		return true
	default:
		c.Mu.Unlock()
		c.Disconnect()
		return false
	}
}

func (c *Client) Disconnect() {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	if c.Closed {
		return
	}
	c.Closed = true
	SubMu.Lock()
	for name := range c.Subscriptions {
		delete(ActiveSubscribers[name], c)
		if len(ActiveSubscribers[name]) == 0 {
			delete(ActiveSubscribers, name)
		}
	}
	SubMu.Unlock()
	clear(c.Subscriptions)
	close(c.Done)
	if c.Conn != nil {
		c.Conn.Close()
	}
}

func (c *Client) SubscriptionCount() int {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	return len(c.Subscriptions)
}
