// Package tunnel defines domain interfaces and models for SSH tunnel management.
package tunnel

import "context"

// Tunneler opens and closes a single SSH port-forward tunnel.
type Tunneler interface {
	// Open establishes the SSH connection and starts a local listener that
	// forwards to remoteHost:remotePort through the SSH server described by
	// config. Returns a State with the chosen local port on success.
	Open(ctx context.Context, config Config, remoteHost string, remotePort int) (*State, error)
	// Close tears down the tunnel and releases all associated resources.
	Close() error
}

// Manager manages a pool of named tunnels with reference counting.
// Multiple callers requesting the same named tunnel share the same underlying
// connection; the tunnel is closed only when all callers have released it.
type Manager interface {
	// GetOrOpen returns an existing open tunnel for tunnelName or creates a new
	// one. Each successful call increments the internal reference count.
	GetOrOpen(ctx context.Context, tunnelName string, config Config, remoteHost string, remotePort int) (*State, error)
	// Release decrements the reference count for the named tunnel.
	// When the count reaches zero the tunnel is closed automatically.
	Release(tunnelName string) error
	// CloseAll forcibly closes every managed tunnel regardless of reference counts.
	CloseAll() error
}
