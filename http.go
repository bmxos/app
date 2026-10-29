package app

import (
	"bufio"
	"net"
	"time"
)

// Role is what an identity is permitted to do. Ordered, so a check can be
// "at least this role".
type Role uint8

const (
	RoleNone       Role = iota // the server default: RoleOperator
	RoleOperator               // runs the machine
	RoleMaintainer             // also changes its configuration
	RoleAdmin                  // also manages identities and firmware updates
)

// Route is one HTTP endpoint. The device's network stack serves few
// concurrent connections with small buffers, every response closes its
// connection, and there is no net/http: a streaming route holds a
// connection for its lifetime.
type Route struct {
	Path   string
	Method string // "GET", "POST", "PUT"; "" matches any method
	Role   Role   // RoleNone means the server default, RoleOperator

	// ReadTimeout extends the server's 5s default for routes that read for
	// longer, such as a firmware upload.
	ReadTimeout time.Duration

	// MaxBody is the largest request body the route accepts, checked
	// against Content-Length before the handler runs. Zero means
	// DefaultMaxBody.
	MaxBody int

	Serve func(w *Response, r *Request)
}

// DefaultMaxBody is the request body limit for routes that do not set one.
const DefaultMaxBody = 64 << 10

// Request is a parsed request.
type Request struct {
	Method     string
	Path       string
	Query      string // raw query string; empty when absent
	Body       *bufio.Reader
	ContentLen int

	// Identity is the authenticated identity's name, for auditing.
	Identity string
}

// Response writes one response. The server installs the writers;
// handlers that stream drop to Conn and own the socket.
type Response struct {
	Conn net.Conn

	WriteJSON  func(doc any)
	WriteError func(status int, code, msg string)
	WriteHTML  func(status int, body string)
	WriteRaw   func(status int, contentType string, body []byte)
}

// Write sends a complete response with an explicit content type, for the
// plain-text and binary cases the other writers do not cover.
func (w *Response) Write(status int, contentType string, body []byte) {
	w.WriteRaw(status, contentType, body)
}
