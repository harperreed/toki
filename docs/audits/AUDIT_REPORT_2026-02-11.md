# Documentation Audit Report
Generated: 2026-02-11 | Commit: e6fbe17

> **Status: RESOLVED** - All issues identified in this audit have been fixed.
> Vestigial sync MCP tools removed, stale docs deleted, paths/commands/URLs corrected.

## Executive Summary

| Metric | Count |
|--------|-------|
| Documents scanned | 6 |
| Claims verified | ~85 |
| Verified TRUE | ~62 (73%) |
| **Verified FALSE** | **23 (27%)** |
| Outdated/stale docs | 3 |

## False Claims Requiring Fixes

### README.md

| Line | Claim | Reality | Fix |
|------|-------|---------|-----|
| 11 | "SQLite storage" | SQLite OR Markdown backend (user-configurable since 5cdf069) | Update to mention both backends |
| 16 | `go install github.com/harper/toki/cmd/toki@latest` | Module path matches but actual repo is `harperreed/toki` | Verify if `harper` is a valid alias; otherwise update to `harperreed` |
| 22 | `git clone https://github.com/harper/toki` | Actual remote is `git@github.com:harperreed/toki.git` | Update to `https://github.com/harperreed/toki` |
| 119 | `"args": ["serve"]` in MCP config | Actual command is `toki mcp`, not `toki serve` | Change to `"args": ["mcp"]` |
| 128 | "**11 Tools**" | Actual count is **13** (includes `sync_status` and `sync_now` vestigial tools) or **11** if only counting functional tools (includes `delete_project` not in SKILL.md) | Clarify count; consider removing defunct sync tools |

### docs/MCP_USAGE.md

| Line | Claim | Reality | Fix |
|------|-------|---------|-----|
| 39 | `"args": ["serve"]` in config | Actual command is `toki mcp` | Change to `"args": ["mcp"]` |
| 54 | `toki mcp` for manual testing | Correct, but contradicts the `serve` arg on line 39 | Make consistent |
| 63 | Database at `$XDG_DATA_HOME/toki/toki.db` | Only true for SQLite backend; markdown backend uses directory structure | Add note about backend-dependent storage |
| 710-736 | Troubleshooting references `toki mcp` | Correct command, but the config examples above say `serve` | Already addressed above |

### docs/MCP_MANUAL_TESTING.md

| Line | Claim | Reality | Fix |
|------|-------|---------|-----|
| 36 | `"args": ["serve"]` in config | Actual command is `toki mcp` | Change to `"args": ["mcp"]` |
| 350 | Resource `toki://todos/all` | No such resource; should be `toki://todos` | Fix to `toki://todos` |
| 444 | `toki config` command | Command does not exist; use `toki setup` | Update troubleshooting |
| 446-449 | `ls -la ~/.config/toki/` and `sqlite3 ~/.config/toki/toki.db` | Database is at `~/.local/share/toki/toki.db`, not `~/.config/toki/` | Fix paths |
| 493 | "Default database location: `~/.config/toki/toki.db`" | Actual default: `~/.local/share/toki/toki.db` | Fix path |

### cmd/toki/skill/SKILL.md

| Line | Claim | Reality | Fix |
|------|-------|---------|-----|
| 19-30 | Lists 10 MCP tools | Actual count is 13 (missing `delete_project`, `sync_status`, `sync_now`) | Add missing tools or note they exist |
| 66 | `toki export markdown` CLI command | Exists but alias is `md` not documented | Minor - add alias note |

### CHARM_REMOVAL_PLAN.md

| Line | Claim | Reality | Fix |
|------|-------|---------|-----|
| 23 | `cmd/toki/sync.go` listed for modification | File was deleted entirely (commit f59fcc3) | Mark plan as completed/archived |
| 80-84 | Sync commands to be modified/removed | All sync commands removed; plan is stale | Mark as completed |
| 152-155 | `go.mod` should remove charmbracelet | charmbracelet deps remain (bubbles, bubbletea, lipgloss for TUI) | Update plan to note TUI deps are intentionally kept |

### docs/issue-badgerdb-concurrent-access.md

| Line | Claim | Reality | Fix |
|------|-------|---------|-----|
| All | Describes BadgerDB/Charm KV locking issue | Project migrated to SQLite; issue is resolved | Archive or add "RESOLVED" header |

## Pattern Summary

| Pattern | Count | Root Cause |
|---------|-------|------------|
| Wrong MCP command (`serve` vs `mcp`) | 3 | Docs written with planned command name, implementation used different name |
| Wrong database path (`~/.config/` vs `~/.local/share/`) | 3 | Confusion between XDG config dir and XDG data dir |
| Stale plan/issue docs | 3 | Plans executed but docs not updated afterward |
| Tool count mismatch | 3 | Vestigial sync tools added; SKILL.md and README not synced |
| Wrong repo URL | 1 | `harper` vs `harperreed` GitHub username |
| Non-existent CLI commands referenced | 2 | `toki serve` and `toki config` never implemented / removed |

## Undocumented Features (Gap Detection)

These features exist in the codebase but are NOT documented in any user-facing doc:

| Feature | Location | Notes |
|---------|----------|-------|
| `toki setup` command | cmd/toki/setup.go | Interactive TUI wizard for storage config |
| `toki migrate` command | cmd/toki/migrate.go | Migrate between SQLite and markdown backends |
| `toki project cleanup` | cmd/toki/project.go | Remove duplicate projects |
| `toki install-skill` | cmd/toki/skill/ | Install Claude Code skill |
| `toki version` | cmd/toki/version.go | Display version info |
| Markdown storage backend | internal/storage/markdown.go | Alternative to SQLite |
| `delete_project` MCP tool | internal/mcp/tools.go | Not in SKILL.md |
| `sync_status` MCP tool | internal/mcp/tools.go | Vestigial, returns "removed" message |
| `sync_now` MCP tool | internal/mcp/tools.go | Vestigial, returns "removed" message |

## Human Review Queue

- [ ] Verify if `github.com/harper/toki` is a valid Go module redirect for `github.com/harperreed/toki`
- [ ] Decide whether to remove vestigial `sync_status`/`sync_now` MCP tools or keep them
- [ ] Decide whether CHARM_REMOVAL_PLAN.md and issue-badgerdb-concurrent-access.md should be archived or deleted
- [ ] Decide if markdown backend should be documented in README (it's now the default for new users)

## Severity Classification

### Critical (will break user setup)
1. **MCP config uses `serve` instead of `mcp`** - Users following README will get a non-functional MCP server
2. **Wrong database path in troubleshooting** - Users looking at `~/.config/toki/` will find nothing

### Moderate (misleading but won't break anything)
3. **Tool count mismatch** - Minor confusion
4. **Stale plan/issue docs** - May confuse contributors
5. **Missing repo URL** - `harper` vs `harperreed`

### Low (cosmetic / nice-to-have)
6. **Undocumented commands** - Features exist but aren't discoverable via docs
7. **SQLite-only language in README** - Doesn't mention markdown backend
