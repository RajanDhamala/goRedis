package helpers

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Prototype limits prevent invalid lengths from panicking or allocating unbounded memory.
const MaxCommandBytes = 64 * 1024 * 1024
const MaxArguments = 65536

func readLine(reader *bufio.Reader) (string, error) {
	var line []byte
	for {
		part, err := reader.ReadSlice('\n')
		if len(line)+len(part) > 65536 {
			return "", fmt.Errorf("line too long")
		}
		line = append(line, part...)
		if err == bufio.ErrBufferFull {
			continue
		}
		if err == io.EOF && len(line) > 0 {
			err = io.ErrUnexpectedEOF
		}
		return string(line), err
	}
}
func ReadRESPFromFirstLine(reader *bufio.Reader, line string) ([]string, error) {
	if !strings.HasSuffix(line, "\r\n") || !strings.HasPrefix(line, "*") {
		return nil, fmt.Errorf("expected RESP array")
	}
	count, err := strconv.Atoi(strings.TrimSuffix(line[1:], "\r\n"))
	if err != nil || count <= 0 || count > MaxArguments {
		return nil, fmt.Errorf("invalid array length")
	}
	args := make([]string, 0, count)
	total := 0
	for i := 0; i < count; i++ {
		lengthLine, err := readLine(reader)
		if err != nil {
			return nil, fmt.Errorf("incomplete bulk string header: %w", err)
		}
		if !strings.HasPrefix(lengthLine, "$") || !strings.HasSuffix(lengthLine, "\r\n") {
			return nil, fmt.Errorf("expected bulk string")
		}
		length, err := strconv.Atoi(strings.TrimSuffix(lengthLine[1:], "\r\n"))
		if err != nil || length < 0 || length > MaxCommandBytes-total {
			return nil, fmt.Errorf("invalid bulk string length")
		}
		total += length
		data := make([]byte, length+2)
		if _, err := io.ReadFull(reader, data); err != nil {
			return nil, fmt.Errorf("incomplete bulk string: %w", err)
		}
		if data[length] != '\r' || data[length+1] != '\n' {
			return nil, fmt.Errorf("invalid bulk string terminator")
		}
		args = append(args, string(data[:length]))
	}
	return args, nil
}
func ReadCommand(reader *bufio.Reader) ([]string, error) {
	line, err := readLine(reader)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(line, "*") {
		return ReadRESPFromFirstLine(reader, line)
	}
	// Retain simple, unquoted inline commands for manual prototype testing.
	return strings.Fields(line), nil
}
