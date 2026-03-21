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
