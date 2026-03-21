package password

import (
	"context"
	"fmt"

	domainlogger "github.com/kdjun99/lazy-dbx/internal/domain/logger"
	domainpw "github.com/kdjun99/lazy-dbx/internal/domain/password"
)

// PlaintextResolver returns the plaintext password from the config.
// It logs a warning because storing passwords in config files is insecure.
type PlaintextResolver struct {
	log domainlogger.Logger
}

// NewPlaintextResolver creates a PlaintextResolver that logs warnings via log.
func NewPlaintextResolver(log domainlogger.Logger) *PlaintextResolver {
	return &PlaintextResolver{log: log}
}

// Resolve returns the plaintext password and logs a security warning.
func (r *PlaintextResolver) Resolve(ctx context.Context, config domainpw.Config) (string, error) {
	if config.Plaintext == "" {
		return "", fmt.Errorf("%w: plaintext password is empty for %q", domainpw.ErrPasswordResolution, config.ConnectionName)
	}

	r.log.Warn(ctx, "plaintext_resolver", "resolve",
		"plaintext password in config is insecure",
		domainlogger.F("connection", config.ConnectionName),
	)

	return config.Plaintext, nil
}
