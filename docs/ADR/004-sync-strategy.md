# ADR-004: Sync and Backup Strategy

## Status

Proposed

## Context

zipfs modifies files in an extracted workspace directory and must write changes back to the original zip file. Data loss is unacceptable -- the original zip must be recoverable. The source zip might be modified externally between `zipfs open` and `zipfs sync`. The sync operation must handle added, modified, and deleted files, and be as atomic as the filesystem allows.

## Decision

### Backup Strategy

Before overwriting the source zip, a backup is always created:

1. Rename `source.zip` to `source.bak.zip`
2. If `source.bak.zip` already exists, rotate:
   - `source.bak.zip` -> `source.bak.2.zip`
   - `source.bak.2.zip` -> `source.bak.3.zip`
   - (and so on up to the rotation depth)
3. Maximum backup rotation depth: **3** (configurable in config.json)
4. The rename operation is atomic on the same filesystem (single `rename(2)` syscall)

Additionally, the workspace retains `original.zip` as a second, independent recovery point that is never modified.

### Conflict Detection

1. On `zipfs open`: compute and store SHA-256 hash of the source zip file
2. On `zipfs sync`: recompute the SHA-256 hash of the current source zip file
3. Compare hashes:
   - **Match**: Source is unchanged. Proceed with sync.
   - **Mismatch**: Source was modified externally since open.
     - Default behavior: error with message explaining the conflict and suggesting `--force`
     - `--force` flag: proceed anyway. Backup is still created, so the externally-modified version is preserved as `.bak.zip`.
     - `--merge`: NOT supported in v1 (three-way merge on zip files is out of scope)

### Sync Process

Ordered, sequential steps:

```
 1. Acquire exclusive lock on session
 2. Verify session state is "open"
 3. Set state to "syncing" in metadata
 4. Verify source path exists and parent directory is writable
 5. Compute SHA-256 of current source zip
 6. Compare with stored hash
 7. If conflict and no --force: abort, restore state to "open"
 8. Build new zip from contents/ into temp file
 9. Rotate existing backups (bak.2 -> bak.3, bak -> bak.2)
10. Rename source.zip -> source.bak.zip
11. Rename temp file -> source.zip
12. Update metadata: last_synced_at, zip_hash_sha256 (of new zip)
13. Set state to "open"
14. Release lock
```

### Temp File Strategy

The new zip is built into a temporary file in the **same directory** as the source zip:
- This guarantees the final rename is atomic (same filesystem, single `rename(2)`)
- Temp file naming: `.source.zip.zipfs-tmp-<random-suffix>`
- The dot prefix hides it from casual `ls` output
- If sync fails at any point during zip building (step 8), the temp file is cleaned up and the original source is untouched

### Dry Run

`zipfs sync --dry-run` performs steps 1-7, then instead of building the zip, computes a diff:
- Lists modified files (content or metadata changed)
- Lists added files (new files in contents/ not in original)
- Lists deleted files (files from original not in contents/)
- Reports estimated new zip size

### Compression

- Default: `deflate` (standard zip compression method)
- When possible, preserve the original compression method per file entry
- Store the original compression method in an internal index during extraction
- Files not in the original (newly added): use deflate

### File Permissions and Metadata Preservation

- Preserve Unix file permissions from zip entries during extraction
- Restore permissions to zip entries during repacking
- Preserve modification timestamps where the zip format supports them
- Do NOT follow symlinks during repacking (security -- see ADR-008)

### Recovery Scenarios

| Scenario | State | Recovery |
|----------|-------|----------|
| Crash during extraction (open) | Workspace may be incomplete | Prune the session, re-open |
| Crash during zip building (step 8) | Temp file left, original untouched | Temp file cleaned on next sync or prune |
| Crash after backup rename (step 10) | Source is now `.bak.zip`, no source.zip | Manual rename `.bak.zip` back. Metadata shows state=`syncing` -- detected on next operation. |
| Crash after final rename (step 11) | New zip in place, metadata not yet updated | Zip is valid. Metadata state=`syncing` detected and auto-recovered on next access. |
| Disk full during zip building | Temp file write fails, original untouched | Error reported. User frees disk, retries. |
| Source zip deleted externally | Step 4 fails | Error: source no longer exists. User can extract from workspace `original.zip` manually. |

### Auto-Recovery from Stale `syncing` State

If metadata shows state=`syncing` when a new operation starts:
1. Check if temp file exists -> clean it up
2. Check if source.zip exists -> state was likely recovered
3. Set state back to `open`
4. Log a warning about the unclean previous sync

## Consequences

### Positive

- Two-phase rename provides near-atomic sync on same filesystem
- Multiple recovery points: workspace `original.zip` + source `.bak.zip`
- Conflict detection prevents silent overwrites of external changes
- Backup rotation prevents unbounded `.bak` file growth
- Dry run enables preview before committing changes
- Auto-recovery handles most crash scenarios without user intervention

### Negative

- Requires same filesystem for atomicity (temp file must be in source directory)
- Brief window between the two renames (steps 10-11) where only `.bak.zip` exists
- Large zips take time to repack (no incremental update possible with standard zip format)
- Disk usage spikes during sync (source + bak + temp all exist briefly)
- Backup rotation on each sync means frequent syncs accumulate backups (mitigated by rotation depth limit)
