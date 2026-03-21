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

// ConnectionStatus represents the state of a database connection.
type ConnectionStatus int

const (
	ConnectionStatusDisconnected ConnectionStatus = iota
	ConnectionStatusConnecting
	ConnectionStatusConnected
)

// StatusIcon returns an icon string representing connection status.
// connected="●", connecting="◌", disconnected="○".
func StatusIcon(status ConnectionStatus) string {
	switch status {
	case ConnectionStatusConnected:
		return "●"
	case ConnectionStatusConnecting:
		return "◌"
	default:
		return "○"
	}
}

// ModeLabel returns the mode label string for a connection.
// readonly=>[RO], readwrite=>[RW].
func ModeLabel(readonly bool) string {
	if readonly {
		return "[RO]"
	}
	return "[RW]"
}
