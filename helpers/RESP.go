package helpers

import (
	"strconv"
	"strings"
)

func Simple(value string) []byte { return []byte("+" + cleanLine(value) + "\r\n") }
func Error(value string) []byte  { return []byte("-" + cleanLine(value) + "\r\n") }
func Integer(value int64) []byte { return []byte(":" + strconv.FormatInt(value, 10) + "\r\n") }
func Null() []byte               { return []byte("$-1\r\n") }
func Bulk(value string) []byte {
	return []byte("$" + strconv.Itoa(len(value)) + "\r\n" + value + "\r\n")
}
func Array(values ...[]byte) []byte {
	result := []byte("*" + strconv.Itoa(len(values)) + "\r\n")
	for _, value := range values {
		result = append(result, value...)
	}
	return result
}
func Strings(values []string) []byte {
	items := make([][]byte, len(values))
	for i, value := range values {
		items[i] = Bulk(value)
	}
	return Array(items...)
}
func cleanLine(value string) string { return strings.NewReplacer("\r", " ", "\n", " ").Replace(value) }
