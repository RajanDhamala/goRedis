package internal

import (
	"strconv"
	"strings"

	r "github.com/rajandhamala/goRedis/helpers"
	"github.com/rajandhamala/goRedis/src"
)

func (s *Server) hello(msg []string, client *src.Client) []byte {
	if len(msg) > 1 {
		version, err := strconv.Atoi(msg[1])
		if err != nil {
			return integerError()
		}
		if version != 2 {
			return r.Error("NOPROTO unsupported protocol version")
		}
	}
	// Validate the entire handshake before applying authentication or client metadata.
	username, password, name := "", "", ""
	hasAuth, hasName := false, false
	for i := 2; i < len(msg); {
		switch strings.ToUpper(msg[i]) {
		case "AUTH":
			if hasAuth || i+2 >= len(msg) {
				return r.Error("ERR syntax error in HELLO AUTH option")
			}
			username, password = msg[i+1], msg[i+2]
			hasAuth = true
			i += 3
		case "SETNAME":
			if hasName || i+1 >= len(msg) {
				return r.Error("ERR syntax error in HELLO SETNAME option")
			}
			name = msg[i+1]
			hasName = true
			i += 2
			if !validClientName(name) {
				return r.Error("ERR Client names cannot contain spaces or control characters")
			}
		default:
			return r.Error("ERR unsupported HELLO option")
		}
	}
	if hasAuth {
		if reply := s.authenticate(client, username, password); reply != nil {
			return reply
		}
	}
	if !client.Authenticated {
		return r.Error("NOAUTH Authentication required.")
	}
	if hasName {
		client.Name = name
	}

	return r.Array(r.Bulk("server"), r.Bulk("redis"), r.Bulk("version"), r.Bulk("0.1.0"), r.Bulk("proto"), r.Integer(2), r.Bulk("id"), r.Integer(client.ID), r.Bulk("mode"), r.Bulk("standalone"), r.Bulk("role"), r.Bulk("master"), r.Bulk("modules"), r.Array())
}
func clientCommand(msg []string, client *src.Client) []byte {
	switch strings.ToUpper(msg[1]) {
	case "SETINFO":
		if len(msg) == 4 && (strings.EqualFold(msg[2], "LIB-NAME") || strings.EqualFold(msg[2], "LIB-VER")) {
			return r.Simple("OK")
		}
	case "SETNAME":
		if len(msg) == 3 {
			if !validClientName(msg[2]) {
				return r.Error("ERR Client names cannot contain spaces or control characters")
			}
			client.Name = msg[2]
			return r.Simple("OK")
		}
	case "GETNAME":
		if len(msg) == 2 {
			if client.Name == "" {
				return r.Null()
			}
			return r.Bulk(client.Name)
		}
	case "ID":
		if len(msg) == 2 {
			return r.Integer(client.ID)
		}
	}
	return r.Error("ERR unknown CLIENT subcommand or wrong number of arguments")
}
func validClientName(name string) bool {
	for _, b := range []byte(name) {
		if b <= 32 || b >= 127 {
			return false
		}
	}
	return true
}
