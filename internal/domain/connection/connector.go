package connection

import (
	"context"
	"database/sql"

	"github.com/kdjun99/lazy-dbx/internal/domain/config"
	"github.com/kdjun99/lazy-dbx/internal/domain/tunnel"
)

// Connector establishes a database connection from a ConnectionEntry.
// Password is pre-resolved and passed as an argument.
// TunnelState is non-nil when the connection should route through an SSH tunnel.
type Connector interface {
	Connect(ctx context.Context, entry config.ConnectionEntry, password string, tunnelState *tunnel.State) (*sql.DB, error)
}
