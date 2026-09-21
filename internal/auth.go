package internal

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"strings"

	r "github.com/rajandhamala/goRedis/helpers"
	"github.com/rajandhamala/goRedis/src"
)

// Server's authentication configuration is immutable after construction.
// Only the default user is supported; this is shared-password auth, not Redis ACLs.
type Server struct {
	passwordHash [sha256.Size]byte
}

func NewServer(password string) (*Server, error) {
	if strings.TrimSpace(password) == "" {
		return nil, errors.New("REDIS_PASSWORD must be set to a non-empty password")
	}
	return &Server{passwordHash: sha256.Sum256([]byte(password))}, nil
}

func (s *Server) authenticate(client *src.Client, username, password string) []byte {
	candidate := sha256.Sum256([]byte(password))
	matches := subtle.ConstantTimeCompare(candidate[:], s.passwordHash[:])
	if username != "default" || matches != 1 {
		return r.Error("WRONGPASS invalid username-password pair or user is disabled.")
	}
	// CommandMu protects connection authentication state, just like command execution.
	client.Authenticated = true
	return nil
}

func (s *Server) authCommand(msg []string, client *src.Client) []byte {
	username, password := "default", msg[1]
	if len(msg) == 3 {
		username, password = msg[1], msg[2]
	}
	if reply := s.authenticate(client, username, password); reply != nil {
		return reply
	}
	return r.Simple("OK")
}
