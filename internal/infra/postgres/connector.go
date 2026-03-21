// Package postgres provides a PostgreSQL database connector implementation.
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	// PostgreSQL driver registration
	_ "github.com/lib/pq"

	"github.com/kdjun99/lazy-dbx/internal/domain/config"
	"github.com/kdjun99/lazy-dbx/internal/domain/connection"
	"github.com/kdjun99/lazy-dbx/internal/domain/tunnel"
)

const (
	defaultConnectTimeout  = "10s"
	defaultMaxOpenConns    = 5
	defaultMaxIdleConns    = 2
	defaultConnMaxLifetime = "5m"
)

// Connector implements connection.Connector for PostgreSQL databases.
type Connector struct{}

// Connect establishes a PostgreSQL connection using the provided entry and password.
// If tunnelState is non-nil, the connection routes through the tunnel's local port.
func (c *Connector) Connect(ctx context.Context, entry config.ConnectionEntry, password string, tunnelState *tunnel.State) (*sql.DB, error) {
	dsn := c.buildDSN(entry, password, tunnelState)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: postgres open: %w", connection.ErrConnectionFailed, err)
	}

	c.applyPoolConfig(db, entry)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("%w: postgres ping: %w", connection.ErrConnectionFailed, err)
	}

	return db, nil
}

// buildDSN constructs the PostgreSQL DSN (connection URL).
func (c *Connector) buildDSN(entry config.ConnectionEntry, password string, tunnelState *tunnel.State) string {
	host := entry.Host
	port := entry.Port

	if tunnelState != nil {
		host = tunnelState.LocalHost
		port = tunnelState.LocalPort
	}

	timeoutSec := parseTimeoutSeconds(entry.ConnectTimeout)

	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable&connect_timeout=%d",
		entry.User, password, host, port, entry.Database, timeoutSec)
}

// applyPoolConfig sets connection pool parameters on the *sql.DB.
func (c *Connector) applyPoolConfig(db *sql.DB, entry config.ConnectionEntry) {
	maxOpen := entry.MaxOpenConns
	if maxOpen == 0 {
		maxOpen = defaultMaxOpenConns
	}
	maxIdle := entry.MaxIdleConns
	if maxIdle == 0 {
		maxIdle = defaultMaxIdleConns
	}
	lifetime := entry.ConnMaxLifetime
	if lifetime == "" {
		lifetime = defaultConnMaxLifetime
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)

	d, err := time.ParseDuration(lifetime)
	if err == nil {
		db.SetConnMaxLifetime(d)
	}
}

// parseTimeoutSeconds parses a duration string and returns the number of whole seconds.
// Falls back to 10 on parse error or if empty.
func parseTimeoutSeconds(timeout string) int {
	if timeout == "" {
		timeout = defaultConnectTimeout
	}
	d, err := time.ParseDuration(timeout)
	if err != nil {
		return 10
	}
	secs := int(d.Seconds())
	if secs <= 0 {
		return 10
	}
	return secs
}
