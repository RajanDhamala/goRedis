package src

import "github.com/rajandhamala/goRedis/helpers"

func PublishEvent(msg []string) (int64, error) {
	name, data := msg[1], msg[2]
	SubMu.RLock()
	clients := make([]*Client, 0, len(ActiveSubscribers[name]))
	for client := range ActiveSubscribers[name] {
		clients = append(clients, client)
	}
	SubMu.RUnlock()
	payload := helpers.Strings([]string{"message", name, data})
	var queued int64
	for _, client := range clients {
		if client.TrySend(payload) {
			queued++
		}
	}
	return queued, nil
}
