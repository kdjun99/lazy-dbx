package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kdjun99/lazy-dbx/internal/domain/config"
)

func TestEnvColor(t *testing.T) {
	tests := []struct {
		name string
		env  config.Environment
		want tcell.Color
	}{
		{"production", config.Production, tcell.ColorRed},
		{"staging", config.Staging, tcell.ColorYellow},
		{"test", config.Test, tcell.ColorGreen},
		{"unknown", config.Environment("dev"), tcell.ColorWhite},
		{"empty", config.Environment(""), tcell.ColorWhite},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EnvColor(tt.env))
		})
	}
}

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		name       string
		connected  bool
		connecting bool
		want       string
	}{
		{"connected", true, false, "●"},
		{"connecting", false, true, "◌"},
		{"disconnected", false, false, "○"},
		{"connected takes precedence over connecting", true, true, "●"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, StatusIcon(tt.connected, tt.connecting))
		})
	}
}

func TestModeLabel(t *testing.T) {
	tests := []struct {
		name     string
		readonly bool
		want     string
	}{
		{"readonly", true, "[RO]"},
		{"readwrite", false, "[RW]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ModeLabel(tt.readonly))
		})
	}
}
