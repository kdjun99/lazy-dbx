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
		{"readonly mode", true, "[RO]"},
		{"readwrite mode", false, "[RW]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, FormatMode(tt.readonly))
		})
	}
}

func TestFormatMode_ConsistentWithModeLabel(t *testing.T) {
	assert.Equal(t, ModeLabel(true), FormatMode(true))
	assert.Equal(t, ModeLabel(false), FormatMode(false))
}

func TestNewStatusBar(t *testing.T) {
	sb := NewStatusBar()
	assert.NotNil(t, sb)
	assert.NotNil(t, sb.Widget())
}

func TestStatusBar_SetConnectionInfo_RendersContent(t *testing.T) {
	sb := NewStatusBar()
	sb.SetConnectionInfo("main-db (mysql)")
	text := sb.widget.GetText(false)
	assert.Contains(t, text, "main-db (mysql)")
}

func TestStatusBar_SetMode_RendersContent(t *testing.T) {
	sb := NewStatusBar()
	sb.SetMode(true)
	assert.Equal(t, "[RO]", sb.mode)

	sb.SetMode(false)
	assert.Equal(t, "[RW]", sb.mode)
}

func TestStatusBar_SetKeybindings_RendersContent(t *testing.T) {
	sb := NewStatusBar()
	sb.SetKeybindings("Ctrl+Q:Quit  Tab:Focus")
	text := sb.widget.GetText(false)
	assert.Contains(t, text, "Ctrl+Q:Quit")
}

func TestStatusBar_Clear_ClearsAllSections(t *testing.T) {
	sb := NewStatusBar()
	sb.SetConnectionInfo("test")
	sb.SetMode(true)
	sb.SetKeybindings("keys")
	sb.Clear()
	assert.Empty(t, sb.keybindings)
	assert.Empty(t, sb.connectionInfo)
	assert.Empty(t, sb.mode)
}

func TestStatusBar_Render_ComposesNonEmptySections(t *testing.T) {
	sb := NewStatusBar()
	sb.SetKeybindings("keys")
	sb.SetConnectionInfo("conn")
	sb.SetMode(true)
	text := sb.widget.GetText(false)
	assert.Equal(t, "keys | conn | [RO]", text)
}

func TestStatusBar_Render_SkipsEmptySections(t *testing.T) {
	sb := NewStatusBar()
	sb.SetMode(true)
	assert.Equal(t, "[RO]", sb.mode)
	text := sb.widget.GetText(false)
	assert.Equal(t, "[RO]", text)
}

func TestStatusBar_SetMessage_NormalText(t *testing.T) {
	sb := NewStatusBar()
	sb.SetMessage("Operation successful", false)
	text := sb.widget.GetText(false)
	assert.Equal(t, "Operation successful", text)
}

func TestStatusBar_SetMessage_ErrorShowsRed(t *testing.T) {
	sb := NewStatusBar()
	sb.SetMessage("connection refused", true)
	// Raw text includes tview color tags.
	rawText := sb.widget.GetText(false)
	assert.Contains(t, rawText, "[red]")
	assert.Contains(t, rawText, "connection refused")
}
