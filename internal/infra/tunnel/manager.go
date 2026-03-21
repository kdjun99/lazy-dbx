package tunnel

import (
	"context"
	"fmt"
	"sync"

	domaintunnel "github.com/kdjun99/lazy-dbx/internal/domain/tunnel"
)

type tunnelEntry struct {
	tunneler *SSHTunneler
	state    *domaintunnel.State
	refcount int
}

// TunnelManager manages shared SSH tunnels with reference counting.
type Manager struct {
	mu      sync.Mutex
	tunnels map[string]*tunnelEntry
}

// NewTunnelManager creates a new Manager.
func NewTunnelManager() *Manager {
	return &Manager{
		tunnels: make(map[string]*tunnelEntry),
	}
}

// GetOrOpen returns an existing tunnel or opens a new one, incrementing refcount.
func (m *Manager) GetOrOpen(ctx context.Context, tunnelName string, config domaintunnel.Config, remoteHost string, remotePort int) (*domaintunnel.State, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if entry, ok := m.tunnels[tunnelName]; ok {
		entry.refcount++
		return entry.state, nil
	}

	t := NewSSHTunneler()
	state, err := t.Open(ctx, config, remoteHost, remotePort)
	if err != nil {
		return nil, err
	}

	m.tunnels[tunnelName] = &tunnelEntry{
		tunneler: t,
		state:    state,
		refcount: 1,
	}
	return state, nil
}

// Release decrements the refcount; closes the tunnel when it reaches zero.
func (m *Manager) Release(tunnelName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.tunnels[tunnelName]
	if !ok {
		return nil
	}

	entry.refcount--
	if entry.refcount <= 0 {
		delete(m.tunnels, tunnelName)
		return entry.tunneler.Close()
	}
	return nil
}

// CloseAll forcibly closes all managed tunnels.
func (m *Manager) CloseAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for name, entry := range m.tunnels {
		if err := entry.tunneler.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing tunnel %q: %w", name, err))
		}
		delete(m.tunnels, name)
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
