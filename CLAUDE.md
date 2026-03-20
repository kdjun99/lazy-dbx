# lazy-dbx — Project Conventions

> Terminal-based Database IDE in Go. All AI agents MUST follow these conventions.

## Project Overview

- **Language**: Go 1.24
- **TUI**: `rivo/tview` | **Config**: `pelletier/go-toml/v2` | **CLI**: `spf13/cobra`
- **DB**: `go-sql-driver/mysql`, `lib/pq`
- **Roadmap**: `docs/ROADMAP.md` and `docs/waves/`

## Architecture: Clean Architecture (3-Layer)

```
TUI (internal/tui/)           → Domain only. Rendering + keybindings.
Domain (internal/domain/)     → stdlib only. Pure logic + interface definitions.
Infrastructure (internal/infra/) → Domain + third-party. Implements Domain interfaces.
```

### Dependency Rules (enforced by depguard lint)

| From → To | TUI | Domain | Infrastructure |
|-----------|-----|--------|----------------|
| **TUI** | — | OK | **DENIED** |
| **Domain** | **DENIED** | OK (peer) | **DENIED** |
| **Infrastructure** | **DENIED** | OK (implements interfaces) | OK (peer) |

### Principles
- **TUI**: imports Domain interfaces only. All rendering, no logic.
- **Domain**: defines interfaces. Zero external dependencies. This is the core.
- **Infrastructure**: implements Domain interfaces. Owns all external I/O.
- **DI**: `main.go` / `cmd/` assembles the app — creates Infra, injects into Domain/TUI.
- All cross-layer communication through interfaces, not concrete types.

## Error Handling: No Silent Failures

**Every user action MUST produce visible feedback. "Nothing happened" is a bug.**

### Result Pattern

Every service method returns a structured result with explicit error:

```go
type Result[T any] struct {
    Data    T
    Error   error
}
```

### Feedback Contract

TUI must handle all three states for every user action:

| State | UI Behavior |
|-------|-------------|
| **Loading** | Status bar shows spinner or "Executing..." |
| **Success** | Result displayed + status bar confirmation |
| **Error** | Error message in status bar + detail in log |

### Rules
- NEVER swallow errors with `_ = someFunc()` — comment why if ignored
- NEVER `if err != nil { return }` without feedback
- All errors must reach: (1) user-visible feedback, (2) debug log, or (3) both
- Panics are bugs. Recover at TUI boundary only as last resort, log stack trace

## Structured Logging

All domain and infra operations MUST write structured logs for debuggability.

- **File**: `~/.config/lazy-dbx/debug.log` | **Format**: JSON lines
- **Fields**: `timestamp`, `level`, `component`, `action`, `detail`, `error`
- **Levels**: `debug` (verbose), `info` (normal ops), `warn` (recoverable), `error` (failures)

### Rules
- Every service method logs at least one entry
- Error logs MUST include original error message + context
- AI agents: **read `debug.log` first** before modifying code when debugging

## Testing Strategy

### TDD Workflow (MANDATORY)

```
1. Write test first       → define expected behavior (Red)
2. Confirm test fails     → for the RIGHT reason
3. Implement minimal code → make the test pass (Green)
4. Refactor               → tests stay green
5. Run `make check`       → full verification
```

### Test Modification Policy

**Tests are the spec. Tests are sacred.**

| Situation | Action |
|-----------|--------|
| `make test` fails after code change | Fix **implementation code**, NOT the test |
| Test seems wrong | **Report to user**. Do NOT modify. |
| Adding new behavior | Write **new test first**, then implement |
| Refactoring | All existing tests must pass as-is |

### By Layer

| Layer | Test Type | Min Coverage |
|-------|-----------|-------------|
| Domain (`internal/domain/`) | Unit (testify, mockgen) | **90%** |
| Infra (`internal/infra/`) | Unit + Integration (testcontainers) | **70%** |
| TUI (`internal/tui/`) | Unit (extracted logic) | **50%** |
| Overall | | **80%** |

### Conventions
- Test files: `*_test.go` in same package
- Fixture data: `testdata/` directory within each package
- **Table-driven tests** for functions with multiple cases
- Mock interfaces, not concrete types (`go.uber.org/mock`)
- Assertions: `testify/assert`, `testify/require`
- Integration tests: `//go:build integration` tag, run via `make test-integration`
- Integration DB: testcontainers (MySQL, PostgreSQL, OpenSSH)

## Code Quality — Enforced by Tooling

### Key Commands

| Command | Purpose |
|---------|---------|
| `make check` | **Run before every commit.** fmt + lint + vet + test |
| `make test` | Unit tests |
| `make test-integration` | Integration tests (requires Docker) |
| `make test-coverage` | Coverage report |

### Lint Policy (see `.golangci.yml` for full config)

Zero warnings policy. Key enforced rules:
- **depguard**: layer dependency enforcement
- **errcheck** + **nilerr**: no silent failures
- **gosec**: security issues
- **gocyclo** (max 15) + **nestif** (max 4): complexity limits
- **goimports**: local imports grouped separately

### Import Ordering

```go
import (
    "fmt"                                                    // 1. stdlib
    "github.com/rivo/tview"                                  // 2. third-party
    "github.com/kdjun99/lazy-dbx/internal/domain/connection" // 3. local
)
```

### Naming
- Interfaces: verb/role-based (`Connector`, `CatalogProvider`, `PasswordResolver`)
- Implementations: noun-based (`MySQLConnector`, `TOMLConfigLoader`)
- Tests: `Test<Function>_<scenario>` (e.g., `TestParseConfig_MissingField`)

### Project Structure

```
lazy-dbx/
├── cmd/root.go                    # CLI entrypoint (cobra)
├── internal/
│   ├── tui/                       # TUI layer
│   ├── domain/                    # Domain layer
│   │   ├── connection/            │   ├── catalog/
│   │   ├── query/                 │   ├── safety/
│   │   ├── audit/                 │   └── script/
│   └── infra/                     # Infrastructure layer
│       ├── config/                │       ├── tunnel/
│       ├── mysql/                 │       ├── postgres/
│       ├── storage/               │       └── password/
├── docs/                          # DESIGN.md, ROADMAP.md, waves/
├── .golangci.yml                  # Lint config (depguard layer rules)
├── Makefile                       # build, test, lint commands
└── main.go
```

## Git Conventions

- Branch: `wave-<N>/<feature-short-name>` (e.g., `wave-1/config-loader`)
- Commits: conventional commits (`feat:`, `fix:`, `refactor:`, `test:`, `docs:`)
- One logical change per commit
- Update wave doc status when feature complete

## AI Agent Workflow

### Implementing a Feature
1. **Read** relevant wave doc (`docs/waves/wave-N-*.md`)
2. **Write tests first** (TDD Red)
3. **Confirm tests fail** for the right reason
4. **Implement** Domain → Infrastructure → TUI wiring
5. **Confirm tests pass** (TDD Green)
6. **Run `make check`** — fix implementation code only
7. **Update** wave doc status

### When `make test` Fails
1. Read the failure message
2. Fix **implementation code** — NEVER modify the failing test
3. If test seems wrong → **report to user**, wait for direction

### Debugging TUI Issues
1. Read `~/.config/lazy-dbx/debug.log` first
2. No log entry for action → TUI→Domain wiring problem
3. Log shows error but UI doesn't → Domain→TUI feedback path problem
