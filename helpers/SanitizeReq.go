package helpers

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func ReadRESPFromFirstLine(reader *bufio.Reader, line string) ([]string, error) {
	line = strings.TrimSuffix(line, "\r\n")

	if !strings.HasPrefix(line, "*") {
		return nil, fmt.Errorf("expected RESP array")
	}

	count, err := strconv.Atoi(line[1:])
	if err != nil {
		return nil, err
	}

	args := make([]string, 0, count)

	for i := 0; i < count; i++ {
		lengthLine, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		lengthLine = strings.TrimSuffix(lengthLine, "\r\n")

		if !strings.HasPrefix(lengthLine, "$") {
			return nil, fmt.Errorf("expected bulk string")
		}

		length, err := strconv.Atoi(lengthLine[1:])
		if err != nil {
			return nil, err
		}

		data := make([]byte, length+2)

		_, err = io.ReadFull(reader, data)
		if err != nil {
			return nil, err
		}

		if data[length] != '\r' || data[length+1] != '\n' {
			return nil, fmt.Errorf("invalid RESP bulk string")
		}

		args = append(args, string(data[:length]))
	}

	return args, nil
}

func ReadCommand(reader *bufio.Reader) ([]string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	// for the production & redis client compablity
	if strings.HasPrefix(line, "*") {
		return ReadRESPFromFirstLine(reader, line)
	}

	// tesing
	line = strings.TrimSpace(line)

	if line == "" {
		return nil, nil
	}

	return strings.Fields(line), nil
}
