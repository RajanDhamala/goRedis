package helpers

import (
	"bufio"
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"
)

type byteReader struct{ io.Reader }

func (r byteReader) Read(p []byte) (int, error) { return r.Reader.Read(p[:1]) }

func TestRESPBinaryAndFragmentedPipeline(t *testing.T) {
	input := "*3\r\n$3\r\nSET\r\n$0\r\n\r\n$7\r\na\x00\r\n雪\r\n*1\r\n$4\r\nPING\r\n"
	reader := bufio.NewReader(byteReader{strings.NewReader(input)})
	for _, want := range [][]string{{"SET", "", "a\x00\r\n雪"}, {"PING"}} {
		got, err := ReadCommand(reader)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("got %q, %v; want %q", got, err, want)
		}
	}
	if _, err := ReadCommand(reader); err != io.EOF {
		t.Fatal(err)
	}
}
func TestRESPInvalidFrames(t *testing.T) {
	for _, input := range []string{"*-1\r\n", "*0\r\n", "*999999999999999999999\r\n", "*1\n$1\r\nx\r\n", "*1\r\n$-1\r\n", "*1\r\n$999999999999999999\r\n", "*1\r\n$67108865\r\n", "*1\r\n$1\r\nx!!", "*1\r\n$5\r\nhi", "*1\r\n:1\r\n"} {
		t.Run(input, func(t *testing.T) {
			if _, err := ReadCommand(bufio.NewReader(strings.NewReader(input))); err == nil {
				t.Fatal("accepted malformed frame")
			}
		})
	}
}
func TestRESPEncoding(t *testing.T) {
	tests := []struct {
		got  []byte
		want string
	}{
		{Simple("OK"), "+OK\r\n"}, {Error("ERR bad\r\ninput"), "-ERR bad  input\r\n"},
		{Integer(-2), ":-2\r\n"}, {Null(), "$-1\r\n"}, {Bulk(""), "$0\r\n\r\n"},
		{Bulk("雪"), "$3\r\n雪\r\n"}, {Array(), "*0\r\n"},
		{Array(Bulk("x"), Integer(2)), "*2\r\n$1\r\nx\r\n:2\r\n"},
	}
	for _, test := range tests {
		if !bytes.Equal(test.got, []byte(test.want)) {
			t.Fatalf("got %q want %q", test.got, test.want)
		}
	}
}
