# ADR-003: Session Management

## Status

Proposed

## Context

AI agents and human users may need multiple zip files open simultaneously. Sessions must be identifiable, listable, and cleanable. The system needs lifecycle management covering the full workflow from opening a zip to closing and cleaning up the workspace.

Edge cases include: stale sessions from crashed processes, disk pressure from accumulated workspaces, and concurrent access from multiple processes.

## Decision

### Session Lifecycle

```
  open ──> [use] ──> sync ──> [use] ──> close
   │                  │                    │
   │                  └── (repeatable) ────┘
   │                                       │
   └── close (discard) ───────────────────>│
                                           v
                                        removed
```

States: `open`, `syncing`, `closed` (terminal -- workspace removed)

### Operations

#### 1. Open

```
zipfs open <path.zip> [--name <name>] [--max-size <bytes>]
```

Steps:
1. Validate source zip file exists, is readable, and is a valid zip archive
2. Check session name uniqueness (if --name provided)
3. Check global limits (max sessions, max total disk)
4. Pre-scan zip central directory for security checks (see ADR-008):
   - Total uncompressed size vs max_extracted_size
   - Compression ratios vs max_compression_ratio
   - Entry count vs max_file_count
5. Create workspace directory structure
6. Copy source zip to `workspace/original.zip`
7. Compute SHA-256 hash of source zip
8. Extract contents to `workspace/contents/`
9. Write `metadata.json` with state=`open`
10. Output: session ID, name, workspace path, file count, extracted size

#### 2. Use (read/write operations)

No state transition. Operations update `last_accessed_at` in metadata:
- `ls`, `tree`, `grep`, `read` -- read-only operations on contents/
- `write`, `delete` -- mutating operations on contents/
- `path` -- returns absolute path to contents/ directory
- `status` -- compares current contents/ against original extraction

#### 3. Sync

```
zipfs sync [<session>] [--force] [--dry-run]
```

Steps:
1. Acquire exclusive lock on session (see Concurrency section)
2. Verify session state is `open`
3. Set state to `syncing` in metadata
4. Verify source path still exists and is writable
5. Compute SHA-256 hash of current source zip
6. Compare hash with stored hash from metadata
7. If hashes differ and no `--force`: abort, restore state to `open`, report conflict
8. Build new zip from contents/ into a temp file in the source directory
9. Rename source.zip to source.bak.zip (see ADR-004)
10. Rename temp file to source.zip
11. Update metadata: `last_synced_at`, `zip_hash_sha256`
12. Set state to `open`
13. Release lock

#### 4. Close

```
zipfs close [<session>] [--sync | --no-sync]
```

Behavior:
- `--sync`: Sync first (step 3 above), then remove workspace directory
- `--no-sync`: Remove workspace directory immediately, discard all changes
- Neither flag, unsaved changes exist:
  - Interactive TTY: prompt for confirmation
  - Non-interactive (piped, MCP): return error listing unsaved changes

Steps:
1. If syncing: perform full sync operation
2. Remove entire workspace directory
3. Session ceases to exist (no metadata retained)

#### 5. Prune

```
zipfs prune [--all] [--stale <duration>] [--dry-run]
```

Options:
- `--all`: Remove ALL workspaces regardless of state
- `--stale <duration>`: Remove sessions whose `last_accessed_at` exceeds the duration (e.g., `24h`, `7d`, `30d`)
- `--dry-run`: List what would be removed without removing anything

Duration format: Go `time.ParseDuration` compatible plus day shorthand (`d` = 24h).

### Default Session Resolution

When a session identifier is omitted from any command:

| Open Sessions | Behavior |
|---------------|----------|
| Exactly 1 | Use it automatically (convenience for single-zip workflows) |
| 0 | Error: "no sessions open. Use `zipfs open <file.zip>` to start." |
| 2+ | Error: "multiple sessions open. Specify one of: [list names/IDs]" |

This auto-resolution dramatically reduces token usage for AI agents in the common single-zip scenario.

### Concurrency

File-based locking using `metadata.json.lock`:

- Read operations (`ls`, `tree`, `grep`, `read`, `path`, `status`): shared lock (multiple readers)
- Write operations (`write`, `delete`): shared lock (writes to contents/ are filesystem-level atomic for individual files)
- Sync and close: exclusive lock (no concurrent readers or writers)
- Lock acquisition timeout: 10 seconds, then error with `LOCKED` code
- Implementation: `flock(2)` syscall via Go's `syscall.Flock`
- Lock file is created alongside `metadata.json`, cleaned up on close

### Limits

| Limit | Default | Configurable Via |
|-------|---------|------------------|
| Max concurrent sessions | 32 | config.json, `ZIPFS_MAX_SESSIONS` |
| Max extracted size per session | 1GB | config.json, `ZIPFS_MAX_EXTRACTED_SIZE`, `--max-size` flag |
| Max total workspace disk usage | 10GB | config.json |
| Max files per zip | 100,000 | config.json, `ZIPFS_MAX_FILE_COUNT` |

## Consequences

### Positive

- Clear lifecycle with recovery at each stage
- Default session resolution minimizes agent token usage in the common case
- Prune with duration enables automated cleanup (e.g., via cron)
- Hash-based conflict detection prevents silent data loss on sync
- File-based locking prevents corruption from concurrent access

### Negative

- File locking adds implementation complexity
- State tracking in `metadata.json` could become stale on unclean process exit (mitigated by prune --stale)
- Exclusive lock during sync blocks all other operations on the session
- Default session resolution may surprise users who forget they have a session open
