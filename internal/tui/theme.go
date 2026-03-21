// Package tui provides terminal UI components for lazy-dbx.
package tui

import (
	"github.com/gdamore/tcell/v2"

	"github.com/kdjun99/lazy-dbx/internal/domain/config"
)

// EnvColor returns the display color for a given environment.
// Production=red, Staging=yellow, Test=green, default=white.
func EnvColor(env config.Environment) tcell.Color {
	switch env {
	case config.Production:
		return tcell.ColorRed
	case config.Staging:
		return tcell.ColorYellow
	case config.Test:
		return tcell.ColorGreen
	default:
		return tcell.ColorWhite
	}
}

// StatusIcon returns an icon string representing connection status.
// connected="●", connecting="◌", disconnected="○".
func StatusIcon(connected bool, connecting bool) string {
	if connected {
		return "●"
	}
	if connecting {
		return "◌"
	}
	return "○"
}

// ModeLabel returns the mode label string for a connection.
// readonly=>[RO], readwrite=>[RW].
func ModeLabel(readonly bool) string {
	if readonly {
		return "[RO]"
	}
	return "[RW]"
}
