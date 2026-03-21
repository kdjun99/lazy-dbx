# Wave 2: TUI Shell — Layout & Navigation

> Goal: TUI launches, shows connection tree, user can select and connect to a DB.

## Features

| # | Feature | Description | Status |
|---|---------|-------------|--------|
| T1 | Base TUI frame | `tview` based panel layout shell | **Done** |
| T2 | Connections panel | Top-left connection tree view | **Done** |
| T6 | Status Bar | Bottom bar: keybindings, connection info, timing | **Done** |
| T7 | Tab panel switching | `Tab` key to move focus between panels | **Done** |
| T8 | Keyboard navigation | `j/k` tree navigation, `Enter` connect/expand | **Done** |
| T9 | Connection status indicator | Connected/disconnected icon in tree | **Done** |
| P10 | Connection type distinction | `read-only` / `read-write` visual label in UI | **Done** |

## Implementation Notes

### TUI Frame (T1)
- Use `rivo/tview` with Flex layout
- Initial layout: left panel (connections) + right placeholder + bottom status bar
- Query editor and results panels added in Wave 3

### Connections Panel (T2, T8, T9, P10)
- TreeView component from tview
- Nodes: group → subgroup → connection
- Icons: connection type (read-only/read-write) + status (connected/disconnected)
- Enter on connection node triggers Wave 1 connection logic

### Status Bar (T6)
- Show available keybindings for current context
- Show active connection name + environment tag
- Show read-only/read-write mode

## Done Criteria
- [x] `lazy-dbx` launches TUI with connection tree populated from config
- [x] User can navigate tree with j/k and connect with Enter
- [x] Status bar shows connection info and keybindings
- [x] Connection type (read-only/read-write) is visually distinct
