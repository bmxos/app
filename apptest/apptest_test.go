package apptest

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/bmxos/app"
)

// counter is a small component touching every surface the harness models.
func counter(n *int) app.Component {
	return app.Component{
		Name: "counter",
		Commands: []app.Command{
			{Name: "count", Aliases: []string{"c"}, Class: app.QueryOnly, Run: func(ctx *app.CmdCtx, args []string) {
				if len(args) > 0 && args[0] == "reset" {
					*n = 0
				}
				ctx.Printf("count %d\n", *n)
			}},
			{Name: "bump", Run: func(ctx *app.CmdCtx, _ []string) { *n++ }},
			{Name: "echo", Raw: true, Class: app.ReadOnly, Run: func(ctx *app.CmdCtx, args []string) { ctx.Printf("%s", strings.Join(args, " ")) }},
		},
		Routes: []app.Route{
			{Path: "/count", Method: "GET", Serve: func(w *app.Response, r *app.Request) { w.WriteJSON(map[string]int{"count": *n}) }},
			{Path: "/count", Serve: func(w *app.Response, r *app.Request) { w.WriteError(405, "method", r.Method+" not allowed") }},
			{Path: "/whoami", Serve: func(w *app.Response, r *app.Request) { w.Write(200, "text/plain", []byte(r.Identity)) }},
		},
		Persisted: []app.Persisted{{Name: "counter", Blocks: 1,
			Marshal:   func() ([]byte, error) { return json.Marshal(*n) },
			Unmarshal: func(b []byte) error { return json.Unmarshal(b, n) },
		}},
	}
}

func TestConsole(t *testing.T) {
	var n int
	h := New()
	if err := h.Register(counter(&n)); err != nil {
		t.Fatal(err)
	}
	h.Run("bump")
	h.Run("BUMP")
	if out, _ := h.Run("C"); out != "count 2\n" {
		t.Errorf("alias, any case: %q", out)
	}
	if out, _ := h.Run("echo Keep THIS"); out != "Keep THIS" {
		t.Errorf("raw args lost their case: %q", out)
	}
	if _, err := h.Run("nope"); err == nil {
		t.Error("an unknown verb ran")
	}

	h.Locked = true
	if _, err := h.Run("bump"); !errors.Is(err, ErrLocked) {
		t.Errorf("mutating command ran while locked: %v", err)
	}
	if _, err := h.Run("count reset"); !errors.Is(err, ErrLocked) {
		t.Errorf("query-only command with arguments ran while locked: %v", err)
	}
	if out, err := h.Run("count"); err != nil || out != "count 2\n" {
		t.Errorf("bare query refused while locked: %q %v", out, err)
	}
}

func TestRoutes(t *testing.T) {
	n := 5
	h := New()
	h.Identity = "alice"
	if err := h.Register(counter(&n)); err != nil {
		t.Fatal(err)
	}
	if r := h.Do("GET", "/count", ""); r.Status != 200 || string(r.Body) != `{"count":5}` {
		t.Errorf("GET /count = %d %s", r.Status, r.Body)
	}
	if r := h.Do("POST", "/count", ""); r.Status != 405 {
		t.Errorf("catch-all not used: %d", r.Status)
	}
	if r := h.Do("GET", "/whoami?x=1", ""); string(r.Body) != "alice" || r.ContentType != "text/plain" {
		t.Errorf("identity or raw write lost: %+v", r)
	}
	if r := h.Do("GET", "/missing", ""); r.Status != 404 {
		t.Errorf("unrouted path = %d", r.Status)
	}
	if r := h.Do("GET", "/count", strings.Repeat("x", app.DefaultMaxBody+1)); r.Status != 413 {
		t.Errorf("oversized body = %d", r.Status)
	}
}

func TestPersistence(t *testing.T) {
	n := 41
	h := New()
	if err := h.Register(counter(&n)); err != nil {
		t.Fatal(err)
	}
	b, err := h.Save("counter")
	if err != nil {
		t.Fatal(err)
	}
	n = 0
	if err := h.Load("counter", b); err != nil || n != 41 {
		t.Errorf("round trip: n=%d err=%v", n, err)
	}
}

func TestRegisterRefuses(t *testing.T) {
	var n int
	h := New()
	if err := h.Register(counter(&n)); err != nil {
		t.Fatal(err)
	}
	if err := h.Register(app.Component{Name: "other", Commands: []app.Command{{Name: "C", Run: func(*app.CmdCtx, []string) {}}}}); err == nil {
		t.Error("a verb clashing with another component's alias was accepted")
	}
	if err := h.Register(app.Component{Name: "counter"}); err == nil {
		t.Error("a component registered twice was accepted")
	}
	if err := h.Register(app.Component{Name: "x", Persisted: []app.Persisted{{Name: "r"}}}); err == nil {
		t.Error("a region without a size was accepted")
	}
}

func TestStartAndHost(t *testing.T) {
	var order []string
	h := New()
	first := app.Component{Name: "first", Start: func(app.Host) error { order = append(order, "first"); return nil }}
	second := app.Component{Name: "second", Start: func(host app.Host) error {
		select {
		case <-host.Ready("first"):
		default:
			t.Error("first not ready when second starts")
		}
		order = append(order, strings.Join(host.Components(), "+"))
		return nil
	}}
	for _, c := range []app.Component{first, second} {
		if err := h.Register(c); err != nil {
			t.Fatal(err)
		}
	}
	if err := h.Start(); err != nil {
		t.Fatal(err)
	}
	if strings.Join(order, ",") != "first,first+second" {
		t.Errorf("start order %v", order)
	}
}
