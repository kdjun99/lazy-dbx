package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// KeyAction maps a key event to a named action for dispatch logic.
type KeyAction struct {
	Key  tcell.Key
	Rune rune
	Name string
}

// RegisterKeybindings sets up global key capture on the tview application.
// Ctrl+Q quits, Ctrl+C cancels running query or quits, Tab/Backtab cycle focus,
// Ctrl+E executes query, Esc is consumed as no-op.
func RegisterKeybindings(app *tview.Application, focusMgr *FocusManager, quit func(), onExecute func(), onCancel func() bool) {
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlQ:
			quit()
			return nil
		case tcell.KeyCtrlC:
			if onCancel != nil && onCancel() {
				return nil
			}
			quit()
			return nil
		case tcell.KeyTab:
			focusMgr.Next()
			return nil
		case tcell.KeyBacktab:
			focusMgr.Prev()
			return nil
		case tcell.KeyCtrlE:
			onExecute()
			return nil
		case tcell.KeyEsc:
			return nil
		}
		return event
	})
}

// DispatchKeyAction resolves a key event to an action name for testing purposes.
// Returns empty string if no action is mapped.
func DispatchKeyAction(event *tcell.EventKey) string {
	switch event.Key() {
	case tcell.KeyCtrlQ:
		return "quit"
	case tcell.KeyCtrlC:
		return "cancel_or_quit"
	case tcell.KeyTab:
		return "focus_next"
	case tcell.KeyBacktab:
		return "focus_prev"
	case tcell.KeyCtrlE:
		return "execute"
	case tcell.KeyEsc:
		return "noop"
	}
	return ""
}
