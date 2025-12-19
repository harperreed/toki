# Toki Project Guidelines

## Go Patterns

### Config/Options Initialization

When creating config or options structs, prefer calling a `default*()` function over inline initialization:

```go
// GOOD - new fields with defaults are preserved
cfg := defaultConfig()
cfg.DeviceID = id

// BAD - easy to miss new fields during refactors
cfg := &Config{Server: "...", DeviceID: id}
```

This prevents silent regressions when new fields with meaningful defaults are added. The AutoSync field was silently dropped during a refactor because `InitConfig()` used inline initialization instead of `defaultConfig()`.

### Refactoring Discipline

Before batch-replacing patterns:

1. Understand the FULL context of each occurrence
2. Group occurrences by their surrounding context
3. Apply different fixes to different groups if needed
4. Don't assume all occurrences are identical

Example: `err :=` vs `err =` depends on whether `err` is already declared in scope. Batch-replacing without checking context causes cascading fix cycles.

## Scenario Testing

Scenario tests live in `.scratch/` (gitignored) and exercise the real binary with real dependencies.

Use `.scratch/scenario-template.sh` as a starting point for new scenarios.

Key isolation requirements:
- Set `XDG_CONFIG_HOME` and `XDG_DATA_HOME` before build
- Set `GIT_DIR` to non-existent path AFTER build (breaks VCS stamping otherwise)
- Toki's git detection can prompt for project creation - disable with GIT_DIR
