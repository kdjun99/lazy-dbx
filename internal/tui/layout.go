package tui

import (
	"github.com/rivo/tview"
)

// BuildLayout creates the main application layout.
// tree is placed on the left (fixed width 30), rightPanel fills remaining space,
// and statusBar is pinned to the bottom (fixed height 1).
func BuildLayout(tree tview.Primitive, rightPanel tview.Primitive, statusBar tview.Primitive) *tview.Flex {
	topFlex := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(tree, 30, 0, true).
		AddItem(rightPanel, 0, 1, false)

	mainFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(topFlex, 0, 1, true).
		AddItem(statusBar, 1, 0, false)

	return mainFlex
}
