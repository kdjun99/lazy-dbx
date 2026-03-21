// Package connection defines domain interfaces and models for database connection management.
package connection

import (
	"context"

	"github.com/kdjun99/lazy-dbx/internal/domain"
)

// Manager defines connection lifecycle operations for TUI consumption.
type Manager interface {
	Connect(ctx context.Context, path string) domain.Result[Info]
	Disconnect(ctx context.Context, path string) domain.Result[struct{}]
	Ping(ctx context.Context, path string) domain.Result[PingResult]
	ListConnections(ctx context.Context) domain.Result[[]Info]
	Shutdown(ctx context.Context)
}
