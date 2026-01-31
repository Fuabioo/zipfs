# zipfs

A Go CLI and MCP server that exposes zip files as virtual filesystems for seamless AI agent integration.

## Problem Statement

When AI coding agents (Claude Code, etc.) need to manipulate files inside zip archives (e.g., Excel files in a zip), they waste enormous amounts of tokens writing ad-hoc extraction/repacking scripts. This creates a "token black hole" where agents repeatedly implement the same zip extraction logic.

zipfs solves this by providing a transparent filesystem layer over zip archives. Agents can work with zip contents as if they were normal directories, while zipfs handles all extraction and repacking automatically.

## Features

- Open zip files as workspace directories
- Session management (multiple zips open simultaneously)
- Tree/ls/grep over zip contents
- Read/write individual files
- Sync changes back to zip with automatic `.bak.zip` backup
- MCP server mode for AI agent integration
- CLI mode for human/script usage
- XDG Base Directory compliant (`~/.local/share/zipfs/`)
- Integrates with xlq (excelize-mcp) via `--basepath`

## Quick Start

### CLI Examples

```bash
# Open a zip file
zipfs open /tmp/report.zip --name report

# Browse contents
zipfs tree report
zipfs ls report/data/

# Get workspace path (for tool integration)
zipfs path report
# Output: ~/.local/share/zipfs/workspaces/report/contents

# Use with xlq for Excel files inside the zip
xlq --basepath $(zipfs path report) head --file financials.xlsx

# Sync changes back to the zip
zipfs sync report

# Clean up
zipfs close report
zipfs prune  # remove all workspaces
```

### Status and Session Management

```bash
# List all open sessions
zipfs sessions

# Check status of a specific session
zipfs status report

# Read a file from the zip
zipfs read report:data/config.json

# Write/update a file in the zip
echo "new content" | zipfs write report:data/notes.txt

# Search within zip contents
zipfs grep "pattern" report
```

## MCP Integration

zipfs can run as an MCP server, exposing zip filesystem operations as tools that AI agents can invoke.

### Configuration

Add to your `claude_desktop_config.json` or `.claude/settings.json`:

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

### Available MCP Tools

- `zipfs_open` - Open a zip file as a workspace session
- `zipfs_close` - Close a workspace session
- `zipfs_ls` - List directory contents within a zip
- `zipfs_tree` - Display tree view of zip contents
- `zipfs_read` - Read file contents from zip
- `zipfs_write` - Write/update file in zip workspace
- `zipfs_delete` - Delete file or directory in workspace
- `zipfs_grep` - Search for patterns in zip contents
- `zipfs_path` - Get workspace path for tool integration
- `zipfs_sync` - Sync workspace changes back to zip
- `zipfs_sessions` - List all open sessions
- `zipfs_prune` - Remove stale or all workspace sessions
- `zipfs_status` - Show modified/added/deleted files since extraction

### Example MCP Workflow

```
1. zipfs_open(path="/data/reports.zip", name="reports")
   -> { session_id, workspace_path, file_count }

2. zipfs_ls(session="reports")
   -> { entries: ["Q4-Report.xlsx", "summary.docx", ...] }

3. zipfs_path(session="reports")
   -> { path: "/home/user/.local/share/zipfs/workspaces/reports/contents" }

4. xlq_head(basepath="...", file="Q4-Report.xlsx", sheet="Revenue")
   -> { rows: [...] }

5. zipfs_sync(session="reports")
   -> { synced: true, backup: "/data/reports.bak.zip" }
```

## Installation

```bash
go install github.com/Fuabioo/zipfs@latest
```

## Workspace Structure

zipfs uses XDG Base Directory specification for workspace management:

```
~/.local/share/zipfs/
├── workspaces/
│   └── <session-name>/
│   │   ├── contents/          # Extracted zip contents (the "mounted" filesystem)
│   │   ├── original.zip       # Copy of the original zip at open time
│   │   └── metadata.json      # Session metadata
│   └── ...
└── config.json                # Global configuration (optional)
```

When syncing, the original zip is backed up as `.bak.zip` before being overwritten.

## Use Cases

### Excel Manipulation in Zips

```bash
zipfs open archive.zip --name data
xlq --basepath $(zipfs path data) head --file report.xlsx --sheet Summary
xlq --basepath $(zipfs path data) write-cell --file report.xlsx --sheet Summary --cell B5 --value 42
zipfs sync data
```

### Pipe-Friendly Workflows

```bash
zipfs open /tmp/archive.zip --name work
cat newfile.txt | zipfs write work:data/newfile.txt
zipfs read work:data/config.json | jq '.version = "2.0"' | zipfs write work:data/config.json
zipfs sync work
zipfs close work --no-sync
```

### Batch File Updates

```bash
for zip in *.zip; do
  name=$(basename "$zip" .zip)
  zipfs open "$zip" --name "$name"
  zipfs write "$name:config.json" --content '{"updated": true}'
  zipfs sync "$name"
  zipfs close "$name" --no-sync
done
```

## Documentation

See `docs/ADR/` for architecture decision records covering:
- Project scope and goals
- Workspace layout and XDG compliance
- Session management lifecycle
- Sync and backup strategy
- MCP protocol design
- CLI interface design
- xlq integration patterns
- Security model
- Go project structure

## License

MIT
