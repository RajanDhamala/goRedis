package internal

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	r "github.com/rajandhamala/goRedis/helpers"
	"github.com/rajandhamala/goRedis/snapshot"
	"github.com/rajandhamala/goRedis/src"
)

// Negative maximum means variadic. Validate before any handler indexes arguments.
var commandArity = map[string][2]int{
	"MULTI": {1, 1}, "EXEC": {1, 1}, "DISCARD": {1, 1}, "AUTH": {2, 3}, "PING": {1, 2}, "ECHO": {2, 2}, "QUIT": {1, 1}, "HELLO": {1, -1}, "CLIENT": {2, -1}, "SELECT": {2, 2},
	"GET": {2, 2}, "SET": {3, -1}, "DEL": {2, -1}, "EXISTS": {2, -1}, "TYPE": {2, 2},
	"INCR": {2, 2}, "DECR": {2, 2}, "INCRBY": {3, 3}, "DECRBY": {3, 3}, "TTL": {2, 2}, "PTTL": {2, 2}, "EXPIRE": {3, 3}, "PEXPIRE": {3, 3},
	"INFO": {1, 2}, "HSET": {4, -1}, "HGET": {3, 3}, "HDEL": {3, -1}, "HLEN": {2, 2}, "HEXISTS": {3, 3}, "HGETALL": {2, 2},
	"SADD": {3, -1}, "SREM": {3, -1}, "SISMEMBER": {3, 3}, "SMEMBERS": {2, 2}, "SCARD": {2, 2},
	"LPUSH": {3, -1}, "RPUSH": {3, -1}, "LPOP": {2, 2}, "RPOP": {2, 2}, "LRANGE": {4, 4}, "LLEN": {2, 2},
	"ZADD": {4, -1}, "ZSCORE": {3, 3}, "TEST": {2, 2},
	"PUBLISH": {3, 3}, "SUBSCRIBE": {2, -1}, "UNSUBSCRIBE": {1, -1},
}

func (s *Server) HandleMethods(msg []string, client *src.Client) {
	src.CommandMu.Lock()
	defer src.CommandMu.Unlock()
	reply := s.dispatchCommand(msg, client)
	if reply != nil {
		client.TrySend(reply)
	}
}

func validateCommand(msg []string, client *src.Client) []byte {
	if len(msg) == 0 {
		return r.Error("ERR empty command")
	}
	method := strings.ToUpper(msg[0])
	if !client.Authenticated && method != "AUTH" && method != "HELLO" && method != "QUIT" {
		return r.Error("NOAUTH Authentication required.")
	}
	arity, ok := commandArity[method]
	if !ok {
		return r.Error(fmt.Sprintf("ERR unknown command '%s'", msg[0]))
	}
	if len(msg) < arity[0] || (arity[1] >= 0 && len(msg) > arity[1]) || ((method == "HSET" || method == "ZADD") && len(msg)%2 != 0) {
		return r.Error(fmt.Sprintf("ERR wrong number of arguments for '%s' command", strings.ToLower(method)))
	}
	if client.SubscriptionCount() > 0 && method != "SUBSCRIBE" && method != "UNSUBSCRIBE" && method != "PING" && method != "QUIT" {
		return r.Error("ERR Can't execute command while subscribed")
	}
	return nil
}

