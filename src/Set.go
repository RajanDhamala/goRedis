// set has only one key we will use empty struct{} to save memory

package src

import (
	"errors"
	"fmt"
)

type Set struct {
	Data map[string]struct{}
}

// no locks rn will add later for concurrency safety

var GobalSet = make(map[string]*Set)

// # Set Features
// # SADD Key Member -> add members could be array of members
// # SREM Key Member -> remove members could be array of members
// # SISMEMBER Key Member -> chek if  member exist
// # SMEMBERS key -> get all members
// # SCARD key -> get no of members
//

// # SADD Key Member -> add members could be array of members
func SADD(msg []string) (string, error) {
	key := msg[1]
	member := msg[2]

	fmt.Println("create set called", key)
	_, ok := GobalSet[key]

	if !ok {
		// meaning we dont have have key in global set  later will check ttl if expired or not
		// and hadnel accordingly rn just create new set
		hashmap := make(map[string]struct{})
		temp := Set{
			Data: hashmap,
		}
		hashmap[member] = struct{}{}
		GobalSet[key] = &temp
		// create new map and then just store it in global set with key added  and its value as struct{}
		return "new set created", nil
	}

	return "item added to set", nil
}

// # SREM Key Member -> remove members could be array of members
func SREM(msg []string) (bool, error) {
	// rn remove one member later mentioned members msg.lengh >1 are all members

	key := msg[1]
	member := msg[1]

	data, ok := GobalSet[key]
	if !ok {
		return false, errors.New("set not found")
	}

	delete(data.Data, member)
	return true, nil
}

// # SISMEMBER Key Member -> chek if  member exist
func SISMEMBER(msg []string) (bool, error) {
	key := msg[1]

	member := msg[2]

	ref, ok := GobalSet[key]

	if !ok {
		return false, errors.New("set not found")
	}

	_, ok = ref.Data[member]

	if !ok {
		return false, errors.New("set member not found")
	}
	return true, nil
}

// # SMEMBERS key -> get all members
func SMEMBERS(msg []string) ([]string, error) {
	key := msg[1]

	ref, ok := GobalSet[key]

	if !ok {
		return nil, errors.New("set not found")
	}
	temp := []string{}

	for member := range ref.Data {
		temp = append(temp, member)
	}

	return temp, nil
}

// # SCARD key -> get no of members
func SCARD(msg []string) (int, error) {
	key := msg[1]

	ref, ok := GobalSet[key]

	if !ok {
		return 0, errors.New("set not found")
	}
	length := len(ref.Data)
	return length, nil
}
