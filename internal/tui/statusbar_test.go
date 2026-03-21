package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatConnectionInfo(t *testing.T) {
	tests := []struct {
		name     string
		connName string
		dbType   string
		env      string
		expected string
	}{
		{
			name:     "all fields provided",
			connName: "main-db",
			dbType:   "mysql",
			env:      "production",
			expected: "main-db (mysql) [production]",
		},
		{
			name:     "different values",
			connName: "analytics",
			dbType:   "postgres",
			env:      "staging",
			expected: "analytics (postgres) [staging]",
		},
		{
			name:     "empty fields",
			connName: "",
			dbType:   "",
			env:      "",
			expected: " () []",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatConnectionInfo(tt.connName, tt.dbType, tt.env)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatMode(t *testing.T) {
	tests := []struct {
		name     string
		readonly bool
		expected string
	}{
		{
			name:     "readonly mode",
			readonly: true,
			expected: "[RO]",
		},
		{
			name:     "readwrite mode",
			readonly: false,
			expected: "[RW]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatMode(tt.readonly)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewStatusBar(t *testing.T) {
	sb := NewStatusBar()
	assert.NotNil(t, sb)
	assert.NotNil(t, sb.Widget())
}

func TestStatusBar_SetConnectionInfo(t *testing.T) {
	sb := NewStatusBar()
	sb.SetConnectionInfo("main-db (mysql) [production]")
	// Just verify it doesn't panic
}

func TestStatusBar_SetMode(t *testing.T) {
	sb := NewStatusBar()
	sb.SetMode(true)
	sb.SetMode(false)
	// Just verify it doesn't panic
}

func TestStatusBar_SetKeybindings(t *testing.T) {
	sb := NewStatusBar()
	sb.SetKeybindings("j/k Navigate | Tab Switch | Ctrl+Q Quit")
	// Just verify it doesn't panic
}

func TestStatusBar_Clear(t *testing.T) {
	sb := NewStatusBar()
	sb.SetConnectionInfo("test")
	sb.SetMode(true)
	sb.Clear()
	// Just verify it doesn't panic
}

func TestStatusBar_SetMessage(t *testing.T) {
	sb := NewStatusBar()
	sb.SetMessage("Operation successful", false)
	sb.SetMessage("Error occurred", true)
	// Just verify it doesn't panic (no real app for QueueUpdateDraw)
}