// Called with CommandMu held; validation here does not inspect transaction state.
func (s *Server) executeCommand(msg []string, client *src.Client, journal *snapshot.Recorder) []byte {
	if reply := validateCommand(msg, client); reply != nil {
		return reply
	}
	method := strings.ToUpper(msg[0])
	if want := expectedType(method); want != "" {
		kind := src.KeyType(msg[1])
		if kind != "none" && kind != want {
			return r.Error("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
	}
	switch method {
	case "AUTH":
		return s.authCommand(msg, client)
	case "PING":
		if client.SubscriptionCount() > 0 {
			payload := ""
			if len(msg) == 2 {
				payload = msg[1]
			}
			return r.Strings([]string{"pong", payload})
		}
		if len(msg) == 2 {
			return r.Bulk(msg[1])
		}
		return r.Simple("PONG")
	case "ECHO":
		return r.Bulk(msg[1])
	case "QUIT":
		return r.Simple("OK")
	case "HELLO":
		return s.hello(msg, client)
	case "CLIENT":
		return clientCommand(msg, client)
	case "SELECT":
		db, err := strconv.ParseInt(msg[1], 10, 64)
		if err != nil {
			return integerError()
		}
		if db != 0 {
			return r.Error("ERR DB index is out of range; only database 0 is supported")
		}
		return r.Simple("OK")
	case "GET":
		value, err := src.GetKey(msg[1])
		if err != nil {
			return r.Null()
		}
		return r.Bulk(value)
	case "SET":
		return setCommand(msg, journal)
	case "TYPE":
		return r.Simple(src.KeyType(msg[1]))
	case "DEL":
		var removed int64
		for _, key := range msg[1:] {
			if src.KeyType(key) != "none" {
				src.DeleteKey(key)
				removed++
			}
		}
		journal.Delete(msg[1:]...)
		return r.Integer(removed)
	case "EXISTS":
		var count int64
		for _, key := range msg[1:] {
			if src.KeyType(key) != "none" {
				count++
			}
		}
		return r.Integer(count)
	case "INCR", "DECR", "INCRBY", "DECRBY":
		var value int64
		var err error
		if method == "INCR" {
			value, err = src.Incr(msg)
		} else if method == "DECR" {
			value, err = src.Decr(msg)
		} else {
			value, err = src.CounterBy(msg)
		}
		if err != nil {
			return r.Error("ERR " + err.Error())
		}
		journal.String(msg[1])
		return r.Integer(value)
	case "TTL", "PTTL":
		if src.KeyType(msg[1]) == "none" {
			return r.Integer(-2)
		}
		expiry := src.Expiry(msg[1])
		if expiry.IsZero() {
			return r.Integer(-1)
		}
		remaining := time.Until(expiry)
		if remaining < 0 {
			src.DeleteKey(msg[1])
			return r.Integer(-2)
		}
		if method == "PTTL" {
			return r.Integer(remaining.Milliseconds())
		}
		seconds := int64(remaining / time.Second)
		if remaining%time.Second >= 500*time.Millisecond {
			seconds++
		}
		return r.Integer(seconds)
	case "EXPIRE", "PEXPIRE":
		amount, err := strconv.ParseInt(msg[2], 10, 64)
		if err != nil {
			return integerError()
		}
		unit := time.Second
		if method == "PEXPIRE" {
			unit = time.Millisecond
		}
		if amount > int64((1<<63-1)/unit) {
			return integerError()
		}
		expiry := time.Now()
		if amount > 0 {
			expiry = expiry.Add(time.Duration(amount) * unit)
		}
		kind := src.KeyType(msg[1])
		if !src.ExpireAt(msg[1], expiry) {
			return r.Integer(0)
		}
		if kind == "string" {
			journal.String(msg[1])
		}
		return r.Integer(1)
	case "INFO":
		return r.Bulk("# Server\r\nredis_version:0.1.0\r\nredis_mode:standalone\r\n# Persistence\r\naof_enabled:1\r\n")
	case "HSET":
		value, err := src.HSET(msg)
		return integerResult(value, err)
	case "HDEL":
		value, err := src.HDEL(msg)
		return integerResult(value, err)
	case "HGET":
		value, err := src.HGET(msg)
		if err != nil {
			return r.Null()
		}
		return r.Bulk(value)
	case "HLEN":
		value, _ := src.HLEN(msg)
		return r.Integer(int64(value))
	case "HEXISTS":
		value, _ := src.HEXISTS(msg)
		return boolResult(value)
	case "HGETALL":
		values, _ := src.HGETALL(msg)
		return r.Strings(values)
	case "SADD":
		value, err := src.SADD(msg)
		return integerResult(value, err)
	case "SREM":
		value, err := src.SREM(msg)
		return integerResult(value, err)
	case "SISMEMBER":
		value, _ := src.SISMEMBER(msg)
		return boolResult(value)
	case "SMEMBERS":
		values, _ := src.SMEMBERS(msg)
		return r.Strings(values)
	case "SCARD":
		value, _ := src.SCARD(msg)
		return r.Integer(int64(value))
	case "LPUSH", "RPUSH":
		for _, value := range msg[2:] {
			args := []string{method, msg[1], value}
			if method == "LPUSH" {
				src.LPUSH(args)
			} else {
				src.RPUSH(args)
			}
		}
		length, _ := src.LLEN(msg)
		return r.Integer(int64(length))
	case "LPOP", "RPOP":
		var value string
		var err error
		if method == "LPOP" {
			value, err = src.LPOP(msg)
		} else {
			value, err = src.RPOP(msg)
		}
		if err != nil {
			return r.Null()
		}
		return r.Bulk(value)
	case "LRANGE":
		if _, err := strconv.Atoi(msg[2]); err != nil {
			return integerError()
		}
		if _, err := strconv.Atoi(msg[3]); err != nil {
			return integerError()
		}
		values, _ := src.LRANGE(msg)
		return r.Strings(values)
	case "LLEN":
		value, _ := src.LLEN(msg)
		return r.Integer(int64(value))
	case "ZADD":
		value, err := src.ZADD(msg)
		return integerResult(value, err)
	case "ZSCORE":
		value, err := src.ZSCORE(msg)
		if err != nil {
			return r.Null()
		}
		return r.Bulk(value)
	case "TEST":
		value, _ := src.TraverseZset(msg)
		return r.Bulk(value)
	case "PUBLISH":
		value, _ := src.PublishEvent(msg)
		return r.Integer(value)
	case "SUBSCRIBE":
		for _, channel := range msg[1:] {
			src.SubscribeEvent([]string{method, channel}, client)
			client.TrySend(r.Array(r.Bulk("subscribe"), r.Bulk(channel), r.Integer(int64(client.SubscriptionCount()))))
		}
		return nil
	case "UNSUBSCRIBE":
		channels := msg[1:]
		if len(channels) == 0 {
			client.Mu.Lock()
			for channel := range client.Subscriptions {
				channels = append(channels, channel)
			}
			client.Mu.Unlock()
			sort.Strings(channels)
			if len(channels) == 0 {
				return r.Array(r.Bulk("unsubscribe"), r.Null(), r.Integer(0))
			}
		}
		for _, channel := range channels {
			src.UnsubscribeEvent([]string{method, channel}, client)
			client.TrySend(r.Array(r.Bulk("unsubscribe"), r.Bulk(channel), r.Integer(int64(client.SubscriptionCount()))))
		}
		return nil
	}
	return r.Error("ERR unsupported command")
}

func expectedType(method string) string {
	switch method {
	case "GET", "INCR", "DECR", "INCRBY", "DECRBY":
		return "string"
	case "HSET", "HGET", "HDEL", "HLEN", "HEXISTS", "HGETALL":
		return "hash"
	case "SADD", "SREM", "SISMEMBER", "SMEMBERS", "SCARD":
		return "set"
	case "LPUSH", "RPUSH", "LPOP", "RPOP", "LRANGE", "LLEN":
		return "list"
	case "ZADD", "ZSCORE", "TEST":
		return "zset"
	}
	return ""
}
func integerError() []byte { return r.Error("ERR value is not an integer or out of range") }
func integerResult(value int, err error) []byte {
	if err != nil {
		return r.Error("ERR " + err.Error())
	}
	return r.Integer(int64(value))
}
func boolResult(value bool) []byte {
	if value {
		return r.Integer(1)
	}
	return r.Integer(0)
}
