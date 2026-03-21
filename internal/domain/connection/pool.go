package connection

import (
	"context"
	"database/sql"
)

// Pool manages a collection of named database connections.
type Pool interface {
	// Get retrieves an existing connection by its dot-separated path.
	// Returns ErrConnectionNotInPool if the path is not found.
	Get(path string) (*sql.DB, error)
	// Put stores a connection under the given path.
	Put(path string, db *sql.DB) error
	// Close closes and removes the connection at the given path.
	Close(path string) error
	// CloseAll closes all managed connections.
	CloseAll() error
	// Ping verifies that the connection at the given path is alive.
	Ping(ctx context.Context, path string) error
}
