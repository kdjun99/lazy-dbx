package connection

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainconn "github.com/kdjun99/lazy-dbx/internal/domain/connection"
)

// mockDriver is a minimal sql.Driver for testing that doesn't actually connect.
type mockDriver struct{}

type mockConn struct{}

func (m *mockConn) Prepare(query string) (driver.Stmt, error) { return nil, nil }
func (m *mockConn) Close() error                              { return nil }
func (m *mockConn) Begin() (driver.Tx, error)                 { return nil, nil }

func (m *mockDriver) Open(name string) (driver.Conn, error) {
	return &mockConn{}, nil
}

func init() {
	sql.Register("mock", &mockDriver{})
}

func openMockDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("mock", "mock://test")
	require.NoError(t, err)
	return db
}

func TestInMemoryPool_PutGetClose(t *testing.T) {
	pool := NewInMemoryPool()
	db := openMockDB(t)

	err := pool.Put("group.sub.conn", db)
	require.NoError(t, err)

	got, err := pool.Get("group.sub.conn")
	require.NoError(t, err)
	assert.Equal(t, db, got)

	err = pool.Close("group.sub.conn")
	require.NoError(t, err)

	_, err = pool.Get("group.sub.conn")
	assert.ErrorIs(t, err, domainconn.ErrConnectionNotInPool)
}

func TestInMemoryPool_GetNotFound(t *testing.T) {
	pool := NewInMemoryPool()

	_, err := pool.Get("nonexistent.path")
	assert.ErrorIs(t, err, domainconn.ErrConnectionNotInPool)
}

func TestInMemoryPool_CloseNotFound(t *testing.T) {
	pool := NewInMemoryPool()

	err := pool.Close("nonexistent.path")
	assert.ErrorIs(t, err, domainconn.ErrConnectionNotInPool)
}

func TestInMemoryPool_CloseAll(t *testing.T) {
	pool := NewInMemoryPool()

	db1 := openMockDB(t)
	db2 := openMockDB(t)

	require.NoError(t, pool.Put("a.b.c1", db1))
	require.NoError(t, pool.Put("a.b.c2", db2))

	err := pool.CloseAll()
	require.NoError(t, err)

	_, err = pool.Get("a.b.c1")
	assert.ErrorIs(t, err, domainconn.ErrConnectionNotInPool)

	_, err = pool.Get("a.b.c2")
	assert.ErrorIs(t, err, domainconn.ErrConnectionNotInPool)
}

func TestInMemoryPool_Ping(t *testing.T) {
	pool := NewInMemoryPool()
	db := openMockDB(t)

	require.NoError(t, pool.Put("a.b.c", db))

	ctx := context.Background()
	// mock driver doesn't implement Pinger, so PingContext may fail
	// but we mainly verify the path lookup works
	err := pool.Ping(ctx, "a.b.c")
	// err could be nil or driver-related; we just verify it doesn't panic
	_ = err

	// Not found case should return ErrConnectionNotInPool
	err = pool.Ping(ctx, "nonexistent")
	assert.ErrorIs(t, err, domainconn.ErrConnectionNotInPool)
}

func TestInMemoryPool_Concurrent(t *testing.T) {
	pool := NewInMemoryPool()
	const numGoroutines = 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			path := "group.sub.conn"
			db := openMockDB(t)

			// Concurrent puts and gets
			_ = pool.Put(path, db)
			_, _ = pool.Get(path)
		}(i)
	}

	wg.Wait()
	// No race conditions or panics = success
}

func TestInMemoryPool_DoubleClose(t *testing.T) {
	pool := NewInMemoryPool()
	db := openMockDB(t)

	require.NoError(t, pool.Put("a.b.c", db))
	require.NoError(t, pool.Close("a.b.c"))

	// Second close should return not found error, not panic
	err := pool.Close("a.b.c")
	assert.ErrorIs(t, err, domainconn.ErrConnectionNotInPool)
}
