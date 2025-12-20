# Issue: CLI and MCP Server Cannot Run Simultaneously Due to BadgerDB Limitations

## Summary

The CLI tools (toki, memo, chronicle, health, etc.) and their corresponding MCP servers cannot run at the same time. When the MCP server is running, any CLI command fails with a database lock error. This is a fundamental limitation of BadgerDB, not a bug in our code.

## Error Message

```
Error: failed to initialize charm client: failed to open kv store: database is locked
by another process at "/path/to/charm/kv/toki": Cannot acquire directory lock on
"/path/to/charm/kv/toki". Another process is using this Badger database.
error: resource temporarily unavailable
```

## Root Cause

BadgerDB (the underlying database used by Charm KV) does not support concurrent multi-process access. It uses an exclusive directory lock to ensure only one process can access the database at a time.

### What We Tried

#### Attempt 1: `WithReadOnly(true)`

BadgerDB has a read-only mode option. We tried opening the CLI in read-only mode when a lock was detected.

**Result**: Failed. BadgerDB's read-only mode still requires acquiring the exclusive lock.

#### Attempt 2: `WithReadOnly(true) + WithBypassLockGuard(true)`

BadgerDB has a `BypassLockGuard` option that skips the lock acquisition.

**Result**: Failed. While this bypasses the lock, BadgerDB then detects the database is in an inconsistent state because another process is actively writing:

```
Error: while opening memtables error: while opening fid: 1 error: while updating
skiplist error: end offset: 20 < size: 134217728 error: Log truncate required
to run DB. This might result in data loss
```

This is because BadgerDB's write-ahead log (WAL) is being modified by the MCP server while the CLI tries to read it, resulting in a corrupted view.

### Why This Can't Be Fixed at the Application Level

BadgerDB is fundamentally designed for single-process access. The lock isn't just a convenience—it protects against data corruption. There is no safe way to have two processes access the same BadgerDB database simultaneously.

## Impact

- Users cannot use CLI tools while MCP servers are running
- MCP servers fail to start if multiple are configured (they share Charm client initialization)
- This affects all suite tools: toki, memo, memory, chronicle, health, digest, pagen, position

## Affected Versions

All versions using Charm KV with BadgerDB backend.

## Potential Solutions

### Option A: Accept the Limitation (Low Effort)

Document that CLI and MCP cannot run simultaneously. Users must either:
- Stop the MCP server before using CLI
- Use only MCP tools (via Claude) when MCP is running

**Pros**: No code changes required
**Cons**: Poor user experience

### Option B: CLI Queries MCP Server (Medium Effort)

Instead of the CLI opening the database directly, have CLI commands communicate with the running MCP server via HTTP/socket.

```
Before: CLI → BadgerDB ← MCP Server (conflict!)
After:  CLI → MCP Server → BadgerDB (no conflict)
```

**Implementation**:
1. Add HTTP or Unix socket server to MCP mode
2. CLI detects if MCP is running and routes commands through it
3. Fall back to direct DB access if MCP not running

**Pros**: Solves the problem properly
**Cons**: Requires architectural changes to all tools

### Option C: Switch to SQLite (High Effort)

Replace BadgerDB with SQLite using WAL (Write-Ahead Logging) mode, which supports multiple readers with one writer.

**Pros**: Industry-standard solution for this problem
**Cons**:
- Major refactor of Charm KV
- Would need to migrate existing user data
- May affect Charm Cloud sync behavior

### Option D: Separate Databases (Medium Effort)

MCP server uses its own database copy. Periodically sync between CLI and MCP databases.

**Pros**: Both can run independently
**Cons**:
- Data consistency issues
- Sync complexity
- Storage overhead

## Recommendation

**Short term**: Option A - Document the limitation clearly.

**Long term**: Option B - CLI queries MCP. This is the cleanest architectural solution and aligns with how MCP is meant to work (as a service layer).

## References

- BadgerDB documentation on locking: https://dgraph.io/docs/badger/
- BadgerDB issue on concurrent access: The lock is by design, not a bug
- Charm KV uses BadgerDB: https://github.com/charmbracelet/charm

## Timeline

- 2025-12-19: Issue discovered during testing
- Releases v0.5.3, v0.9.0, v0.4.1, v0.8.1, v1.3.1, v1.4.0 contain attempted fix that does not work
