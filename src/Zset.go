package src

import (
	"errors"
	"fmt"
	"strconv"
)

type ZNode struct {
	Score  int
	Member string

	Next *ZNode
	Skip *ZNode
}

var GlobalZset = make(map[string]*ZNode)

// ZADD key score member
func ZADD(msg []string) (string, error) {
	key := msg[1]
	member := msg[3]

	intscore, err := strconv.Atoi(msg[2])
	if err != nil {
		fmt.Println("error while parsing score")
		return "", errors.New("error while parsing score")
	}

	data, ok := GlobalZset[key]
	if !ok {
		fmt.Println("key not found init the zset now")
		newNode := ZNode{
			Score:  intscore,
			Member: member,
			Next:   nil,
			Skip:   nil,
		}
		GlobalZset[key] = &newNode
		// always pointing to head when adding retriivng other we traverse via skip list to improve perforamnce
		GlobalZset[key] = &newNode
		return "zset init succesfuly", nil
	}

	// for now lets just do level 1 only
	newNode := ZNode{
		Score:  intscore,
		Member: member,
		Next:   nil,
		Skip:   nil,
	}
	currentNode := data
	prevNode := data
	for currentNode != nil {
		if intscore > currentNode.Score {
			// meaning we need to traverse more node ok but before traversing we will check with the next node data
			prevNode = currentNode
			currentNode = currentNode.Next
		} else if intscore == currentNode.Score {
			// meaning we has already reached the threesold point btw
			temp := currentNode.Next
			currentNode.Next = &newNode
			newNode.Next = temp
			break
		} else if intscore < currentNode.Score {
			// need to traverse back and insert may be do -1 and insert try?
			// but since its singly linked list we cant traverse back can use one preNode pointer that will be updated as we vist node but seems overkill

			// prevNode.Next = &newNode

			if currentNode == data {
				// meaning we are start
				newNode.Next = currentNode
				GlobalZset[key] = &newNode
			} else {
				prevNode.Next = &newNode
				newNode.Next = currentNode
			}

			break
		}
		if currentNode == nil {
			prevNode.Next = &newNode
			break
		}
	}

	return "zset add succesfuly", nil
}

// ZSCORE leaderboard alice

func ZSCORE(msg []string) (string, error) {
	key := msg[1]
	member := msg[2]

	data, ok := GlobalZset[key]

	if !ok {
		fmt.Println("member not found in map")
		return "", errors.New("member not found")
	}

	currentNode := data
	Intial := data

	for currentNode != nil {
		if currentNode.Member == member {
			strscore := strconv.Itoa(currentNode.Score)
			Printtraversal(Intial, currentNode)
			return strscore, nil
		}
		currentNode = currentNode.Next
	}

	return "member not found", nil
}

func Printtraversal(inital *ZNode, final *ZNode) {
	currentNode := inital

	fmt.Println("")
	for currentNode != nil {
		if currentNode.Member == final.Member {
			fmt.Print(currentNode.Member)
			return
		}
		fmt.Print(currentNode.Member + "->")
		currentNode = currentNode.Next
	}
}
