// Package apptest runs components on a host, the way the runtime runs them
// on a device: console commands dispatched with the same lock rules and
// argument handling, HTTP routes matched the same way and answered in the
// same shapes, store regions marshalled and restored. It needs no board and
// no licence, so a component's tests run in any Go toolchain.
//
//	h := apptest.New()
//	if err := h.Register(blinky.Component(app.Platform{Board: "test"})); err != nil {
//		t.Fatal(err)
//	}
//	out, err := h.Run("hello")
//
// What it does not model: the network stack's connection limits, streaming
// routes (Response.Conn is nil), authentication (a route's Role is not
// checked), panels' drawing, and where the store places regions.
package apptest

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/bmxos/app"
)

// Harness hosts components for a test.
type Harness struct {
	// Locked makes the console refuse what a locked device refuses:
	// Mutating commands, and QueryOnly commands given arguments.
	Locked bool

	// Identity is the authenticated name handlers see on requests.
	Identity string

	comps   []app.Component
	cmds    map[string]*app.Command
	routes  []app.Route
	regions map[string]app.Persisted
	started map[string]chan struct{}
}

// New returns an empty harness.
func New() *Harness {
	return &Harness{
		cmds:    map[string]*app.Command{},
		regions: map[string]app.Persisted{},
		started: map[string]chan struct{}{},
	}
}

// Register adds a component, refusing what the runtime refuses: a
// component without a name or registered twice, a command without a name
// or handler, a verb or alias already taken, a region without a name or
// size or registered twice, a route without a path or handler.
func (h *Harness) Register(c app.Component) error {
	if c.Name == "" {
		return errors.New("component with empty name")
	}
	if _, dup := h.started[c.Name]; dup {
		return fmt.Errorf("component %q registered twice", c.Name)
	}
	for i := range c.Commands {
		cmd := &c.Commands[i]
		if cmd.Name == "" || cmd.Run == nil {
			return fmt.Errorf("%s: command %q has no name or no Run", c.Name, cmd.Name)
		}
		for _, n := range append([]string{cmd.Name}, cmd.Aliases...) {
			key := strings.ToLower(n)
			if prev, dup := h.cmds[key]; dup {
				return fmt.Errorf("%s: command %q already registered by %q", c.Name, n, prev.Name)
			}
			h.cmds[key] = cmd
		}
	}
	for _, r := range c.Persisted {
		if r.Name == "" || r.Blocks <= 0 || r.Marshal == nil || r.Unmarshal == nil {
			return fmt.Errorf("%s: region %q needs a name, a size, Marshal and Unmarshal", c.Name, r.Name)
		}
		if _, dup := h.regions[r.Name]; dup {
			return fmt.Errorf("%s: region %q registered twice", c.Name, r.Name)
		}
		h.regions[r.Name] = r
	}
	for _, r := range c.Routes {
		if r.Path == "" || r.Serve == nil {
			return fmt.Errorf("%s: route %q has no path or no Serve", c.Name, r.Path)
		}
		h.routes = append(h.routes, r)
	}
	h.comps = append(h.comps, c)
	h.started[c.Name] = make(chan struct{})
	return nil
}

// Start runs every component's Start, in registration order, with the
// harness as its Host; then each PostLoad.
func (h *Harness) Start() error {
	for _, c := range h.comps {
		if c.Start != nil {
			if err := c.Start(h); err != nil {
				return fmt.Errorf("%s: start: %w", c.Name, err)
			}
		}
		close(h.started[c.Name])
	}
	for _, c := range h.comps {
		if c.PostLoad != nil {
			c.PostLoad()
		}
	}
	return nil
}

// Ready implements app.Host.
func (h *Harness) Ready(component string) <-chan struct{} {
	if ch, ok := h.started[component]; ok {
		return ch
	}
	ch := make(chan struct{})
	return ch // never closes: no such component
}

// Components implements app.Host.
func (h *Harness) Components() []string {
	var out []string
	for _, c := range h.comps {
		out = append(out, c.Name)
	}
	return out
}

// ErrLocked is returned for a command the locked console refuses.
var ErrLocked = errors.New("refused: the console is locked")

