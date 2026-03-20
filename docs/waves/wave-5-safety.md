# Wave 5: Safety — Production Protection

> Goal: safe enough to use on production databases without fear.

## Features

### Environment & Access Control
| # | Feature | Description | Status |
|---|---------|-------------|--------|
| P1 | Environment banner | Red warning bar on production connection | Not Started |
| P3 | Read-only default mode | Default read-only, toggle with `--write` or `Ctrl+R` | Not Started |
| P4 | Auto LIMIT append | Auto-append LIMIT to SELECT without one | Not Started |
| P5 | Connection lock | Optional PIN/confirmation for prod connection | Not Started |

### DML Protection
| # | Feature | Description | Status |
|---|---------|-------------|--------|
| P2 | DML confirmation | UPDATE/DELETE/DROP require explicit confirmation | Not Started |
| P6 | DML pre-query | SELECT affected rows before UPDATE/DELETE execution | Not Started |
| P7 | Before/after diff preview | Show data changes before execution | Not Started |
| P8 | Auto rollback data recording | Save pre-change snapshot to `~/.config/lazy-dbx/rollback/` | Not Started |
| P9 | Rollback execution | Generate and execute UNDO query from snapshot | Not Started |

### Transaction Control
| # | Feature | Description | Status |
|---|---------|-------------|--------|
| P11 | Auto/Manual Commit mode | Auto commit (default) ↔ Manual commit toggle | Not Started |
| P12 | Transaction status display | Status bar: open/closed transaction, uncommitted changes | Not Started |
| P13 | Transaction query log | View list of queries executed in current transaction | Not Started |
| P14 | Transaction ROLLBACK shortcut | Emergency rollback keybinding | Not Started |

## Implementation Notes

### Environment Safety (P1, P3, P4, P5)
- P1: check connection's `env` tag, render red banner across top when `production`
- P3: default `read_only_default = true` from settings, intercept DML in read-only mode
- P4: parse SELECT queries, append `LIMIT <auto_limit>` if missing
- P5: optional `lock = true` in connection config, prompt PIN before connect

### DML Protection (P2, P6~P9)
- P2: parse query type, show confirmation dialog for UPDATE/DELETE/DROP
- P6: extract WHERE clause, run `SELECT * FROM <table> WHERE <condition>` first
- P7: show affected rows in a diff view (before state)
- P8: serialize pre-change data to JSON in `~/.config/lazy-dbx/rollback/<timestamp>_<table>.json`
- P9: generate reverse DML from snapshot (UPDATE back to original values, or INSERT for deleted rows)

### Transaction Control (P11~P14)
- P11: toggle auto commit on DB connection level (`SET autocommit = 0/1` for MySQL, `BEGIN` for PG)
- P12: track transaction state in connection wrapper, display in status bar
- P13: buffer executed queries when in manual commit mode, show in popup/panel
- P14: `Ctrl+Z` or dedicated key → immediate `ROLLBACK`

## Done Criteria
- [ ] Production connections show red warning banner
- [ ] DML queries require confirmation with affected rows preview
- [ ] Rollback data is automatically saved and can be restored
- [ ] Auto/Manual commit mode works with transaction status display
- [ ] Transaction query log is viewable during open transaction
