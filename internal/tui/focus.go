package tui

import (
	"github.com/rivo/tview"
)

// FocusManager manages focus cycling between panels.
type FocusManager struct {
	panels  []tview.Primitive
	current int
	app     *tview.Application
}

// NewFocusManager creates a new FocusManager with the given panels.
func NewFocusManager(app *tview.Application, panels ...tview.Primitive) *FocusManager {
	return &FocusManager{
		panels:  panels,
		current: 0,
		app:     app,
	}
}

// Next advances focus to the next panel, wrapping around.
func (f *FocusManager) Next() {
	if len(f.panels) == 0 {
		return
	}
	f.current = (f.current + 1) % len(f.panels)
	f.setFocus()
}

// Prev moves focus to the previous panel, wrapping around.
func (f *FocusManager) Prev() {
	if len(f.panels) == 0 {
		return
	}
	f.current = (f.current - 1 + len(f.panels)) % len(f.panels)
	f.setFocus()
}

// Current returns the currently focused panel.
func (f *FocusManager) Current() tview.Primitive {
	if len(f.panels) == 0 {
		return nil
	}
	return f.panels[f.current]
}

func (f *FocusManager) setFocus() {
	if f.app != nil {
		f.app.SetFocus(f.panels[f.current])
	}
}
