// Package mysql provides a MySQL database connector implementation.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	// MySQL driver registration
	_ "github.com/go-sql-driver/mysql"

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

// Connector implements connection.Connector for MySQL databases.
type Connector struct{}

// Connect establishes a MySQL connection using the provided entry and password.
// If tunnelState is non-nil, the connection routes through the tunnel's local port.
func (c *Connector) Connect(ctx context.Context, entry config.ConnectionEntry, password string, tunnelState *tunnel.State) (*sql.DB, error) {
	host, port := entry.Host, entry.Port
	if tunnelState != nil {
		host = tunnelState.LocalHost
		port = tunnelState.LocalPort
	}

	dsn := c.buildDSN(entry, password, tunnelState)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: mysql open %s:%d/%s: %v", connection.ErrConnectionFailed, host, port, entry.Database, err)
	}

	c.applyPoolConfig(db, entry)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("%w: mysql ping %s:%d/%s: %v", connection.ErrConnectionFailed, host, port, entry.Database, err)
	}

	return db, nil
}

// buildDSN constructs the MySQL DSN string.
func (c *Connector) buildDSN(entry config.ConnectionEntry, password string, tunnelState *tunnel.State) string {
	host := entry.Host
	port := entry.Port

	if tunnelState != nil {
		host = tunnelState.LocalHost
		port = tunnelState.LocalPort
	}

	timeout := entry.ConnectTimeout
	if timeout == "" {
		timeout = defaultConnectTimeout
	}

	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=%s&parseTime=true",
		entry.User, password, host, port, entry.Database, timeout)
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
