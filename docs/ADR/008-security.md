# ADR-008: Security Model

## Status

Proposed

## Context

zipfs extracts arbitrary zip files provided by users or AI agents. Zip files can contain malicious content: zip bombs designed to exhaust disk space, path traversal entries (zip slip) designed to write files outside the intended directory, and symlinks pointing to sensitive locations.

The workspace directory is a real filesystem location. Any vulnerability in extraction or path handling is a real filesystem vulnerability with the user's full permissions. MCP servers run with the user's permissions and may be invoked by AI agents that could be influenced by prompt injection attacks.

## Decision

### Zip Slip Prevention (Path Traversal)

Zip slip is a critical vulnerability where a zip entry contains a path like `../../etc/cron.d/malicious` that escapes the extraction directory.

Mitigation:
1. On extraction, **every** entry path is validated **before** any file is written
2. Resolve the target path using `filepath.Join(contentsDir, entryPath)`
3. Compute `filepath.Rel(contentsDir, resolvedPath)` and verify the result does NOT start with `..`
4. Additionally verify `filepath.Clean(resolvedPath)` has `contentsDir` as a prefix
5. Reject entries with:
   - `../` sequences anywhere in the path
   - Absolute paths (starting with `/`)
   - Paths containing null bytes
   - Paths that resolve outside `contents/` after any resolution
6. If **any single entry** fails validation, the **entire extraction aborts** (fail-closed)
7. Partial extractions are cleaned up (workspace directory removed)

Implementation pattern (Go):
```go
func isSafePath(base, entry string) error {
    target := filepath.Join(base, filepath.Clean("/"+entry))
    rel, err := filepath.Rel(base, target)
    if err != nil {
        return fmt.Errorf("path resolution failed: %w", err)
    }
    if strings.HasPrefix(rel, "..") {
        return fmt.Errorf("path traversal detected: %s", entry)
    }
    return nil
}
```

### Zip Bomb Detection

Zip bombs are archives with extreme compression ratios designed to exhaust disk space or memory on extraction.

Pre-extraction scanning (using zip central directory, before extracting any content):

| Check | Default Limit | Configurable |
|-------|--------------|--------------|
| Total uncompressed size | 1 GB | `max_extracted_size_bytes` |
| Per-file compression ratio | 100:1 | `max_compression_ratio` |
| Total entry count | 100,000 | `max_file_count` |
| Total disk across all sessions | 10 GB | `max_total_disk_bytes` |

Extraction aborts immediately if any limit is exceeded. The zip central directory provides uncompressed sizes without requiring decompression, making this check lightweight.

Additionally, during extraction, actual bytes written are tracked against the declared uncompressed size. If actual output exceeds the declared size by more than 10%, extraction aborts (protects against manipulated central directory entries).

### Symlink Handling

Symlinks in zip archives are a path traversal vector.

Default policy (`allow_symlinks: false`):
- Symlink entries in the zip archive are **skipped** during extraction
- A warning is emitted listing skipped symlinks
- This is the safe default for AI agent usage

Optional policy (`allow_symlinks: true`):
- Symlink entries are extracted
- Target paths are validated: must resolve within the workspace `contents/` directory
- Symlinks pointing outside the workspace are rejected
- During sync (repacking), symlinks in `contents/` are stored as symlinks in the zip, NOT followed

### Workspace Directory Permissions

- `workspaces/` root: `0700` (owner only)
- Individual session directories: `0700`
- `metadata.json`: `0600`
- Extracted file permissions: preserved from zip, but never exceed `0755`
- No setuid/setgid bits are ever set, regardless of zip content

### MCP-Specific Security

The MCP server operates in a trusted single-client model:
- **No additional authentication**: Relies on transport security (stdio is inherently single-client)
- **All path parameters validated**: Every tool input is checked for traversal attempts
- **Session identifiers validated**: Only known session IDs/names are accepted
- **No shell expansion**: Tool parameters are never passed through a shell

Input validation on MCP tool parameters:

