# ADR-005: MCP Server Protocol Design

## Status

Proposed

## Context

zipfs must be usable by AI agents (Claude Code, Cursor, Windsurf, etc.) via the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/). MCP servers expose "tools" that agents can call with structured JSON parameters and receive structured JSON responses. The MCP interface should mirror the CLI interface but be adapted for programmatic, structured I/O.

The sister project xlq (excelize-mcp) already implements a similar dual CLI/MCP architecture, providing a proven pattern to follow.

## Decision

### Transport

- **Primary**: stdio (standard for local MCP servers, used by Claude Code and most MCP clients)
- **Future (not v1)**: SSE for remote or shared scenarios

### Server Startup

```bash
zipfs mcp
```

Configuration in `claude_desktop_config.json`, `.claude/settings.json`, or equivalent:

```json
{
  "mcpServers": {
    "zipfs": {
      "command": "zipfs",
      "args": ["mcp"]
    }
  }
}
```

### Tool Definitions

All tools follow consistent conventions:
- `session` parameter is optional on every tool that operates on a session. Auto-resolved per ADR-003.
- Errors return structured objects with `code` (matching error codes below) and `message`.
- All string content is UTF-8 unless explicitly marked as base64.

---

#### zipfs_open

Opens a zip file and creates a workspace session.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | yes | Absolute path to the zip file |
| `name` | string | no | Human-readable session name |

**Returns:**
```json
{
  "session_id": "a1b2c3d4-...",
  "name": "q4-report",
  "workspace_path": "/home/user/.local/share/zipfs/workspaces/q4-report/contents",
  "file_count": 42,
  "extracted_size_bytes": 1048576
}
```

---

#### zipfs_close

Closes a session and removes its workspace.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `session` | string | no | Session name or ID |
| `sync` | boolean | no | Sync before closing (default: false) |

**Returns:**
```json
{ "closed": true, "synced": false }
```

---

#### zipfs_ls

Lists files and directories in the workspace.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `session` | string | no | Session name or ID |
| `path` | string | no | Relative path within workspace (default: "/") |
| `recursive` | boolean | no | Include subdirectories (default: false) |

**Returns:**
```json
{
  "entries": [
    { "name": "data/", "type": "dir", "size_bytes": 0, "modified_at": "2025-01-30T12:00:00Z" },
    { "name": "report.xlsx", "type": "file", "size_bytes": 524288, "modified_at": "2025-01-30T11:00:00Z" }
  ]
}
```

---

#### zipfs_tree

Returns a tree representation of the workspace contents.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `session` | string | no | Session name or ID |
| `path` | string | no | Root path for the tree (default: "/") |
| `max_depth` | integer | no | Maximum depth to traverse |

**Returns:**
```json
{
  "tree": ".\n├── data/\n│   ├── input.csv\n│   └── output.csv\n├── report.xlsx\n└── README.txt",
  "file_count": 4,
  "dir_count": 1
}
```

---

#### zipfs_read

Reads a file from the workspace.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `session` | string | no | Session name or ID |
| `path` | string | yes | Relative path to file |
| `encoding` | string | no | "utf-8" (default) or "base64" for binary |
| `offset` | integer | no | Byte offset to start reading |
| `limit` | integer | no | Maximum bytes to read |

**Returns:**
```json
{
  "content": "file contents here...",
  "size_bytes": 1234,
  "encoding": "utf-8"
}
```

---

#### zipfs_write

Writes or updates a file in the workspace.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `session` | string | no | Session name or ID |
| `path` | string | yes | Relative path to file |
| `content` | string | yes | File content |
| `encoding` | string | no | "utf-8" (default) or "base64" |
| `create_dirs` | boolean | no | Create parent directories (default: true) |

**Returns:**
```json
{ "written": true, "size_bytes": 1234 }
```

---

#### zipfs_delete

Deletes a file or directory from the workspace.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `session` | string | no | Session name or ID |
| `path` | string | yes | Relative path within workspace |
| `recursive` | boolean | no | For directories (default: false) |

**Returns:**
```json
{ "deleted": true, "path": "data/old-report.xlsx" }
```

---

#### zipfs_grep

