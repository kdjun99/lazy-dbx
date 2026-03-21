package query

import (
	"context"

	"github.com/kdjun99/lazy-dbx/internal/domain"
)

// Executor executes SQL statements against a named connection.
type Executor interface {
	Execute(ctx context.Context, path string, sql string) domain.Result[Result]
}
