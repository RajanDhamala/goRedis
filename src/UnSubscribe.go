package src

func UnsubscribeEvent(msg []string, client *Client) (string, error) {
	name := msg[1]
	client.Mu.Lock()
	defer client.Mu.Unlock()
	SubMu.Lock()
	defer SubMu.Unlock()
	delete(client.Subscriptions, name)
	delete(ActiveSubscribers[name], client)
	if len(ActiveSubscribers[name]) == 0 {
		delete(ActiveSubscribers, name)
	}
	return "", nil
}
