package tunnel

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandTilde_NoTilde(t *testing.T) {
	assert.Equal(t, "/absolute/path", expandTilde("/absolute/path"))
	assert.Equal(t, "relative/path", expandTilde("relative/path"))
	assert.Equal(t, "", expandTilde(""))
}

func TestExpandTilde_TildeExpanded(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	result := expandTilde("~/.ssh/id_rsa")
	assert.Equal(t, filepath.Join(home, ".ssh/id_rsa"), result)
}

func TestExpandTilde_TildeOnly(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	result := expandTilde("~")
	assert.Equal(t, filepath.Join(home, ""), result)
}

func TestSSHTunneler_TildeKeyPath(t *testing.T) {
	// Write a valid key to a temp file, then construct a ~ path pointing to it.
	// We do this by symlinking the temp dir into $HOME-relative space isn't portable,
	// so instead verify expandTilde works end-to-end by writing to a temp file and
	// confirming the tilde path resolves to the same file content.
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	// Create a temp dir inside home so we can build a ~ path.
	tmpDir, err := os.MkdirTemp(home, "lazy-dbx-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	keyFile := filepath.Join(tmpDir, "id_rsa")
	require.NoError(t, os.WriteFile(keyFile, []byte("sentinel"), 0o600))

	// Build a ~ path: replace home prefix with ~.
	rel, err := filepath.Rel(home, keyFile)
	require.NoError(t, err)
	tildePath := "~/" + rel

	expanded := expandTilde(tildePath)
	assert.Equal(t, keyFile, expanded)

	// Confirm the expanded path reads the same content.
	data, err := os.ReadFile(expanded)
	require.NoError(t, err)
	assert.Equal(t, []byte("sentinel"), data)
}
