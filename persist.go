package app

// Persisted is one region of the device's store. The runtime places
// regions, keeps two copies of each (A/B) and loads them in registration
// order.
type Persisted struct {
	Name   string
	Blocks int // one copy, in 512-byte blocks

	// LegacyOff pins a region to a fixed offset, for stores written before
	// the allocator. New regions leave it -1.
	LegacyOff int

	Marshal   func() ([]byte, error)
	Unmarshal func([]byte) error // apply into live state under the owner's lock
	Validate  func() error       // pre-save gate; nil means always saveable
}
