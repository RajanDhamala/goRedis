// hash has both key and value

package src

import (
	"errors"
)

type HashStruct struct {
	data map[string]string
}

var GlobalHash = make(map[string]*HashStruct)

// note to add HICR

// hash Fetures

// #HSET Key Field Value
// # HLEN Key -> to find the length of item in hash
// # HGET Key Field -> gell all items form hash
// # HDEL Key Field -> can be array of fileds
// # HGETALL Key -> get all key value from hash
// # HEXITS Key Field -> check if the field exist in hash

// The command dispatcher holds CommandMu while accessing this collection.

func HSET(msg []string) (int, error) {
	key := msg[1]
	if GlobalHash[key] == nil {
		GlobalHash[key] = &HashStruct{data: make(map[string]string)}
	}
	hash := GlobalHash[key].data
	added := 0
	for i := 2; i < len(msg); i += 2 {
		if _, exists := hash[msg[i]]; !exists {
			added++
		}
		hash[msg[i]] = msg[i+1]
	}
	return added, nil
}

func HGET(msg []string) (string, error) {
	key := msg[1]
	field := msg[2]

	// no locks rn hence not concurrency safety
	value, ok := GlobalHash[key]
	if !ok {
		// user hash not found so we will just reurn btw
		return "", errors.New("Hash not found init first")
	}
	response, ok := value.data[field]
	if !ok {
		// meaning field not found inisde hash
		return "", errors.New("field not found in hash")
	}
	return response, nil
}

// To chek if the filed exist
func HEXISTS(msg []string) (bool, error) {
	key := msg[1]
	field := msg[2]
	// no validaiton for key field rn
	data, ok := GlobalHash[key]

	if !ok {
		// meaning hash doesnot exist or expired
		return false, errors.New("key not found")
	}

	_, ok = data.data[field]
	if !ok {
		// menaing field not found but hash exist
		return false, errors.New("field not found")
	}

	return true, nil
}

func HDEL(msg []string) (int, error) {
	hash := GlobalHash[msg[1]]
	if hash == nil {
		return 0, nil
	}
	removed := 0
	for _, field := range msg[2:] {
		if _, ok := hash.data[field]; ok {
			delete(hash.data, field)
			removed++
		}
	}
	if len(hash.data) == 0 {
		delete(GlobalHash, msg[1])
	}
	return removed, nil
}

func HGETALL(msg []string) ([]string, error) {
	key := msg[1]
	data, ok := GlobalHash[key]

	temp := []string{}

	// # HINCR update the value by converitng to int if fails skip cause we store all data as string rn

	if !ok {
		return temp, errors.New("hash not found ")
	}

	for field, value := range data.data {
		temp = append(temp, field, value)
	}
	return temp, nil
}

// # HLEN Key to find the length of item in hash
func HLEN(msg []string) (int, error) {
	key := msg[1]

	data, ok := GlobalHash[key]

	if !ok {
		return 0, errors.New("Hash not found")
	}
	length := len(data.data)

	return length, nil
}

// # HINCR update the value by converitng to int if fails skip cause we store all data as string rn
func HICR(msg []string) (int, error) {
	key := msg[1]

	data, ok := GlobalHash[key]

	if !ok {
		return 0, errors.New("Hash not found")
	}
	length := len(data.data)

	return length, nil
}
