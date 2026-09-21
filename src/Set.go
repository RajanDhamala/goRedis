// set has only one key we will use empty struct{} to save memory

package src

import (
	"errors"
)

type Set struct {
	Data map[string]struct{}
}

// The command dispatcher holds CommandMu while accessing this collection.

var GobalSet = make(map[string]*Set)

// # Set Features
// # SADD Key Member -> add members could be array of members
// # SREM Key Member -> remove members could be array of members
// # SISMEMBER Key Member -> chek if  member exist
// # SMEMBERS key -> get all members
// # SCARD key -> get no of members
//

// # SADD Key Member -> add members could be array of members
func SADD(msg []string) (int, error) {
	key := msg[1]
	if GobalSet[key] == nil {
		GobalSet[key] = &Set{Data: make(map[string]struct{})}
	}
	added := 0
	for _, member := range msg[2:] {
		if _, ok := GobalSet[key].Data[member]; !ok {
			GobalSet[key].Data[member] = struct{}{}
			added++
		}
	}
	return added, nil
}

func SREM(msg []string) (int, error) {
	set := GobalSet[msg[1]]
	if set == nil {
		return 0, nil
	}
	removed := 0
	for _, member := range msg[2:] {
		if _, ok := set.Data[member]; ok {
			delete(set.Data, member)
			removed++
		}
	}
	if len(set.Data) == 0 {
		delete(GobalSet, msg[1])
	}
	return removed, nil
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
