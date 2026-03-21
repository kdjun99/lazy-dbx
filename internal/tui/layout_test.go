package tui

import (
	"testing"

	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildLayout_NotNil(t *testing.T) {
	tree := tview.NewTreeView()
	editor := tview.NewTextView()
	results := tview.NewTextView()
	statusBar := tview.NewTextView()

	layout := BuildLayout(tree, editor, results, statusBar)
	require.NotNil(t, layout)
}

func TestBuildLayout_ItemCount(t *testing.T) {
	tree := tview.NewTreeView()
	editor := tview.NewTextView()
	results := tview.NewTextView()
	statusBar := tview.NewTextView()

	layout := BuildLayout(tree, editor, results, statusBar)
	// Outer flex has 2 items: top (horizontal flex) and bottom (statusBar)
	assert.Equal(t, 2, layout.GetItemCount())
}
