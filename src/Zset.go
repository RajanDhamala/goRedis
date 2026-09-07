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
		// always pointing to head when adding retriivng other we traverse via skip list to improve perforamnce
		GlobalZset[key] = &newNode
		return "zset init succesfuly", nil
	}

	// for now lets just do level 1 only
	NewNode := ZNode{
		Score:  intscore,
		Member: member,
		Next:   nil,
		Skip:   nil,
	}
	currentNode := data
	PrevNode := data
	for currentNode != nil {
		if intscore > currentNode.Score {
			// new score is greater so keep moving
			// current node so traverse more
			PrevNode = currentNode
			currentNode = currentNode.Next
			continue
		} else if intscore == currentNode.Score {
			// equal score so can postion anywhere near adjsent neighbour node with same score
			temp := currentNode.Next
			currentNode.Next = &NewNode
			NewNode.Next = temp
			break
		} else if intscore < currentNode.Score {
			// score is lesser than teh current node so as its a singly linked list we cant traverse backwards btw
			// we can use temp intermediate point to handel this case
			// two cases:
			// 1. currentNode is the head
			// 2. currentNode is somewhere in the middle

			// case 1
			if data == currentNode {
				// indicated that we are at inital point
				NewNode.Next = currentNode
				GlobalZset[key] = &NewNode
			} else {
				// case 2
				NewNode.Next = currentNode
				PrevNode.Next = &NewNode
			}
			break
		}
	}

	if currentNode == nil {
		PrevNode.Next = &NewNode
	}

	return "zset add succesfuly", nil
}

func TraverseZset(msg []string) (string, error) {
	key := msg[1]

	data, ok := GlobalZset[key]

	if !ok {
		return "", errors.New("invalid key")
	}

	currentNode := data

	fmt.Println("")
	for currentNode != nil {
		fmt.Print(currentNode.Member, "->")
		currentNode = currentNode.Next
	}

	return "", nil
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
