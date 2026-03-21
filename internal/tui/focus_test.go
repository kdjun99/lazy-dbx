package tui

import (
	"testing"

	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFocusCycle_Next(t *testing.T) {
	p1 := tview.NewTextView()
	p2 := tview.NewTextView()
	p3 := tview.NewTextView()

	fm := NewFocusManager(nil, p1, p2, p3)
	require.NotNil(t, fm)

	assert.Equal(t, p1, fm.Current())

	fm.Next()
	assert.Equal(t, p2, fm.Current())

	fm.Next()
	assert.Equal(t, p3, fm.Current())
}

func TestFocusCycle_Prev(t *testing.T) {
	p1 := tview.NewTextView()
	p2 := tview.NewTextView()
	p3 := tview.NewTextView()

	fm := NewFocusManager(nil, p1, p2, p3)
	require.NotNil(t, fm)

	assert.Equal(t, p1, fm.Current())

	fm.Prev()
	// Should wrap to last
	assert.Equal(t, p3, fm.Current())
}

func TestFocusCycle_Wrap(t *testing.T) {
	p1 := tview.NewTextView()
	p2 := tview.NewTextView()

	fm := NewFocusManager(nil, p1, p2)
	require.NotNil(t, fm)

	fm.Next()
	assert.Equal(t, p2, fm.Current())

	// Should wrap back to first
	fm.Next()
	assert.Equal(t, p1, fm.Current())

	// Prev from first should wrap to last
	fm.Prev()
	assert.Equal(t, p2, fm.Current())
}

func TestNewFocusManager_NoPanics(t *testing.T) {
	// No panels edge case
	fm := NewFocusManager(nil)
	require.NotNil(t, fm)
	assert.Nil(t, fm.Current())
}
