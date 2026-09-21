package src

func SubscribeEvent(msg []string, client *Client) (string, error) {
	name := msg[1]
	client.Mu.Lock()
	defer client.Mu.Unlock()
	if client.Closed {
		return "", nil
	}
	SubMu.Lock()
	defer SubMu.Unlock()
	client.Subscriptions[name] = struct{}{}
	if ActiveSubscribers[name] == nil {
		ActiveSubscribers[name] = make(map[*Client]struct{})
	}
	ActiveSubscribers[name][client] = struct{}{}
	return "", nil
}
