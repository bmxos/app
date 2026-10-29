package app

import "time"

// Platform is what a component knows of the board and product it runs on.
// The generated main.go fills it from the board's seam.
type Platform struct {
	Product string    // the product name
	BuildID string    // version+commit, stamped by the harness
	Board   string    // the board's own description of itself
	Boot    time.Time // when main started

	// Board operations. Any may be nil where a board cannot do it;
	// components check before calling.
	LED      func(name string, on bool) error
	Reboot   func()
	PowerOff func()
}
