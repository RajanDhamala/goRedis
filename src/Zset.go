package src

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

type ZNode struct {
	Score  float64
	Member string
	Next   *ZNode
	Skip   [4]*ZNode
}

var GlobalZset = make(map[string]*ZNode)

// Keep the prototype linked-list representation; updates remove and reinsert a member.
func ZADD(msg []string) (int, error) {
	scores := make([]float64, (len(msg)-2)/2)
	for i := range scores {
		score, err := strconv.ParseFloat(msg[2+i*2], 64)
		if err != nil || math.IsNaN(score) {
			return 0, errors.New("value is not a valid float")
		}
		scores[i] = score
	}
	added := 0
	for i, score := range scores {
		key, member := msg[1], msg[3+i*2]
		head := GlobalZset[key]
		link := &head
		found := false
		for *link != nil {
			if (*link).Member == member {
				*link = (*link).Next
				found = true
				break
			}
			link = &(*link).Next
		}
		if !found {
			added++
		}
		link = &head
		for *link != nil && ((*link).Score < score || ((*link).Score == score && (*link).Member < member)) {
			link = &(*link).Next
		}
		*link = &ZNode{Score: score, Member: member, Next: *link}
		GlobalZset[key] = head
	}
	return added, nil
}
func ZSCORE(msg []string) (string, error) {
	for node := GlobalZset[msg[1]]; node != nil; node = node.Next {
		if node.Member == msg[2] {
			return strconv.FormatFloat(node.Score, 'g', -1, 64), nil
		}
	}
	return "", errors.New("member not found")
}
func TraverseZset(msg []string) (string, error) {
	var members []string
	for node := GlobalZset[msg[1]]; node != nil; node = node.Next {
		members = append(members, node.Member)
	}
	return strings.Join(members, "->"), nil
}
