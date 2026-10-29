// Package app is the BMXOS application contract: what a component declares
// to the runtime that hosts it. It holds types only, depends on nothing but
// the standard library, and is MIT-licensed, so an application can be
// written and tested anywhere; package apptest runs components on a host.
//
// A component is a value, built by a constructor the product's generated
// main.go calls:
//
//	func Component(p app.Platform) app.Component
//
// The runtime (github.com/bmxos/sdk/app, licensed) registers components in
// order, owns boot, the console, the HTTP server, the panels and the store,
// and dispatches to what each component declared.
package app

// Component is everything one application module contributes to the
// platform's shared surfaces. Zero values mean "nothing on this surface"; a
// component may be as small as one console command.
type Component struct {
	Name string

	// System marks a platform component, one the runtime provides to every
	// application (storage, networking, diagnostics), as opposed to the
	// application's own vocabulary. Help lists the two groups apart.
	System bool

	Commands  []Command   // console verbs
	Persisted []Persisted // store regions
	Routes    []Route     // HTTP endpoints
	Panels    []Panel     // UI screens and settings tiles
	Features  []Feature   // capability flags

	// Start runs on the boot goroutine after platform bring-up and before
	// the store loads and the console loop. Spawn long-lived goroutines
	// here; do not block.
	Start func(h Host) error

	// PostLoad runs after this component's Persisted regions have loaded
	// (or defaulted), and after every earlier component's PostLoad:
	// registration order is load order.
	PostLoad func()
}

// Host is what a component's Start sees of the runtime hosting it.
type Host interface {
	// Ready returns a channel closed once the named component has
	// started, for a component that must wait on another.
	Ready(component string) <-chan struct{}

	// Components lists the registered components, in registration order.
	Components() []string
}

// Feature describes one capability flag. Its state is persisted by the
// runtime and switched from the console or the UI.
type Feature struct {
	Key   string // stable identifier, used in the store and on the console
	Title string // short label for the UI
	Desc  string // one or two sentences an operator can act on

	// Default is the state with nothing persisted.
	Default bool

	// Premium marks a capability intended to require a licence.
	Premium bool

	// Implemented is false while a flag exists ahead of the capability it
	// gates; the UI says so rather than offering a switch that does nothing.
	Implemented bool
}