| Parameter | Validation |
|-----------|-----------|
| `path` (in open) | Must be absolute, must exist, must be a regular file (not symlink/device) |
| `path` (in read/write/ls/etc.) | Must be relative, must not contain `..`, must resolve within workspace |
| `session` | Must match a known session by name, ID, or ID prefix |
| `pattern` (grep) | Compiled with timeout to prevent ReDoS |
| `glob` | Validated, recursive wildcards limited |
| `name` | `[a-zA-Z0-9_-]` only, max 64 chars |

### Regex Denial of Service (ReDoS) Prevention

The `grep` command accepts user-provided regex patterns. Malicious patterns can cause catastrophic backtracking.

Mitigation:
- Regex compilation uses a timeout (default: 5 seconds)
- If pattern compilation or first match exceeds timeout, the operation is cancelled
- Consider using Go's `regexp` package which uses RE2 (guaranteed linear time, no backtracking)
- RE2 doesn't support all PCRE features but provides safety guarantees

### What zipfs Does NOT Protect Against

These are explicitly out of scope:

1. **Malicious file content**: A virus or exploit inside a zip entry is not zipfs's responsibility. zipfs is a filesystem tool, not an antivirus.
2. **Intentional misuse**: A user deliberately placing dangerous files in the workspace.
3. **Time-of-check-time-of-use (TOCTOU)** on the source zip between open and sync: Mitigated by hash verification, but not eliminated.
4. **Disk exhaustion from legitimate large zips**: Mitigated by configurable limits, but the user can override limits.
5. **Encrypted/password-protected zips**: Not supported in v1. Attempting to open one results in a clear error.

### Threat Model Summary

| Threat | Severity | Mitigation | Residual Risk |
|--------|----------|------------|---------------|
| Zip Slip (path traversal in entries) | Critical | Dual path validation, fail-closed extraction | Negligible (well-understood, deterministic check) |
| Zip Bomb (decompression bomb) | High | Pre-scan + runtime size tracking | Low (manipulated central directory, mitigated by runtime check) |
| Symlink escape to sensitive files | High | Default: skip symlinks. Optional: validate targets | Low with default policy |
| Workspace escape via MCP parameters | High | Input validation on all tool parameters | Negligible |
| Source zip TOCTOU | Medium | SHA-256 hash comparison on sync | Low (window between hash check and rename) |
| ReDoS via grep pattern | Medium | RE2 engine (linear time guarantee) | Negligible with RE2 |
| Session name injection | Low | Strict `[a-zA-Z0-9_-]` validation | Negligible |
| Disk exhaustion | Low | Configurable limits, prune command | User can override limits |

### Security Configuration

In `~/.local/share/zipfs/config.json`:

```json
{
  "security": {
    "max_extracted_size_bytes": 1073741824,
    "max_file_count": 100000,
    "max_compression_ratio": 100,
    "max_total_disk_bytes": 10737418240,
    "max_sessions": 32,
    "allow_symlinks": false,
    "regex_timeout_ms": 5000
  }
}
```

Environment variable overrides (prefix: `ZIPFS_`):

| Variable | Purpose |
|----------|---------|
| `ZIPFS_MAX_EXTRACTED_SIZE` | Max extraction size per session (bytes) |
| `ZIPFS_MAX_FILE_COUNT` | Max entries per zip |
| `ZIPFS_MAX_SESSIONS` | Max concurrent sessions |
| `ZIPFS_ALLOW_SYMLINKS` | Enable symlink extraction (`true`/`false`) |

Environment variables override config.json values. CLI flags (e.g., `--max-size` on open) override both.

## Consequences

### Positive

- Fail-closed approach prevents extraction of malicious archives
- Defense in depth: multiple independent checks for path traversal
- Configurable limits allow users to adjust for their specific use case
- Threat model is explicit and documented
- RE2-based regex eliminates ReDoS entirely
- Safe defaults (no symlinks, conservative size limits)

### Negative

- Strict validation may reject some legitimate edge-case zip files (e.g., very high compression ratios on already-compressed data)
- Pre-scanning adds overhead to `zipfs open` (but prevents catastrophic outcomes)
- No malware scanning is provided (documented as out of scope)
- Security configuration complexity may confuse users who just want to open a zip
