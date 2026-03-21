package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryEditor_New(t *testing.T) {
	e := NewQueryEditor()
	require.NotNil(t, e)
	require.NotNil(t, e.Widget())
}

func TestQueryEditor_GetSetText(t *testing.T) {
	e := NewQueryEditor()
	e.SetText("SELECT 1")
	assert.Equal(t, "SELECT 1", e.GetText())
}

func TestQueryEditor_Clear(t *testing.T) {
	e := NewQueryEditor()
	e.SetText("SELECT 1")
	e.Clear()
	assert.Equal(t, "", e.GetText())
}

func TestQueryAtPosition_SingleStatement(t *testing.T) {
	text := "SELECT * FROM users"
	assert.Equal(t, "SELECT * FROM users", QueryAtPosition(text, 0, 5))
}

func TestQueryAtPosition_MultipleStatements_CursorOnFirst(t *testing.T) {
	text := "SELECT 1;\nSELECT 2;\nSELECT 3"
	// Cursor on row 0, col 3 → inside "SELECT 1"
	assert.Equal(t, "SELECT 1", QueryAtPosition(text, 0, 3))
}

func TestQueryAtPosition_MultipleStatements_CursorOnSecond(t *testing.T) {
	text := "SELECT 1;\nSELECT 2;\nSELECT 3"
	// Cursor on row 1, col 3 → inside "SELECT 2"
	assert.Equal(t, "SELECT 2", QueryAtPosition(text, 1, 3))
}

func TestQueryAtPosition_MultipleStatements_CursorOnThird(t *testing.T) {
	text := "SELECT 1;\nSELECT 2;\nSELECT 3"
	// Cursor on row 2, col 3 → inside "SELECT 3"
	assert.Equal(t, "SELECT 3", QueryAtPosition(text, 2, 3))
}

func TestQueryAtPosition_MultiLineStatement(t *testing.T) {
	text := "SELECT\n  *\nFROM users;\nSELECT 1"
	// Cursor on row 1 (inside first multi-line statement)
	assert.Equal(t, "SELECT\n  *\nFROM users", QueryAtPosition(text, 1, 0))
}

func TestQueryAtPosition_EmptyText(t *testing.T) {
	assert.Equal(t, "", QueryAtPosition("", 0, 0))
	assert.Equal(t, "", QueryAtPosition("   ", 0, 0))
}

func TestQueryAtPosition_TrailingSemicolon(t *testing.T) {
	text := "SELECT 1;"
	// Should return "SELECT 1" (trimmed, without semicolon)
	assert.Equal(t, "SELECT 1", QueryAtPosition(text, 0, 3))
}

func TestQueryAtPosition_SemicolonOnSameLine(t *testing.T) {
	text := "SELECT 1; SELECT 2"
	// Cursor at col 0 → inside "SELECT 1"
	assert.Equal(t, "SELECT 1", QueryAtPosition(text, 0, 0))
	// Cursor at col 12 → inside "SELECT 2"
	assert.Equal(t, "SELECT 2", QueryAtPosition(text, 0, 12))
}

func TestQueryAtPosition_CursorAtEnd(t *testing.T) {
	text := "SELECT 1;\nSELECT 2"
	// Cursor at end of text
	assert.Equal(t, "SELECT 2", QueryAtPosition(text, 1, 8))
}

func TestQueryAtPosition_WhitespaceAroundStatements(t *testing.T) {
	text := "  SELECT 1 ;  \n  SELECT 2  "
	assert.Equal(t, "SELECT 1", QueryAtPosition(text, 0, 5))
	assert.Equal(t, "SELECT 2", QueryAtPosition(text, 1, 5))
}
