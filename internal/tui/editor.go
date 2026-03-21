package tui

import (
	"strings"

	"github.com/rivo/tview"
)

// QueryEditor wraps a tview.TextArea for SQL input.
type QueryEditor struct {
	textArea *tview.TextArea
}

// NewQueryEditor creates a new QueryEditor with placeholder and border.
func NewQueryEditor() *QueryEditor {
	ta := tview.NewTextArea().
		SetPlaceholder("Type SQL here... (Ctrl+E to execute)")
	ta.SetTitle("Query Editor").SetBorder(true)

	return &QueryEditor{textArea: ta}
}

// GetText returns the current text content, trimmed of surrounding whitespace.
func (e *QueryEditor) GetText() string {
	return strings.TrimSpace(e.textArea.GetText())
}

// SetText sets the text content of the editor.
func (e *QueryEditor) SetText(s string) {
	e.textArea.SetText(s, true)
}

// Clear removes all text from the editor.
func (e *QueryEditor) Clear() {
	e.textArea.SetText("", true)
}

// Widget returns the underlying tview.Primitive for layout embedding.
func (e *QueryEditor) Widget() tview.Primitive {
	return e.textArea
}
