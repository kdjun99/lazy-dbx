package password_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainlogger "github.com/kdjun99/lazy-dbx/internal/domain/logger"
	domainpw "github.com/kdjun99/lazy-dbx/internal/domain/password"
	infralog "github.com/kdjun99/lazy-dbx/internal/infra/logger"
	"github.com/kdjun99/lazy-dbx/internal/infra/password"
)

// --- CmdResolver ---

func TestCmdResolver_Resolve_Success(t *testing.T) {
	r := password.NewCmdResolver()
	cfg := domainpw.Config{
		ConnectionName: "test-conn",
		Cmd:            "echo supersecret",
	}

	got, err := r.Resolve(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, "supersecret", got)
}

func TestCmdResolver_Resolve_TrimsWhitespace(t *testing.T) {
	r := password.NewCmdResolver()
	cfg := domainpw.Config{
		ConnectionName: "test-conn",
		Cmd:            "printf '  mypassword\n'",
	}

	got, err := r.Resolve(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, "mypassword", got)
}

func TestCmdResolver_Resolve_CommandFailure(t *testing.T) {
	r := password.NewCmdResolver()
	cfg := domainpw.Config{
		ConnectionName: "test-conn",
		Cmd:            "exit 1",
	}

	_, err := r.Resolve(context.Background(), cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, domainpw.ErrPasswordResolution)
}

