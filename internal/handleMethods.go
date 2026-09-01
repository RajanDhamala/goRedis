package internal

import (
	"strconv"
	"strings"

	"github.com/rajandhamala/goRedis/snapshot"
	"github.com/rajandhamala/goRedis/src"
)

var ActiveSubscribers = make(map[string]map[*src.Client]struct{})

func HandleMethods(msg []string, client *src.Client) {
	length := len(msg)

	if length < 1 {
		client.Send <- []byte("no command found\n")
		return
	}

	method := strings.ToUpper(msg[0])

	switch method {

	case "GET":
		if length != 2 {
			client.Send <- []byte("usage: GET key\n")
			return
		}

		data, err := src.GetKey(msg[1])
		if err != nil {
			client.Send <- []byte(err.Error() + "\n")
			return
		}
		client.Send <- []byte(data + "\n")

	case "SET":
		if length != 4 {
			client.Send <- []byte("usage: SET key value ttl\n")
			return
		}
		data := []byte(strings.Join(msg, " ") + "\n")
		snapshot.AofChan <- data

		_, err := src.AddKey(msg[1], msg[2], msg[3])
		if err != nil {
			client.Send <- []byte(err.Error() + "\n")
			return
		}
		client.Send <- []byte("Key added successfully\n")

	case "DEL":
		if length != 2 {
			client.Send <- []byte("usage: DEL key\n")
			return
		}
		data := []byte(strings.Join(msg, " ") + "\n")
		snapshot.AofChan <- data

		_, err := src.DelKey(msg[1])
		if err != nil {

			client.Send <- []byte(err.Error() + "\n")
			return
		}

		client.Send <- []byte("Key deleted successfully\n")

	case "INCR":
		resp, err := src.Incr(msg)
		if err != nil {

			client.Send <- []byte(err.Error() + "\n")
			return
		}

		client.Send <- []byte(strconv.Itoa(resp))

	case "DECR":
		resp, err := src.Decr(msg)
		if err != nil {

			client.Send <- []byte(err.Error() + "\n")
			return
		}
		client.Send <- []byte(strconv.Itoa(resp))

	case "PUBLISH":
		resp, err := src.PublishEvent(msg)
		if err != nil {
			client.Send <- []byte("failed to publish event \n")
		}

		client.Send <- []byte(resp)

	case "SUBSCRIBE":
		resp, err := src.SubscribeEvent(msg, client)
		if err != nil {
			client.Send <- []byte("failed to subscribe event \n")
		}

		client.Send <- []byte(resp)

	case "UNSUBSCRIBE":
		resp, err := src.UnsubscribeEvent(msg, client)
		if err != nil {
			client.Send <- []byte("failed to unsubscribe event")
		}

		client.Send <- []byte(resp)

	case "EXISTS":
		resp, err := src.CheckKeyExistance(msg, client)
		if err != nil {
			client.Send <- []byte("key not found")
		}
		if resp {
			client.Send <- []byte("1")
		} else {
			client.Send <- []byte("0")
		}

	case "TTL":
		resp, err := src.CheckTTL(msg, client)
		if err != nil {
			client.Send <- []byte("failed to updae TTL")
		}

		client.Send <- []byte(strconv.FormatInt(resp, 10))

	case "EXPIRE":
		resp, err := src.ExpireKey(msg, client)
		if err != nil {
			client.Send <- []byte("failed to expire key")
		}

		client.Send <- []byte(resp)

	case "INFO":
		resp, err := src.GetRuntimeInfo(msg, client)
		if err != nil {
			client.Send <- []byte("failed to retrive server info")
		}

		client.Send <- []byte(resp)

	case "HSET":
		resp, err := src.HSET(msg)
		if err != nil {
			client.Send <- []byte("failed to create set")
		}

		client.Send <- []byte(resp)

	case "HDEL":
		resp, err := src.HDEL(msg)
		if err != nil {
			client.Send <- []byte("failed delete item from set")
		}

		client.Send <- []byte(resp)

	case "HLEN":
		resp, err := src.HLEN(msg)
		if err != nil {
			client.Send <- []byte("failed to retrive length")
		}

		client.Send <- []byte(strconv.Itoa(resp))

	case "HEXISTS":
		resp, err := src.HEXISTS(msg)
		if err != nil {
			client.Send <- []byte("failed to check existance")
		}

		if resp {
			client.Send <- []byte("true")
		} else {
			client.Send <- []byte("false")
		}

	// could have used sprintf also
	// formatted := fmt.Sprintf("%t", resp)
	// client.Send <- []byte(formatted)

	case "HGETALL":
		resp, err := src.HGETALL(msg)
		if err != nil {
			client.Send <- []byte("failed to retrive items")
		}

		client.Send <- []byte(strings.Join(resp, " "))

	case "SADD":
		resp, err := src.SADD(msg)
		if err != nil {
			client.Send <- []byte("failed to remove member")
		}

		client.Send <- []byte(resp)

	case "SREM":
		resp, err := src.SREM(msg)
		if err != nil {
			client.Send <- []byte("failed to retrive items")
		}

		if resp {
			client.Send <- []byte("true")
		} else {
			client.Send <- []byte("false")
		}

	case "SISMEMBER":
		resp, err := src.SISMEMBER(msg)
		if err != nil {
			client.Send <- []byte("failed to retrive set member")
		}

		if resp {
			client.Send <- []byte("true")
		} else {
			client.Send <- []byte("false")
		}

	case "SMEMBERS":
		resp, err := src.SMEMBERS(msg)
		if err != nil {
			client.Send <- []byte("failed to retrive members")
		}

		client.Send <- []byte(strings.Join(resp, " "))

	case "SCARD":
		resp, err := src.SCARD(msg)
		if err != nil {
			client.Send <- []byte("failed to retrive length of members")
		}

		client.Send <- []byte(strconv.Itoa(resp))

	case "LPUSH":
		resp, err := src.LPUSH(msg)
		if err != nil {
			client.Send <- []byte("failed to LPUSH")
		}

		client.Send <- []byte(resp)

	case "RPUSH":
		resp, err := src.RPUSH(msg)
		if err != nil {
			client.Send <- []byte("failed to RPUSH")
		}

		client.Send <- []byte(resp)

	case "LPOP":
		resp, err := src.LPOP(msg)
		if err != nil {
			client.Send <- []byte("failed to LROP")
		}

		client.Send <- []byte(resp)

	case "RPOP":
		resp, err := src.RPOP(msg)
		if err != nil {
			client.Send <- []byte("failed to RPOP")
		}

		client.Send <- []byte(resp)

	case "LRANGE":
		resp, err := src.LRANGE(msg)
		if err != nil {
			client.Send <- []byte("failed to retrive range of items")
		}
		client.Send <- []byte(strings.Join(resp, " "))

	case "LLEN":
		resp, err := src.LLEN(msg)
		if err != nil {
			client.Send <- []byte("failed to length")
		}

		client.Send <- []byte(strconv.Itoa(resp))

	default:
		client.Send <- []byte("Unsupported method\n")
	}
}
