# lazy-dbx Roadmap

> Implementation tracking for all 57 features across 7 waves.

## Progress Overview

| Wave | Theme | Features | Status |
|------|-------|----------|--------|
| [Wave 1](waves/wave-1-foundation.md) | Foundation — Config + DB Connection | 15 | **Done** |
| [Wave 2](waves/wave-2-tui-shell.md) | TUI Shell — Layout & Navigation | 7 | **Done** |
| [Wave 3](waves/wave-3-query-loop.md) | Core Loop — Query & Results | 7 | Not Started |
| [Wave 4](waves/wave-4-catalog.md) | Schema Catalog — Browse & Preview | 5 | Not Started |
| [Wave 5](waves/wave-5-safety.md) | Safety — Production Protection | 13 | Not Started |
| [Wave 6](waves/wave-6-editor.md) | Editor — Highlighting & Completion | 7 | Not Started |
| [Wave 7](waves/wave-7-operations.md) | Operations — Scripts, Audit & Export | 10 | Not Started |

## Dependency Graph

```
Wave 1 (Foundation)
  └─► Wave 2 (TUI Shell)
        └─► Wave 3 (Core Query Loop)
              ├─► Wave 4 (Schema Catalog)
              │     └─► Wave 6 (Editor — uses catalog for autocomplete)
              └─► Wave 5 (Safety — needs query execution)
                    └─► Wave 7 (Operations — audit, export, scripts)
```

## Milestones

| Milestone | Wave | Description |
|-----------|------|-------------|
| First Connection | 1 | CLI connects to a real DB through SSH tunnel |
| First Screen | 2 | TUI launches with connection tree |
| Minimum Usable | 3 | Query input → execute → view results |
| Schema Aware | 4 | Browse tables/columns in connected DB |
| Production Ready | 5 | Safe to use on production databases |
| Comfortable | 6 | Syntax highlighting, autocomplete, multi-tab |
| Complete | 7 | Scripts, audit log, export — fully featured |
