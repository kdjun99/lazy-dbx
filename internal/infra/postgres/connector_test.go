package postgres

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kdjun99/lazy-dbx/internal/domain/config"
	"github.com/kdjun99/lazy-dbx/internal/domain/tunnel"
)

func TestConnector_BuildDSN_NoTunnel(t *testing.T) {
	tests := []struct {
		name     string
		entry    config.ConnectionEntry
		password string
		wantDSN  string
	}{
		{
			name: "basic connection",
			entry: config.ConnectionEntry{
				Host:           "db.example.com",
				Port:           5432,
				User:           "admin",
				Database:       "mydb",
				ConnectTimeout: "10s",
			},
			password: "secret",
			wantDSN:  "postgres://admin:secret@db.example.com:5432/mydb?sslmode=disable&connect_timeout=10",
		},
		{
			name: "default timeout when empty",
			entry: config.ConnectionEntry{
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Database: "testdb",
			},
			password: "pass",
			wantDSN:  "postgres://postgres:pass@localhost:5432/testdb?sslmode=disable&connect_timeout=10",
		},
		{
			name: "custom timeout",
			entry: config.ConnectionEntry{
				Host:           "10.0.0.1",
				Port:           5433,
				User:           "user",
				Database:       "db",
				ConnectTimeout: "30s",
			},
			password: "pw",
			wantDSN:  "postgres://user:pw@10.0.0.1:5433/db?sslmode=disable&connect_timeout=30",
		},
	}

	c := &Connector{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn := c.buildDSN(tt.entry, tt.password, nil)
			assert.Equal(t, tt.wantDSN, dsn)
		})
	}
}

func TestConnector_BuildDSN_WithTunnel(t *testing.T) {
	entry := config.ConnectionEntry{
		Host:           "db.internal",
		Port:           5432,
		User:           "admin",
		Database:       "prod",
		ConnectTimeout: "10s",
	}
	tunnelState := &tunnel.State{
		LocalHost:  "127.0.0.1",
		LocalPort:  54321,
		RemoteHost: "db.internal",
		RemotePort: 5432,
		Status:     tunnel.StatusOpen,
	}

	c := &Connector{}
	dsn := c.buildDSN(entry, "secret", tunnelState)
	assert.Equal(t, "postgres://admin:secret@127.0.0.1:54321/prod?sslmode=disable&connect_timeout=10", dsn)
}

func TestConnector_BuildDSN_PoolConfig(t *testing.T) {
	entry := config.ConnectionEntry{
		Host:            "localhost",
		Port:            5432,
		User:            "root",
		Database:        "test",
		ConnectTimeout:  "10s",
		MaxOpenConns:    10,
		MaxIdleConns:    3,
		ConnMaxLifetime: "10m",
	}

	c := &Connector{}
	dsn := c.buildDSN(entry, "pass", nil)
	assert.NotEmpty(t, dsn)

	// Verify pool config values are used correctly
	assert.Equal(t, 10, entry.MaxOpenConns)
	assert.Equal(t, 3, entry.MaxIdleConns)
	assert.Equal(t, "10m", entry.ConnMaxLifetime)
}

func TestParseTimeoutSeconds(t *testing.T) {
	tests := []struct {
		name    string
		timeout string
		want    int
	}{
		{"empty uses default", "", 10},
		{"30s returns 30", "30s", 30},
		{"1m returns 60", "1m", 60},
		{"invalid falls back", "abc", 10},
		{"zero falls back", "0s", 10},
		{"negative falls back", "-5s", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTimeoutSeconds(tt.timeout)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestApplyPoolConfig(t *testing.T) {
	c := &Connector{}

	t.Run("default values applied when zero", func(t *testing.T) {
		db, err := sql.Open("postgres", "postgres://localhost:5432/test")
		assert.NoError(t, err)
		defer db.Close() //nolint:errcheck

		entry := config.ConnectionEntry{}
		c.applyPoolConfig(db, entry)

		assert.Equal(t, defaultMaxOpenConns, db.Stats().MaxOpenConnections)
	})

	t.Run("custom values applied", func(t *testing.T) {
		db, err := sql.Open("postgres", "postgres://localhost:5432/test")
		assert.NoError(t, err)
		defer db.Close() //nolint:errcheck

		entry := config.ConnectionEntry{
			MaxOpenConns:    10,
			MaxIdleConns:    3,
			ConnMaxLifetime: "10m",
		}
		c.applyPoolConfig(db, entry)

		assert.Equal(t, 10, db.Stats().MaxOpenConnections)
	})

	t.Run("invalid lifetime does not panic", func(t *testing.T) {
		db, err := sql.Open("postgres", "postgres://localhost:5432/test")
		assert.NoError(t, err)
		defer db.Close() //nolint:errcheck

		entry := config.ConnectionEntry{
			MaxOpenConns:    5,
			MaxIdleConns:    2,
			ConnMaxLifetime: "invalid",
		}
		assert.NotPanics(t, func() {
			c.applyPoolConfig(db, entry)
		})
	})
}
