package app

import "time"

// PanelPlace says where a panel appears.
type PanelPlace int

const (
	NavScreen PanelPlace = iota
	SettingsTile
)

// Panel is one UI surface: where it goes, what it is called, its lifecycle
// and its View.
type Panel struct {
	Title string
	Hint  string // settings-tile subtitle; unused for nav screens
	Place PanelPlace

	// View is the panel's content. The runtime does not draw: a renderer
	// package supplies the concrete views, so a headless product links no
	// GUI toolkit.
	View PanelView

	// Enter and Leave bracket visibility, for panels that hold a resource
	// (a camera stream) only while shown.
	Enter func()
	Leave func()

	// FrameEvery asks for a repaint cadence while the panel is visible.
	// Zero means the default 200ms.
	FrameEvery time.Duration
}

// PanelView is implemented by a renderer's view types. The method is a
// marker that keeps arbitrary values out of Panel.View.
type PanelView interface {
	PanelView()
}
