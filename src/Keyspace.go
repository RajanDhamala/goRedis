package src

import "time"

// Collection metadata and the collection maps are protected by CommandMu.
// String entry access additionally uses KeyMu for the expiration worker.
var collectionExpiry = make(map[string]time.Time)

func KeyType(key string) string {
	KeyMu.Lock()
	entry, ok := ActiveKeys[key]
	if ok && !entry.TTL.IsZero() && !time.Now().Before(entry.TTL) {
		delete(ActiveKeys, key)
		ok = false
	}
	KeyMu.Unlock()
	if ok {
		return "string"
	}
	if expiry, ok := collectionExpiry[key]; ok && !time.Now().Before(expiry) {
		DeleteKey(key)
		return "none"
	}
	if _, ok := GlobalHash[key]; ok {
		return "hash"
	}
	if _, ok := GobalSet[key]; ok {
		return "set"
	}
	if _, ok := GlobalList[key]; ok {
		return "list"
	}
	if _, ok := GlobalZset[key]; ok {
		return "zset"
	}
	delete(collectionExpiry, key)
	return "none"
}

func DeleteKey(key string) {
	KeyMu.Lock()
	delete(ActiveKeys, key)
	KeyMu.Unlock()
	delete(GlobalHash, key)
	delete(GobalSet, key)
	delete(GlobalList, key)
	delete(GlobalZset, key)
	delete(collectionExpiry, key)
}

func SetValue(key, value string, expiry time.Time) {
	DeleteKey(key)
	KeyMu.Lock()
	ActiveKeys[key] = &Entry{Value: value, TTL: expiry}
	KeyMu.Unlock()
}

func Expiry(key string) time.Time {
	KeyMu.RLock()
	entry, ok := ActiveKeys[key]
	var expiry time.Time
	if ok {
		expiry = entry.TTL
	}
	KeyMu.RUnlock()
	if ok {
		return expiry
	}
	return collectionExpiry[key]
}

func ExpireAt(key string, expiry time.Time) bool {
	kind := KeyType(key)
	if kind == "none" {
		return false
	}
	if !time.Now().Before(expiry) {
		DeleteKey(key)
		return true
	}
	if kind == "string" {
		KeyMu.Lock()
		if entry, ok := ActiveKeys[key]; ok {
			entry.TTL = expiry
		}
		KeyMu.Unlock()
	} else {
		collectionExpiry[key] = expiry
	}
	return true
}
