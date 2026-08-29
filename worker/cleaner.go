package worker

import (
	"time"

	"github.com/rajandhamala/goRedis/src"
)

// for deleting expired keys
// currenly highly unscalable due to looping over all keys and checking expiry & mutex locks
// later we will introduce more efficent way of invalidaing expired keys

func FLushExpiredKeys() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for tickTime := range ticker.C {

		src.KeyMu.Lock()

		for key, val := range src.ActiveKeys {

			if val.TTL.IsZero() {
				continue
			}

			if tickTime.After(val.TTL) {
				delete(src.ActiveKeys, key)
			}
		}

		src.KeyMu.Unlock()
	}
}
