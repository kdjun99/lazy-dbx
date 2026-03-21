package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
)

func TestDispatchKeyAction(t *testing.T) {
	tests := []struct {
		name     string
		event    *tcell.EventKey
		expected string
	}{
		{
			name:     "CtrlQ quits",
			event:    tcell.NewEventKey(tcell.KeyCtrlQ, 0, tcell.ModNone),
			expected: "quit",
		},
		{
			name:     "CtrlC quits",
			event:    tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModNone),
			expected: "quit",
		},
		{
			name:     "Tab advances focus",
			event:    tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone),
			expected: "focus_next",
		},
		{
			name:     "Backtab retreats focus",
			event:    tcell.NewEventKey(tcell.KeyBacktab, 0, tcell.ModNone),
			expected: "focus_prev",
		},
		{
			name:     "Esc is no-op",
			event:    tcell.NewEventKey(tcell.KeyEsc, 0, tcell.ModNone),
			expected: "noop",
		},
		{
			name:     "unrecognized key returns empty",
			event:    tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DispatchKeyAction(tt.event)
			assert.Equal(t, tt.expected, got)
		})
	}
}
