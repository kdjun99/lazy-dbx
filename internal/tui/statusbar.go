package tui

import (
	"fmt"
	"time"

	"github.com/rivo/tview"
)

// FormatConnectionInfo formats connection details into a display string.
func FormatConnectionInfo(name, dbType, env string) string {
	return fmt.Sprintf("%s (%s) [%s]", name, dbType, env)
}

// FormatMode returns the mode indicator string.
func FormatMode(readonly bool) string {
	if readonly {
		return "[RO]"
	}
	return "[RW]"
}

// StatusBar is the bottom status bar component.
type StatusBar struct {
	widget         *tview.TextView
	app            *tview.Application
	keybindings    string
	connectionInfo string
	mode           string
}

// NewStatusBar creates a new StatusBar.
func NewStatusBar() *StatusBar {
	widget := tview.NewTextView()
	widget.SetDynamicColors(true)

	return &StatusBar{
		widget: widget,
	}
}

// SetApp sets the tview application for QueueUpdateDraw.
func (s *StatusBar) SetApp(app *tview.Application) {
	s.app = app
}

// SetMessage displays a message in the status bar.
// Errors are shown in red. Message auto-clears after 5 seconds.
func (s *StatusBar) SetMessage(msg string, isError bool) {
	if isError {
		s.widget.SetText(fmt.Sprintf("[red]%s[-]", msg))
	} else {
		s.widget.SetText(msg)
	}

	time.AfterFunc(5*time.Second, func() {
		if s.app != nil {
			s.app.QueueUpdateDraw(func() {
				s.render()
			})
		} else {
			s.render()
		}
	})
}

// SetConnectionInfo updates the center connection info section.
func (s *StatusBar) SetConnectionInfo(info string) {
	s.connectionInfo = info
	s.render()
}

// SetMode updates the right mode section.
func (s *StatusBar) SetMode(readonly bool) {
	s.mode = FormatMode(readonly)
	s.render()
}

// SetKeybindings updates the left keybindings section.
func (s *StatusBar) SetKeybindings(text string) {
	s.keybindings = text
	s.render()
}

// Clear resets all sections.
func (s *StatusBar) Clear() {
	s.keybindings = ""
	s.connectionInfo = ""
	s.mode = ""
	s.render()
}

// Widget returns the underlying tview widget.
func (s *StatusBar) Widget() *tview.TextView {
	return s.widget
}

// render composes all sections into one line.
func (s *StatusBar) render() {
	text := fmt.Sprintf("%s | %s | %s", s.keybindings, s.connectionInfo, s.mode)
	s.widget.SetText(text)
}
