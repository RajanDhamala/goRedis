package internal

import (
	"strconv"
	"strings"
	"time"

	r "github.com/rajandhamala/goRedis/helpers"
	"github.com/rajandhamala/goRedis/snapshot"
	"github.com/rajandhamala/goRedis/src"
)

func setCommand(msg []string) []byte {
	var expiry time.Time
	nx, xx, get, keep, hasExpiry := false, false, false, false, false
	for i := 3; i < len(msg); i++ {
		option := strings.ToUpper(msg[i])
		switch option {
		case "NX":
			if nx || xx {
				return r.Error("ERR syntax error")
			}
			nx = true
		case "XX":
			if nx || xx {
				return r.Error("ERR syntax error")
			}
			xx = true
		case "GET":
			get = true
		case "KEEPTTL":
			if hasExpiry || keep {
				return r.Error("ERR syntax error")
			}
			keep = true
		case "EX", "PX", "EXAT", "PXAT":
			if hasExpiry || keep || i+1 == len(msg) {
				return r.Error("ERR syntax error")
			}
			hasExpiry = true
			i++
			amount, err := strconv.ParseInt(msg[i], 10, 64)
			if err != nil {
				return integerError()
			}
			unit := time.Millisecond
			if option == "EX" || option == "EXAT" {
				unit = time.Second
			}
			if amount <= 0 || amount > int64((1<<63-1)/unit) {
				return r.Error("ERR invalid expire time in 'set' command")
			}
			if option == "EXAT" {
				expiry = time.Unix(amount, 0)
			} else if option == "PXAT" {
				expiry = time.UnixMilli(amount)
			} else {
				expiry = time.Now().Add(time.Duration(amount) * unit)
			}
		default:
			return r.Error("ERR syntax error")
		}
	}
	kind := src.KeyType(msg[1])
	old := r.Null()
	if get {
		if kind != "none" && kind != "string" {
			return r.Error("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
		if value, err := src.GetKey(msg[1]); err == nil {
			old = r.Bulk(value)
		}
	}
	if (nx && kind != "none") || (xx && kind == "none") {
		return old
	}
	if keep {
		expiry = src.Expiry(msg[1])
	}
	src.SetValue(msg[1], msg[2], expiry)
	snapshot.RecordString(msg[1])
	if get {
		return old
	}
	return r.Simple("OK")
}
