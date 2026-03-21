// Package connection provides an in-memory database connection pool implementation.
package connection

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	domainconn "github.com/kdjun99/lazy-dbx/internal/domain/connection"
)

// InMemoryPool implements domain/connection.Pool using a thread-safe in-memory map.
type InMemoryPool struct {
	mu    sync.RWMutex
	conns map[string]*sql.DB
}

// NewInMemoryPool creates a new empty InMemoryPool.
func NewInMemoryPool() *InMemoryPool {
	return &InMemoryPool{
		conns: make(map[string]*sql.DB),
	}
}

// Get retrieves an existing connection by its dot-separated path.
// Returns ErrConnectionNotInPool if the path is not found.
func (p *InMemoryPool) Get(path string) (*sql.DB, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	db, ok := p.conns[path]
	if !ok {
		return nil, fmt.Errorf("%w: %s", domainconn.ErrConnectionNotInPool, path)
	}

	return db, nil
}

// Put stores a connection under the given path.
func (p *InMemoryPool) Put(path string, db *sql.DB) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.conns[path] = db

	return nil
}

// Close closes and removes the connection at the given path.
func (p *InMemoryPool) Close(path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	db, ok := p.conns[path]
	if !ok {
		return fmt.Errorf("%w: %s", domainconn.ErrConnectionNotInPool, path)
	}

	delete(p.conns, path)

	return db.Close()
}

// CloseAll closes all managed connections, collecting any errors.
func (p *InMemoryPool) CloseAll() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error
	for path, db := range p.conns {
		if err := db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close %s: %w", path, err))
		}
		delete(p.conns, path)
	}

	if len(errs) > 0 {
		return fmt.Errorf("pool close errors: %v", errs)
	}

	return nil
}

// Ping verifies that the connection at the given path is alive.
func (p *InMemoryPool) Ping(ctx context.Context, path string) error {
	p.mu.RLock()
	db, ok := p.conns[path]
	p.mu.RUnlock()

	if !ok {
		return fmt.Errorf("%w: %s", domainconn.ErrConnectionNotInPool, path)
	}

	return db.PingContext(ctx)
}

// ensure InMemoryPool implements domainconn.Pool at compile time
var _ domainconn.Pool = (*InMemoryPool)(nil)