// Run dispatches one console line and returns what the command printed.
// Arguments are lower-cased unless the command is Raw, as on a device.
func (h *Harness) Run(line string) (string, error) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", nil
	}
	cmd, ok := h.cmds[strings.ToLower(fields[0])]
	if !ok {
		return "", fmt.Errorf("unknown command %q", fields[0])
	}
	args := fields[1:]
	if h.Locked && (cmd.Class == app.Mutating || (cmd.Class == app.QueryOnly && len(args) > 0)) {
		return "", ErrLocked
	}
	if !cmd.Raw {
		for i, a := range args {
			args[i] = strings.ToLower(a)
		}
	}
	var out strings.Builder
	cmd.Run(&app.CmdCtx{Line: line, Printf: func(f string, a ...any) { fmt.Fprintf(&out, f, a...) }}, args)
	return out.String(), nil
}

// Reply is one HTTP response a route wrote.
type Reply struct {
	Status      int
	ContentType string
	Body        []byte
}

// Error is the body WriteError sends.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Do sends a request to the matching route: an exact path and method first,
// then the path's catch-all (Method ""). A path no route serves is a 404,
// as on a device. The query string, if any, follows "?" in target.
func (h *Harness) Do(method, target, body string) Reply {
	path, query, _ := strings.Cut(target, "?")
	var route *app.Route
	for i := range h.routes {
		r := &h.routes[i]
		if r.Path != path {
			continue
		}
		if r.Method == method {
			route = r
			break
		}
		if r.Method == "" && route == nil {
			route = r
		}
	}
	var reply Reply
	write := func(status int, ctype string, b []byte) {
		if reply.Status == 0 {
			reply = Reply{Status: status, ContentType: ctype, Body: b}
		}
	}
	w := &app.Response{
		WriteJSON: func(doc any) {
			b, err := json.Marshal(doc)
			if err != nil {
				write(500, "text/plain", []byte(err.Error()+"\n"))
				return
			}
			write(200, "application/json", b)
		},
		WriteError: func(status int, code, msg string) {
			b, _ := json.Marshal(Error{code, msg})
			write(status, "application/json", b)
		},
		WriteHTML: func(status int, html string) { write(status, "text/html; charset=utf-8", []byte(html)) },
		WriteRaw:  func(status int, ctype string, b []byte) { write(status, ctype, b) },
	}
	if route == nil {
		w.WriteError(404, "not_found", "no route for "+path)
		return reply
	}
	limit := route.MaxBody
	if limit == 0 {
		limit = app.DefaultMaxBody
	}
	if len(body) > limit {
		w.WriteError(413, "too_large", fmt.Sprintf("body is %d bytes, the route accepts %d", len(body), limit))
		return reply
	}
	route.Serve(w, &app.Request{
		Method: method, Path: path, Query: query,
		Body: bufio.NewReader(strings.NewReader(body)), ContentLen: len(body),
		Identity: h.Identity,
	})
	if reply.Status == 0 {
		return Reply{Status: 500, ContentType: "text/plain", Body: []byte("the handler wrote no response\n")}
	}
	return reply
}

// Save is what the runtime stores for a region: Validate, then Marshal,
// refused if it outgrows the region's blocks.
func (h *Harness) Save(region string) ([]byte, error) {
	r, ok := h.regions[region]
	if !ok {
		return nil, fmt.Errorf("no region %q", region)
	}
	if r.Validate != nil {
		if err := r.Validate(); err != nil {
			return nil, fmt.Errorf("%s: not saveable: %w", region, err)
		}
	}
	b, err := r.Marshal()
	if err != nil {
		return nil, fmt.Errorf("%s: marshal: %w", region, err)
	}
	if max := r.Blocks * 512; len(b) > max {
		return nil, fmt.Errorf("%s: %d bytes do not fit its %d blocks (%d bytes)", region, len(b), r.Blocks, max)
	}
	return b, nil
}

// Load restores a region from what Save returned.
func (h *Harness) Load(region string, data []byte) error {
	r, ok := h.regions[region]
	if !ok {
		return fmt.Errorf("no region %q", region)
	}
	if err := r.Unmarshal(data); err != nil {
		return fmt.Errorf("%s: unmarshal: %w", region, err)
	}
	return nil
}
