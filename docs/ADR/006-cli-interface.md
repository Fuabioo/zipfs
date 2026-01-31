# ADR-006: CLI Interface Design

## Status

Proposed

## Context

zipfs must be usable as a standalone CLI tool by humans and shell scripts. The CLI should feel natural alongside standard Unix tools (`ls`, `tree`, `grep`) and be scriptable with machine-parsable output. It serves as the "other half" of the MCP interface -- same operations, different I/O format.

The CLI is also used directly by AI agents operating through shell tools (e.g., Claude Code's Bash tool), so commands must be concise and predictable to minimize token waste.

## Decision

### Command Structure

Top-level subcommand pattern (like `git`, `docker`, `kubectl`):

```
zipfs <command> [flags] [arguments]
```

### Commands

#### Session Management

```bash
zipfs open <path.zip> [--name <name>] [--max-size <bytes>]
```
Opens a zip file, extracts to workspace. Outputs session ID, name, workspace path, file count.

```bash
zipfs close [<session>] [--sync | --no-sync]
```
Closes a session and removes workspace. Without flags and with unsaved changes: prompts for confirmation on TTY, errors on non-TTY.

```bash
zipfs sessions [--json]
```
Lists all open sessions. Default: table format. `--json`: JSON array.

```bash
zipfs prune [--all] [--stale <duration>] [--dry-run]
```
Removes stale or all workspaces. Duration format: `1h`, `24h`, `7d`, `30d`.

#### Filesystem Operations

```bash
zipfs ls [<session>] [<path>] [--long] [--recursive] [--json]
```
Lists files in workspace. `--long`: size, permissions, timestamp. `--recursive`: subdirectories.

```bash
zipfs tree [<session>] [<path>] [--max-depth <n>] [--json]
```
Tree view of workspace contents. Output matches standard `tree` command format.

```bash
zipfs read <session>:<path>
zipfs read [<session>] <path>
```
Outputs file contents to stdout. Binary files: base64 encoded with a stderr warning.

```bash
zipfs write <session>:<path> [--stdin | --content <string>]
zipfs write [<session>] <path> [--stdin | --content <string>]
```
Writes to a file in the workspace. `--stdin`: read from stdin (default when piped). `--content`: inline string. Creates parent directories automatically.

```bash
zipfs delete [<session>] <path> [--recursive]
```
Deletes a file or directory from workspace.

```bash
zipfs grep <pattern> [<session>] [<path>] [--glob <pattern>] [-i] [-n] [--max-results <n>] [--json]
```
Searches file contents. Output matches standard `grep` format: `file:line:content`. `-i`: case insensitive. `-n`: line numbers (default on).

#### Sync and Status

```bash
zipfs sync [<session>] [--force] [--dry-run]
```
Repacks workspace into zip at source path. Creates `.bak.zip` backup. `--force`: ignore conflicts. `--dry-run`: preview changes.

```bash
zipfs status [<session>] [--json]
```
Shows modified/added/deleted files since extraction. Output similar to `git status`.

```bash
zipfs path [<session>]
```
Outputs the absolute path to the `contents/` directory. Designed for command substitution: `xlq --basepath $(zipfs path)`. No trailing newline when piped.

#### MCP Server

```bash
zipfs mcp
```
Starts MCP server on stdio. No human-readable output (MCP protocol only).

#### Meta

```bash
zipfs version
zipfs help [<command>]
```

### Colon Syntax

For `read` and `write`, support `session:path` as a convenience:

```bash
zipfs read report:data/file.txt
# Equivalent to:
zipfs read report data/file.txt
```

The colon syntax is parsed by splitting on the first colon. Session names cannot contain colons (enforced by name validation).

### Output Conventions

1. **Human-readable by default**: tables, tree format, grep-like format
2. **`--json` flag** on all list/status commands for machine parsing
3. **`zipfs path`** outputs bare path with no decoration, no trailing newline when stdout is not a TTY (optimized for `$()` substitution)
4. **Errors** go to stderr, data goes to stdout
5. **Progress indicators** (extraction, sync) only when stderr is a TTY

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Usage error (bad arguments, missing required params) |
| 3 | Conflict detected (source zip modified externally) |
| 4 | Session not found / ambiguous session |
| 5 | Zip bomb detected / security violation |

### Pipe-Friendly Design

```bash
# Extract, modify, repack
zipfs open /tmp/archive.zip --name work
cat newfile.txt | zipfs write work:data/newfile.txt
zipfs read work:data/config.json | jq '.version = "2.0"' | zipfs write work:data/config.json
zipfs sync work
zipfs close work --no-sync

# Use with xlq
xlq --basepath $(zipfs path work) head --file financials.xlsx --sheet Revenue

# Batch processing
zipfs sessions --json | jq -r '.[].name' | xargs -I{} zipfs sync {}
```

### Shell Completion

- Support bash, zsh, and fish completion generation
- `zipfs completion bash|zsh|fish`
- Completes: command names, session names (from active sessions), file paths within workspaces

### CLI Framework

- [cobra](https://github.com/spf13/cobra) for command parsing (Go standard)
- Consistent flag naming: `--long-name` with `-x` short aliases for common flags
- All global flags: `--json`, `--quiet` (suppress non-essential output)

## Consequences

### Positive

- Familiar Unix-like interface with low learning curve
- Pipe-friendly design enables shell scripting and composition
- `--json` output enables integration with `jq` and other tools
- Colon syntax reduces typing for the common read/write case
- Shell completion reduces friction for human users
- Clean exit codes enable scripted error handling

### Negative

- Dual syntax (colon vs positional) adds documentation burden and potential confusion
- Session auto-resolution may surprise users who have multiple sessions open
- Cobra dependency is relatively heavy for a CLI tool (but is the Go ecosystem standard)
- `--json` flag on every command increases testing surface
