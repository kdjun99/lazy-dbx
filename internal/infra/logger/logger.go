// Package logger provides a JSON structured logger implementing domain/logger.Logger.
package logger

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	domainlogger "github.com/kdjun99/lazy-dbx/internal/domain/logger"
)

// JSONLogger writes structured JSON log lines to a file.
type JSONLogger struct {
	mu   sync.Mutex
	file *os.File
}

// New creates a JSONLogger that writes to logPath.
// Parent directories are created if they do not exist.
func New(logPath string) (*JSONLogger, error) {
	if err := os.MkdirAll(filepath.Dir(logPath), 0o750); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o640)
	if err != nil {
		return nil, err
	}

	return &JSONLogger{file: f}, nil
}

// Debug logs a message at debug level.
func (l *JSONLogger) Debug(_ context.Context, component, action, detail string, fields ...domainlogger.Field) {
	l.write(domainlogger.LevelDebug, component, action, detail, fields)
}

// Info logs a message at info level.
func (l *JSONLogger) Info(_ context.Context, component, action, detail string, fields ...domainlogger.Field) {
	l.write(domainlogger.LevelInfo, component, action, detail, fields)
}

// Warn logs a message at warn level.
func (l *JSONLogger) Warn(_ context.Context, component, action, detail string, fields ...domainlogger.Field) {
	l.write(domainlogger.LevelWarn, component, action, detail, fields)
}

// Error logs a message at error level.
func (l *JSONLogger) Error(_ context.Context, component, action, detail string, fields ...domainlogger.Field) {
	l.write(domainlogger.LevelError, component, action, detail, fields)
}

// NopLogger is a no-op logger for use in tests.
type NopLogger struct{}

// NewNopLogger creates a NopLogger that discards all log output.
func NewNopLogger() *NopLogger { return &NopLogger{} }

func (n *NopLogger) Debug(_ context.Context, _, _, _ string, _ ...domainlogger.Field) {}
func (n *NopLogger) Info(_ context.Context, _, _, _ string, _ ...domainlogger.Field)  {}
func (n *NopLogger) Warn(_ context.Context, _, _, _ string, _ ...domainlogger.Field)  {}
func (n *NopLogger) Error(_ context.Context, _, _, _ string, _ ...domainlogger.Field) {}

func (l *JSONLogger) write(level domainlogger.Level, component, action, detail string, fields []domainlogger.Field) {
	entry := map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"level":     string(level),
		"component": component,
		"action":    action,
		"detail":    detail,
	}

	for _, f := range fields {
		entry[f.Key] = f.Value
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	data = append(data, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()
	//nolint:errcheck // best-effort log write; failure silently dropped to avoid recursive error handling
	l.file.Write(data)
}