func TestCmdResolver_Resolve_ContextTimeout(t *testing.T) {
	r := password.NewCmdResolver()
	cfg := domainpw.Config{
		ConnectionName: "test-conn",
		Cmd:            "sleep 10",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := r.Resolve(ctx, cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, domainpw.ErrPasswordResolution)
}

// --- EnvResolver ---

func TestEnvResolver_Resolve_Set(t *testing.T) {
	r := password.NewEnvResolver()
	cfg := domainpw.Config{
		ConnectionName: "my-conn",
	}
	t.Setenv("DB_PASSWORD_MY_CONN", "envpassword")

	got, err := r.Resolve(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, "envpassword", got)
}

func TestEnvResolver_Resolve_HyphensToUnderscores(t *testing.T) {
	r := password.NewEnvResolver()
	cfg := domainpw.Config{
		ConnectionName: "my-prod-db",
	}
	t.Setenv("DB_PASSWORD_MY_PROD_DB", "hyphenpassword")

	got, err := r.Resolve(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, "hyphenpassword", got)
}

func TestEnvResolver_Resolve_Uppercase(t *testing.T) {
	r := password.NewEnvResolver()
	cfg := domainpw.Config{
		ConnectionName: "MixedCase",
	}
	t.Setenv("DB_PASSWORD_MIXEDCASE", "casepassword")

	got, err := r.Resolve(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, "casepassword", got)
}

func TestEnvResolver_Resolve_Unset(t *testing.T) {
	r := password.NewEnvResolver()
	cfg := domainpw.Config{
		ConnectionName: "no-such-conn-xyz-unique",
	}

	_, err := r.Resolve(context.Background(), cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, domainpw.ErrPasswordResolution)
}

func TestEnvResolver_Resolve_CustomEnvField(t *testing.T) {
	r := password.NewEnvResolver()
	cfg := domainpw.Config{
		ConnectionName: "test-conn",
		Env:            "MY_CUSTOM_VAR",
	}
	t.Setenv("MY_CUSTOM_VAR", "customvalue")

	got, err := r.Resolve(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, "customvalue", got)
}

// --- PromptResolver ---

func TestPromptResolver_Resolve_ReadsFromReader(t *testing.T) {
	reader := strings.NewReader("promptpassword\n")
	r := password.NewPromptResolver(reader)
	cfg := domainpw.Config{ConnectionName: "test-conn"}

	got, err := r.Resolve(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, "promptpassword", got)
}

func TestPromptResolver_Resolve_TrimsWhitespace(t *testing.T) {
	reader := strings.NewReader("  mypass  \n")
	r := password.NewPromptResolver(reader)
	cfg := domainpw.Config{ConnectionName: "test-conn"}

	got, err := r.Resolve(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, "mypass", got)
}

func TestPromptResolver_Resolve_EmptyInput(t *testing.T) {
	reader := strings.NewReader("")
	r := password.NewPromptResolver(reader)
	cfg := domainpw.Config{ConnectionName: "test-conn"}

	_, err := r.Resolve(context.Background(), cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, domainpw.ErrPasswordResolution)
}

// --- PlaintextResolver ---

func TestPlaintextResolver_Resolve_ReturnsValue(t *testing.T) {
	log := infralog.NewNopLogger()
	r := password.NewPlaintextResolver(log)
	cfg := domainpw.Config{
		ConnectionName: "test-conn",
		Plaintext:      "mypw",
	}

	got, err := r.Resolve(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, "mypw", got)
}

func TestPlaintextResolver_Resolve_LogsWarning(t *testing.T) {
	warnLog := &captureLogger{}
	r := password.NewPlaintextResolver(warnLog)
	cfg := domainpw.Config{
		ConnectionName: "test-conn",
		Plaintext:      "mypw",
	}

	_, err := r.Resolve(context.Background(), cfg)
	require.NoError(t, err)
	assert.True(t, warnLog.warnCalled, "expected Warn to be called")
}

func TestPlaintextResolver_Resolve_Empty(t *testing.T) {
	log := infralog.NewNopLogger()
	r := password.NewPlaintextResolver(log)
	cfg := domainpw.Config{
		ConnectionName: "test-conn",
		Plaintext:      "",
	}

	_, err := r.Resolve(context.Background(), cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, domainpw.ErrPasswordResolution)
}

// --- MultiResolver ---

func TestMultiResolver_Priority_CmdFirst(t *testing.T) {
	log := infralog.NewNopLogger()
	r := password.NewMultiResolver(log)

	cfg := domainpw.Config{
		ConnectionName: "test-conn",
		Cmd:            "echo cmdpassword",
		Plaintext:      "plaintextpassword",
	}

	got, err := r.Resolve(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, "cmdpassword", got)
}

func TestMultiResolver_Priority_EnvBeforePromptAndPlaintext(t *testing.T) {
	log := infralog.NewNopLogger()
	r := password.NewMultiResolver(log)

	cfg := domainpw.Config{
		ConnectionName: "env-test-multi",
		Plaintext:      "plaintextpassword",
	}
	t.Setenv("DB_PASSWORD_ENV_TEST_MULTI", "envpass")

	got, err := r.Resolve(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, "envpass", got)
}

func TestMultiResolver_Priority_PlaintextLast(t *testing.T) {
	log := infralog.NewNopLogger()
	r := password.NewMultiResolver(log)

	cfg := domainpw.Config{
		ConnectionName: "no-env-xyz-unique-multi",
		Plaintext:      "plaintextonly",
	}

	got, err := r.Resolve(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, "plaintextonly", got)
}

func TestMultiResolver_NoMethodConfigured_Error(t *testing.T) {
	log := infralog.NewNopLogger()
	r := password.NewMultiResolver(log)

	cfg := domainpw.Config{
		ConnectionName: "empty-conn-no-env-set",
	}

	_, err := r.Resolve(context.Background(), cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, domainpw.ErrNoPasswordMethod)
}

func TestMultiResolver_CmdFailure_HardError(t *testing.T) {
	log := infralog.NewNopLogger()
	r := password.NewMultiResolver(log)

	cfg := domainpw.Config{
		ConnectionName: "fail-conn",
		Cmd:            "exit 1",
		Plaintext:      "shouldnotbeused",
	}

	_, err := r.Resolve(context.Background(), cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, domainpw.ErrPasswordResolution)
}

// captureLogger records whether Warn was called.
type captureLogger struct {
	warnCalled bool
}

func (c *captureLogger) Debug(_ context.Context, _, _, _ string, _ ...domainlogger.Field) {}
func (c *captureLogger) Info(_ context.Context, _, _, _ string, _ ...domainlogger.Field)  {}
func (c *captureLogger) Warn(_ context.Context, _, _, _ string, _ ...domainlogger.Field) {
	c.warnCalled = true
}
func (c *captureLogger) Error(_ context.Context, _, _, _ string, _ ...domainlogger.Field) {}
