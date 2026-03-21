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

// GetQueryAtCursor returns the SQL statement under the cursor.
// Statements are delimited by semicolons. If only one statement exists,
// the entire text is returned.
func (e *QueryEditor) GetQueryAtCursor() string {
	text := e.textArea.GetText()
	fromRow, fromCol, _, _ := e.textArea.GetCursor()
	return QueryAtPosition(text, fromRow, fromCol)
}

// QueryAtPosition extracts the SQL statement at the given cursor position.
// Statements are delimited by ';'. Pure function for testability.
func QueryAtPosition(text string, cursorRow, cursorCol int) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}

	// Convert row:col to byte offset.
	lines := strings.Split(text, "\n")
	offset := 0
	for i := 0; i < cursorRow && i < len(lines); i++ {
		offset += len(lines[i]) + 1 // +1 for newline
	}
	if cursorRow < len(lines) {
		col := cursorCol
		if col > len(lines[cursorRow]) {
			col = len(lines[cursorRow])
		}
		offset += col
	}

	// Find the statement boundaries around the cursor offset.
	// Split by ';' and find which segment contains the offset.
	pos := 0
	segments := splitStatements(text)
	for _, seg := range segments {
		segEnd := pos + len(seg.raw)
		if offset >= pos && offset <= segEnd {
			return strings.TrimSpace(seg.text)
		}
		pos = segEnd + 1 // +1 for the ';' delimiter
	}

	// Fallback: return entire text trimmed.
	return strings.TrimSpace(text)
}

type segment struct {
	text string // trimmed content
	raw  string // original content (preserves length for offset calculation)
}

// splitStatements splits SQL text by ';' into segments.
func splitStatements(text string) []segment {
	parts := strings.Split(text, ";")
	segments := make([]segment, len(parts))
	for i, p := range parts {
		segments[i] = segment{
			text: strings.TrimSpace(p),
			raw:  p,
		}
	}
	return segments
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