Searches file contents in the workspace.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `session` | string | no | Session name or ID |
| `pattern` | string | yes | Search pattern (regex) |
| `path` | string | no | Root path to search from (default: "/") |
| `glob` | string | no | File glob filter (e.g., "*.txt") |
| `ignore_case` | boolean | no | Case-insensitive search (default: false) |
| `max_results` | integer | no | Maximum matches to return (default: 100) |

**Returns:**
```json
{
  "matches": [
    { "file": "data/config.json", "line_number": 5, "line_content": "  \"debug\": true," }
  ],
  "total_matches": 1,
  "truncated": false
}
```

---

#### zipfs_path

Returns the filesystem path to the workspace contents directory. This is the key integration point with xlq `--basepath`.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `session` | string | no | Session name or ID |

**Returns:**
```json
{ "path": "/home/user/.local/share/zipfs/workspaces/q4-report/contents" }
```

---

#### zipfs_sync

Syncs workspace changes back to the original zip file.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `session` | string | no | Session name or ID |
| `force` | boolean | no | Ignore external modification conflict (default: false) |
| `dry_run` | boolean | no | Preview changes without syncing (default: false) |

**Returns:**
```json
{
  "synced": true,
  "backup_path": "/tmp/reports.bak.zip",
  "files_modified": 2,
  "files_added": 1,
  "files_deleted": 0
}
```

---

#### zipfs_status

Shows what changed in the workspace since extraction.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `session` | string | no | Session name or ID |

**Returns:**
```json
{
  "modified": ["data/config.json", "report.xlsx"],
  "added": ["data/new-file.txt"],
  "deleted": ["old-readme.txt"],
  "unchanged_count": 38
}
```

---

#### zipfs_sessions

Lists all open sessions.

**Parameters:** none

**Returns:**
```json
{
  "sessions": [
    {
      "id": "a1b2c3d4-...",
      "name": "q4-report",
      "source_path": "/tmp/reports.zip",
      "state": "open",
      "created_at": "2025-01-30T12:00:00Z",
      "last_accessed_at": "2025-01-30T12:30:00Z",
      "file_count": 42,
      "extracted_size_bytes": 1048576
    }
  ]
}
```

---

#### zipfs_prune

Removes stale or all workspaces.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `all` | boolean | no | Remove all sessions (default: false) |
| `stale` | string | no | Duration like "24h", "7d" |
| `dry_run` | boolean | no | Preview without removing (default: false) |

**Returns:**
```json
{
  "pruned": [
    { "id": "x1y2z3...", "name": "old-session", "reason": "stale (3d)" }
  ],
  "freed_bytes": 5242880
}
```

---

### Error Codes

| Code | Description |
|------|-------------|
| `SESSION_NOT_FOUND` | Session identifier doesn't match any session |
| `AMBIGUOUS_SESSION` | Multiple sessions open and none specified |
| `NO_SESSIONS` | No sessions open |
| `ZIP_NOT_FOUND` | Source zip file doesn't exist or isn't readable |
| `ZIP_INVALID` | File is not a valid zip archive |
| `ZIP_BOMB_DETECTED` | Extracted size, ratio, or count exceeds limits |
| `CONFLICT_DETECTED` | Source zip modified externally since open |
| `SYNC_FAILED` | Error during sync operation |
| `PATH_TRAVERSAL` | Attempted path escape from workspace |
| `PATH_NOT_FOUND` | Requested path doesn't exist in workspace |
| `LOCKED` | Another operation has the session locked |
| `LIMIT_EXCEEDED` | Max sessions, max disk usage, etc. |
| `NAME_COLLISION` | Session name already in use |

Error response format:
```json
{
  "error": {
    "code": "CONFLICT_DETECTED",
    "message": "Source zip /tmp/reports.zip has been modified since it was opened. Use force=true to overwrite."
  }
}
```

## Consequences

### Positive

- Complete feature parity between CLI and MCP
- Session auto-resolution reduces token usage for agents (no need to specify session in single-zip workflows)
- Structured error codes enable programmatic error handling by agents
- `zipfs_path` enables zero-friction integration with xlq and any other basepath-aware tool
- Consistent parameter naming and return formats reduce agent confusion

### Negative

- Large tool surface area (13 tools) -- but each is simple and single-purpose
- MCP stdio server is single-tenant (one agent per server instance)
- No streaming for large file reads (content returned as single string) -- mitigated by offset/limit parameters
