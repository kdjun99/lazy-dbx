package password

import "context"

// Resolver resolves a plaintext password from a Config.
// Implementations live in internal/infra/password/.
type Resolver interface {
	Resolve(ctx context.Context, config Config) (string, error)
}
