package src

import (
	"errors"
	"fmt"
	"strconv"
)

type ListNode struct {
	Data string
	Prev *ListNode
	Next *ListNode
}

type List struct {
	Head *ListNode
	Tail *ListNode
	Len  int
}

// no mutex locks and validation rn will add later

var GlobalList = make(map[string]*List)

// # List Features
// LPUSH key value -> add at front
// RPUSH key value -> add at back
// LPOP key -> remove + return first
// RPOP key -> remove + return last
// LRANGE key start stop -> get a range of items
// LLEN key -> get number of items in the list

// LPUSH key value -> add at front
func LPUSH(msg []string) (string, error) {
	key := msg[1]
	value := msg[2]

	data, ok := GlobalList[key]

	if !ok {
		// not found init list for user

		// create new doubly linked list node
		newNode := ListNode{
			Next: nil,
			Prev: nil,
			Data: value,
		}
		// stores the head node tail node and length info to avoid traversing whole linked list for length and push pop opertions
		// think of it as small optimization
		parentNode := List{
			Head: &newNode,
			Tail: &newNode,
			Len:  1,
		}
		GlobalList[key] = &parentNode
		return "list has been created succesfully", nil
	}

	newNode := ListNode{
		Prev: nil,
		Next: data.Head,
		Data: value,
	}
	data.Head.Prev = &newNode
	data.Head = &newNode
	data.Len++

	fmt.Println("lets start list")
	return "", nil
}

// RPUSH key value -> add at back
func RPUSH(msg []string) (string, error) {
	key := msg[1]
	value := msg[2]

	data, ok := GlobalList[key]

	if !ok {

		newNode := ListNode{
			Next: nil,
			Prev: nil,
			Data: value,
		}
		rootNode := List{
			Head: &newNode,
			Tail: &newNode,
			Len:  1,
		}
		GlobalList[key] = &rootNode
		return " lpush completed succesfully", nil
	}
	newNode := ListNode{
		Next: nil,
		Prev: data.Tail,
		Data: value,
	}

	data.Tail.Next = &newNode
	data.Tail = &newNode

	data.Len++

	return "", nil
}

// LPOP key -> remove + return first
func LPOP(msg []string) (string, error) {
	key := msg[1]

	data, ok := GlobalList[key]

	if !ok {
		return "", errors.New("list not found")
	}

	if data.Len == 1 {
		temp := data.Head
		delete(GlobalList, key)
		return temp.Data, nil
	}

	temp := data.Head
	data.Head = data.Head.Next
	data.Head.Prev = nil
	data.Len--

	return temp.Data, nil
}

// RPOP key -> remove + return last
func RPOP(msg []string) (string, error) {
	key := msg[1]

	data, ok := GlobalList[key]

	if !ok {
		return "", errors.New("list not found")
	}

	if data.Len == 1 {
		temp := data.Tail
		delete(GlobalList, key)
		return temp.Data, nil
	}

	temp := data.Tail
	data.Tail.Prev.Next = nil
	data.Tail = data.Tail.Prev
	data.Len--

	return temp.Data, nil
}

// LRANGE key start stop -> get a range of items
func LRANGE(msg []string) ([]string, error) {
	key := msg[1]
	start, err := strconv.Atoi(msg[2])
	if err != nil {
		return nil, errors.New("failed to parse start")
	}

	end, err := strconv.Atoi(msg[3])
	if err != nil {
		return nil, errors.New("failed to parse end")
	}
	// not validaitng it with the length of the list rn will add later

	temp := []string{}
	data, ok := GlobalList[key]

	if !ok {
		return nil, errors.New("list not found")
	}
	current := data.Head
	index := 0

	for current != nil {

		if index > end {
			// we need to return here
			break
		}

		if index >= start {
			// we need to add from here
			temp = append(temp, current.Data)
		}

		current = current.Next
		index++
	}

	return temp, nil
}

// Get number/length of items in the list
func LLEN(msg []string) (int, error) {
	key := msg[1]

	data, ok := GlobalList[key]

	if !ok {
		return 0, errors.New("list not found")
	}

	return data.Len, nil
}
