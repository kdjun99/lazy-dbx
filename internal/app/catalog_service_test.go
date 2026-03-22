package app_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kdjun99/lazy-dbx/internal/app"
	"github.com/kdjun99/lazy-dbx/internal/domain/catalog"
)

// TestCatalogService_CompileCheck verifies CatalogService implements catalog.Provider.
func TestCatalogService_CompileCheck(t *testing.T) {
	t.Log("compile-time check: CatalogService implements catalog.Provider")
	// The var _ check in catalog_service.go already enforces this at compile time.
	// This test documents the contract.
	var _ catalog.Provider = (*app.CatalogService)(nil)
}

// TestCatalogService_ClearCache verifies that ClearCache removes cached entries.
func TestCatalogService_ClearCache(t *testing.T) {
	// We can't test full DB operations without a real DB, but we can verify
	// that ClearCache doesn't panic and that ListDatabases after clear returns an error
	// (no pool entry) rather than cached data.
	svc := app.NewCatalogService(nil, nil, func(_ string) string { return "mysql" })

	// Calling ClearCache on a non-existent path should not panic.
	assert.NotPanics(t, func() {
		svc.ClearCache("group.sub.conn")
	})

	// After clearing, ListDatabases with nil pool should return an error (no cache hit).
	result := svc.ListDatabases(context.Background(), "group.sub.conn")
	assert.Error(t, result.Error)

	// After clearing again, same behavior.
	svc.ClearCache("group.sub.conn")
	result2 := svc.ListDatabases(context.Background(), "group.sub.conn")
	assert.Error(t, result2.Error)
}
