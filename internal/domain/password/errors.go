package password

import "errors"

// Sentinel errors for password resolution.
var (
	// ErrPasswordResolution is returned when a configured password method fails.
	ErrPasswordResolution = errors.New("password resolution failed")
	// ErrNoPasswordMethod is returned when no password method is configured.
	ErrNoPasswordMethod = errors.New("no password method configured")
)
