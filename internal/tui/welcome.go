package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// NewWelcomePanel creates the welcome text panel shown in the right panel on startup.
func NewWelcomePanel() *tview.TextView {
	text := `lazy-dbx — Terminal Database IDE

Keybindings:
  j/k     Navigate connections
  Enter   Connect / Expand
  Tab     Switch panel
  Ctrl+Q  Quit`

	tv := tview.NewTextView()
	tv.SetTitle("Welcome")
	tv.SetBorder(true)
	tv.SetTextAlign(tview.AlignCenter)
	tv.SetText(text)
	tv.SetTextColor(tcell.ColorDefault)

	return tv
}
