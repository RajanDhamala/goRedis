// hash has both key and value

package src

import (
	"errors"
	"fmt"
)

type HashStruct struct {
	data map[string]string
}

var GlobalHash = make(map[string]*HashStruct)

// hash Fetures

// #HSET Key Field Value
// # HLEN Key -> to find the length of item in hash
// # HGET Key Field -> gell all items form hash
// # HDEL Key Field -> can be array of fileds
// # HGETALL Key -> get all key value from hash
// # HEXITS Key Field -> check if the field exist in hash

// mutex locks will be added later for the concurrency safety
// curenntly the (data structure)/ds needs to work on devlopment later we can optmize & secure it.

func HSET(msg []string) (string, error) {
	fmt.Println("inseting item on stack btw")
	Rootkey := msg[1]
	key := msg[2]
	value := msg[3]

	_, ok := GlobalHash[Rootkey]

	if !ok {
		fmt.Println("hash map not found btw")
		// creating new hash as its not found on global hash serch
		temphash := make(map[string]string)
		data := HashStruct{
			temphash,
		}
		GlobalHash[Rootkey] = &data
		data.data[key] = value
		return "new hash map created succesfully", nil
	}

	return "key value inserted succesfully", nil
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

func HDEL(msg []string) (string, error) {
	key := msg[1]

	field := msg[2]

	data, ok := GlobalHash[key]

	if !ok {
		// meaning the hash not exist yet
		return "", errors.New("hash doesn't exist")
	}
	// not even cheking if the filed exist or not direct del
	// cause why check if exist or not as del does no-op when key is absent
	// think of it as tiny optimizaion preserve one req per del req
	delete(data.data, field)
	return "", nil
}

func HGETALL(msg []string) ([]string, error) {
	key := msg[1]
	data, ok := GlobalHash[key]

	temp := []string{}

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
