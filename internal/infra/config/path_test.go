package config_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	infraconfig "github.com/kdjun99/lazy-dbx/internal/infra/config"
)

func TestExpandPath_NoTilde(t *testing.T) {
	result, err := infraconfig.ExpandPath("/absolute/path/file.toml")
	require.NoError(t, err)
	assert.Equal(t, "/absolute/path/file.toml", result)
}

func TestExpandPath_TildeExpanded(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	result, err := infraconfig.ExpandPath("~/.ssh/id_ed25519")
	require.NoError(t, err)
	assert.Equal(t, home+"/.ssh/id_ed25519", result)
	assert.False(t, strings.HasPrefix(result, "~"))
}

func TestExpandPath_TildeOnly(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	result, err := infraconfig.ExpandPath("~")
	require.NoError(t, err)
	assert.Equal(t, home, result)
}

func TestExpandPath_EmptyPath(t *testing.T) {
	result, err := infraconfig.ExpandPath("")
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestExpandPath_RelativePath(t *testing.T) {
	result, err := infraconfig.ExpandPath("relative/path")
	require.NoError(t, err)
	assert.Equal(t, "relative/path", result)
}
