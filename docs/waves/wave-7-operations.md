# Wave 7: Operations — Scripts, Audit & Export

> Goal: daily workflow efficiency — save queries, track activity, export data.

## Features

### Query Scripts
| # | Feature | Description | Status |
|---|---------|-------------|--------|
| S1 | Save query | `Ctrl+S` to save to `~/.config/lazy-dbx/scripts/` | Not Started |
| S2 | Load query | `Ctrl+O` to load saved script | Not Started |
| S3 | Fuzzy search | `sahilm/fuzzy` for connection/script search | Not Started |
| S4 | Tag/group organization | Organize scripts by connection group or custom tags | Not Started |

### Audit Log
| # | Feature | Description | Status |
|---|---------|-------------|--------|
| A1 | Query audit log | Log all executions with timestamp/connection/user | Not Started |
| A2 | Log file path config | Configurable path (`~/.config/lazy-dbx/audit.log`) | Not Started |
| A3 | Retention period | `retention_days` based auto-cleanup | Not Started |

### Export
| # | Feature | Description | Status |
|---|---------|-------------|--------|
| R6 | CSV export | Export result set to CSV | Not Started |
| R7 | JSON export | Export result set to JSON | Not Started |
| R8 | TSV export | Export result set to TSV | Not Started |

## Implementation Notes

### Query Scripts (S1~S4)
- S1: prompt for filename/tag on save, store as `.sql` files
- S2: file picker with fuzzy search, load into current editor tab
- S3: fuzzy match across script names and tags
- S4: directory structure by group or flat with metadata header in SQL file

### Audit Log (A1~A3)
- A1: append-only log format: `[timestamp] [connection] [user] [query] [duration] [rows]`
- A2: default `~/.config/lazy-dbx/audit.log`, overridable in settings
- A3: on startup, prune entries older than `retention_days`

### Export (R6~R8)
- Export current result set from results panel
- Keybind or command to trigger export
- Prompt for output file path
- CSV: standard RFC 4180
- JSON: array of objects with column names as keys
- TSV: tab-separated

## Done Criteria
- [ ] Queries can be saved, tagged, and loaded via fuzzy search
- [ ] All executed queries are logged to audit file
- [ ] Old audit entries are automatically cleaned up
- [ ] Results can be exported to CSV, JSON, and TSV
