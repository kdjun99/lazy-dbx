package app_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdjun99/lazy-dbx/internal/app"
	domainlogger "github.com/kdjun99/lazy-dbx/internal/domain/logger"
	"github.com/kdjun99/lazy-dbx/internal/domain/query"
)

// mockPool is a minimal Pool implementation for testing.
type mockPool struct {
	db  *sql.DB
	err error
}

func (m *mockPool) Get(_ string) (*sql.DB, error)          { return m.db, m.err }
func (m *mockPool) Put(_ string, _ *sql.DB) error          { return nil }
func (m *mockPool) Close(_ string) error                   { return nil }
func (m *mockPool) CloseAll() error                        { return nil }
func (m *mockPool) Ping(_ context.Context, _ string) error { return nil }

// noopLogger is a no-op logger for tests.
type noopLogger struct{}

func (l *noopLogger) Debug(_ context.Context, _, _, _ string, _ ...domainlogger.Field) {}
func (l *noopLogger) Info(_ context.Context, _, _, _ string, _ ...domainlogger.Field)  {}
func (l *noopLogger) Warn(_ context.Context, _, _, _ string, _ ...domainlogger.Field)  {}
func (l *noopLogger) Error(_ context.Context, _, _, _ string, _ ...domainlogger.Field) {}

// TestQueryService_CompileTimeCheck verifies that QueryService implements Executor.
// The compile-time check is also in query_service.go via var _ query.Executor = (*QueryService)(nil).
func TestQueryService_CompileTimeCheck(t *testing.T) {
	pool := &mockPool{}
	svc := app.NewQueryService(pool, &noopLogger{}, 1000)
	require.NotNil(t, svc)

	// verify it satisfies the interface at runtime too
	var _ query.Executor = svc
}

// TestQueryService_PoolError verifies that pool errors are returned.
func TestQueryService_PoolError(t *testing.T) {
	pool := &mockPool{err: assert.AnError}
	svc := app.NewQueryService(pool, &noopLogger{}, 1000)

	result := svc.Execute(context.Background(), "group.sub.conn", "SELECT 1")
	assert.Error(t, result.Error)
}
