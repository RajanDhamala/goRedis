package src

func SubscribeEvent(msg []string, client *Client) (string, error) {
	name := msg[1]

	client.Mu.Lock()

	if client.Subscriptions == nil {
		client.Subscriptions = make(map[string]struct{})
	}
	client.Subscriptions[name] = struct{}{}

	client.Mu.Unlock()

	SubMu.Lock()

	subs, ok := ActiveSubscribers[name]
	if !ok {
		subs = make(map[*Client]struct{})
		ActiveSubscribers[name] = subs
	}

	subs[client] = struct{}{}

	SubMu.Unlock()

	return "Event Subscribed Successfully\n", nil
}
