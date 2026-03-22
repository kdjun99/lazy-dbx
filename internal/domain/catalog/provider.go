package catalog

import (
	"context"

	"github.com/kdjun99/lazy-dbx/internal/domain"
)

// Provider is the interface for listing schema objects from a database connection.
type Provider interface {
	ListDatabases(ctx context.Context, path string) domain.Result[[]Database]
	ListTables(ctx context.Context, path string, database string) domain.Result[[]Table]
	ListColumns(ctx context.Context, path string, database string, table string) domain.Result[[]Column]
}
