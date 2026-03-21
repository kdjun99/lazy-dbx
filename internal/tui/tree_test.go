package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdjun99/lazy-dbx/internal/domain/config"
)

func TestBuildTreeData_Empty(t *testing.T) {
	cfg := &config.ConnectionsConfig{
		Groups: map[string]*config.Group{},
	}
	nodes := BuildTreeData(cfg, false, nil)
	assert.Empty(t, nodes)
}

func TestBuildTreeData_SingleConnection(t *testing.T) {
	cfg := &config.ConnectionsConfig{
		Groups: map[string]*config.Group{
			"mygroup": {
				Subgroups: map[string]*config.Subgroup{
					"mysub": {
						Connections: map[string]config.ConnectionEntry{
							"myconn": {
								Name: "myconn",
								Env:  config.Production,
							},
						},
					},
				},
			},
		},
	}
	nodes := BuildTreeData(cfg, false, nil)
	require.Len(t, nodes, 1)
	assert.Equal(t, "mygroup", nodes[0].Label)
	assert.False(t, nodes[0].IsLeaf)

	require.Len(t, nodes[0].Children, 1)
	assert.Equal(t, "mysub", nodes[0].Children[0].Label)
	assert.False(t, nodes[0].Children[0].IsLeaf)

	require.Len(t, nodes[0].Children[0].Children, 1)
	leaf := nodes[0].Children[0].Children[0]
	assert.True(t, leaf.IsLeaf)
	assert.Equal(t, "mygroup.mysub.myconn", leaf.Path)
	assert.Equal(t, config.Production, leaf.Env)
	assert.Contains(t, leaf.Label, "myconn")
	assert.Contains(t, leaf.Label, "[production]")
	assert.Contains(t, leaf.Label, "[RW]")
	assert.Contains(t, leaf.Label, "○") // disconnected
}

func TestBuildTreeData_MultipleGroups_Sorted(t *testing.T) {
	cfg := &config.ConnectionsConfig{
		Groups: map[string]*config.Group{
			"zebra": {
				Subgroups: map[string]*config.Subgroup{
					"sub1": {
						Connections: map[string]config.ConnectionEntry{
							"c1": {Name: "c1"},
						},
					},
				},
			},
			"alpha": {
				Subgroups: map[string]*config.Subgroup{
					"sub1": {
						Connections: map[string]config.ConnectionEntry{
							"c1": {Name: "c1"},
						},
					},
				},
			},
		},
	}
	nodes := BuildTreeData(cfg, false, nil)
	require.Len(t, nodes, 2)
	assert.Equal(t, "alpha", nodes[0].Label)
	assert.Equal(t, "zebra", nodes[1].Label)
}

func TestBuildTreeData_ConnectedPaths(t *testing.T) {
	cfg := &config.ConnectionsConfig{
		Groups: map[string]*config.Group{
			"g": {
				Subgroups: map[string]*config.Subgroup{
					"s": {
						Connections: map[string]config.ConnectionEntry{
							"c": {Name: "c", Env: config.Test},
						},
					},
				},
			},
		},
	}
	connectedPaths := map[string]bool{"g.s.c": true}
	nodes := BuildTreeData(cfg, false, connectedPaths)
	require.Len(t, nodes, 1)
	leaf := nodes[0].Children[0].Children[0]
	assert.Contains(t, leaf.Label, "●") // connected icon
}

func TestBuildTreeData_ReadonlyMode(t *testing.T) {
	cfg := &config.ConnectionsConfig{
		Groups: map[string]*config.Group{
			"g": {
				Subgroups: map[string]*config.Subgroup{
					"s": {
						Connections: map[string]config.ConnectionEntry{
							"c": {Name: "c"},
						},
					},
				},
			},
		},
	}
	nodes := BuildTreeData(cfg, true, nil)
	require.Len(t, nodes, 1)
	leaf := nodes[0].Children[0].Children[0]
	assert.Contains(t, leaf.Label, "[RO]")
}

func TestBuildTreeData_SubgroupsSorted(t *testing.T) {
	cfg := &config.ConnectionsConfig{
		Groups: map[string]*config.Group{
			"g": {
				Subgroups: map[string]*config.Subgroup{
					"zzz": {
						Connections: map[string]config.ConnectionEntry{
							"c": {Name: "c"},
						},
					},
					"aaa": {
						Connections: map[string]config.ConnectionEntry{
							"c": {Name: "c"},
						},
					},
				},
			},
		},
	}
	nodes := BuildTreeData(cfg, false, nil)
	require.Len(t, nodes, 1)
	subs := nodes[0].Children
	require.Len(t, subs, 2)
	assert.Equal(t, "aaa", subs[0].Label)
	assert.Equal(t, "zzz", subs[1].Label)
}

func TestBuildTreeData_ConnectionsSorted(t *testing.T) {
	cfg := &config.ConnectionsConfig{
		Groups: map[string]*config.Group{
			"g": {
				Subgroups: map[string]*config.Subgroup{
					"s": {
						Connections: map[string]config.ConnectionEntry{
							"zzz": {Name: "zzz"},
							"aaa": {Name: "aaa"},
						},
					},
				},
			},
		},
	}
	nodes := BuildTreeData(cfg, false, nil)
	require.Len(t, nodes, 1)
	conns := nodes[0].Children[0].Children
	require.Len(t, conns, 2)
	assert.Contains(t, conns[0].Label, "aaa")
	assert.Contains(t, conns[1].Label, "zzz")
}
