// Package logger defines the structured logging interface for the domain layer.
package logger

import "context"

// Level represents the severity of a log entry.
type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// Field is a key-value pair for structured log data.
type Field struct {
	Key   string
	Value any
}

// F creates a Field with the given key and value.
func F(key string, value any) Field {
	return Field{Key: key, Value: value}
}

// Logger is the structured logging interface shared across domain and infra layers.
// Implementations write JSON-line entries to a configurable sink.
type Logger interface {
	Debug(ctx context.Context, component, action, detail string, fields ...Field)
	Info(ctx context.Context, component, action, detail string, fields ...Field)
	Warn(ctx context.Context, component, action, detail string, fields ...Field)
	Error(ctx context.Context, component, action, detail string, fields ...Field)
}
