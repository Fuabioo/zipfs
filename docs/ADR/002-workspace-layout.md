# ADR-002: Workspace Layout and XDG Compliance

## Status

Proposed

## Context

zipfs extracts zip files to managed workspace directories on disk. Multiple sessions can be open simultaneously, each requiring storage for extracted contents, a backup of the original zip, and session metadata.

The storage location must be predictable, standard, and discoverable. It should follow Linux desktop conventions and be overridable for CI/CD or testing scenarios.

## Decision

### XDG Base Directory Specification

zipfs follows the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html):

- Base data directory: `$XDG_DATA_HOME/zipfs` (defaults to `~/.local/share/zipfs`)
- Full override: `$ZIPFS_DATA_DIR` overrides the entire base path

### Directory Structure

```
~/.local/share/zipfs/
├── workspaces/
│   ├── <session-id-or-name>/
│   │   ├── contents/          # Extracted zip contents (the "mounted" filesystem)
│   │   ├── original.zip       # Copy of the original zip file at open time
│   │   └── metadata.json      # Session metadata
│   ├── <another-session>/
│   │   ├── contents/
│   │   ├── original.zip
│   │   └── metadata.json
│   └── ...
└── config.json                # Global configuration (optional)
```

### Workspace Components

**`contents/`** -- The extracted zip file contents. This is the directory returned by `zipfs path` and the path that other tools (xlq, grep, etc.) operate on. It mirrors the internal structure of the zip archive exactly.

**`original.zip`** -- A byte-for-byte copy of the source zip file at the time of `zipfs open`. Serves as a recovery point independent of the source file. The source file may be moved, deleted, or modified externally after open.

**`metadata.json`** -- Session state and tracking information:

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "name": "q4-report",
  "source_path": "/tmp/downloads/q4-financials.zip",
  "created_at": "2025-01-30T12:00:00Z",
  "last_synced_at": null,
  "last_accessed_at": "2025-01-30T12:00:00Z",
  "state": "open",
  "zip_hash_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "extracted_size_bytes": 1048576,
  "file_count": 42
}
```

### Session Identification

Each session has two identifiers:

1. **UUID** (`id`): Always generated, globally unique, used internally
2. **Name** (`name`): Optional human-readable string, provided via `--name` at open time

Directory naming rules:
- If `--name` is provided: the workspace directory uses the name (e.g., `workspaces/q4-report/`)
- If no name: the workspace directory uses the UUID (e.g., `workspaces/a1b2c3d4-e5f6-7890-abcd-ef1234567890/`)
- Names must be unique across all sessions. On collision: error with suggestion to use a different name.

Session resolution (how commands find a session):
- By exact name match
- By exact UUID match
- By UUID prefix (minimum 4 characters)
- If no session identifier provided: auto-resolve (see ADR-003)

Name constraints:
- Alphanumeric characters, hyphens, and underscores only: `[a-zA-Z0-9_-]`
- Maximum 64 characters
- Cannot be a valid UUID prefix (to avoid ambiguity)

### Global Configuration

`config.json` at the data root stores user preferences and security limits:

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
  },
  "defaults": {
    "backup_rotation_depth": 3
  }
}
```

### Environment Variable Overrides

| Variable | Purpose | Default |
|----------|---------|---------|
| `ZIPFS_DATA_DIR` | Override entire base data directory | `$XDG_DATA_HOME/zipfs` |
| `ZIPFS_MAX_EXTRACTED_SIZE` | Max extraction size per session | `1073741824` (1GB) |
| `ZIPFS_MAX_SESSIONS` | Max concurrent sessions | `32` |
| `ZIPFS_MAX_FILE_COUNT` | Max files per zip | `100000` |

### Permissions

- Workspace root directory (`workspaces/`): `0700` (user-only)
- Individual workspace directories: `0700`
- Extracted files: preserve permissions from zip archive
- `metadata.json`: `0600`

## Consequences

### Positive

- Standard location, discoverable by users and other tools
- XDG compliance avoids dotfile pollution in `$HOME`
- Metadata enables session lifecycle management (TTL-based prune, status checks)
- `original.zip` copy enables safe recovery independent of source file state
- Environment variable override supports CI/CD and testing

### Negative

- Disk space approximately doubles per session (original.zip + extracted contents)
- Large zip files mean significant per-session disk usage
- XDG_DATA_HOME may not be set on all systems (mitigated by sensible default)
