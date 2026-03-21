package logger_test

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainlogger "github.com/kdjun99/lazy-dbx/internal/domain/logger"
	infralogger "github.com/kdjun99/lazy-dbx/internal/infra/logger"
)

func TestJSONLogger_WritesJSONLines(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "debug.log")

	l, err := infralogger.New(logPath)
	require.NoError(t, err)

	ctx := context.Background()
	l.Info(ctx, "test-component", "test-action", "test detail")

	f, err := os.Open(logPath)
	require.NoError(t, err)
	defer f.Close() //nolint:errcheck

	scanner := bufio.NewScanner(f)
	require.True(t, scanner.Scan(), "expected at least one log line")

	var entry map[string]any
	require.NoError(t, json.Unmarshal(scanner.Bytes(), &entry))

	assert.Equal(t, "info", entry["level"])
	assert.Equal(t, "test-component", entry["component"])
	assert.Equal(t, "test-action", entry["action"])
	assert.Equal(t, "test detail", entry["detail"])
	assert.NotEmpty(t, entry["timestamp"])
}

func TestJSONLogger_AllLevels(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "debug.log")

	l, err := infralogger.New(logPath)
	require.NoError(t, err)

	ctx := context.Background()
	l.Debug(ctx, "comp", "act", "debug msg")
	l.Info(ctx, "comp", "act", "info msg")
	l.Warn(ctx, "comp", "act", "warn msg")
	l.Error(ctx, "comp", "act", "error msg")

	f, err := os.Open(logPath)
	require.NoError(t, err)
	defer f.Close() //nolint:errcheck

	scanner := bufio.NewScanner(f)
	expectedLevels := []string{"debug", "info", "warn", "error"}
	for i, expectedLevel := range expectedLevels {
		require.True(t, scanner.Scan(), "expected line %d", i)
		var entry map[string]any
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &entry))
		assert.Equal(t, expectedLevel, entry["level"], "line %d", i)
	}
}

func TestJSONLogger_ExtraFields(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "debug.log")

	l, err := infralogger.New(logPath)
	require.NoError(t, err)

	ctx := context.Background()
	l.Error(ctx, "comp", "act", "detail",
		domainlogger.F("error", "connection refused"),
		domainlogger.F("host", "localhost"),
	)

	f, err := os.Open(logPath)
	require.NoError(t, err)
	defer f.Close() //nolint:errcheck

	scanner := bufio.NewScanner(f)
	require.True(t, scanner.Scan())

	var entry map[string]any
	require.NoError(t, json.Unmarshal(scanner.Bytes(), &entry))

	assert.Equal(t, "connection refused", entry["error"])
	assert.Equal(t, "localhost", entry["host"])
}

func TestJSONLogger_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "nested", "dirs", "debug.log")

	l, err := infralogger.New(logPath)
	require.NoError(t, err)

	ctx := context.Background()
	l.Info(ctx, "comp", "act", "detail")

	_, err = os.Stat(logPath)
	assert.NoError(t, err, "log file should exist")
}

func TestJSONLogger_ImplementsInterface(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "debug.log")

	l, err := infralogger.New(logPath)
	require.NoError(t, err)

	// Compile-time check that *JSONLogger implements domain Logger interface
	var _ domainlogger.Logger = l
}
