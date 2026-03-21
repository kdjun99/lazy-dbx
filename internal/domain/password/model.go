// Package password defines domain interfaces and models for password resolution.
package password

// Method identifies which password source to use.
type Method int

const (
	// MethodCmd resolves password by executing a shell command.
	MethodCmd Method = iota
	// MethodEnv resolves password from an environment variable.
	MethodEnv
	// MethodPrompt resolves password by prompting the user interactively.
	MethodPrompt
	// MethodPlaintext uses a plaintext password stored in config (warns on use).
	MethodPlaintext
)

// Config holds all password-related fields from a ConnectionEntry.
type Config struct {
	// ConnectionName is used for env var lookup and log messages.
	ConnectionName string
	// Cmd is the shell command to execute (password_cmd field).
	Cmd string
	// Env is the env var name override (password_env field).
	Env string
	// Plaintext is the raw password string (password field).
	Plaintext string
}
