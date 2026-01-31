# ADR-001: Project Scope and Goals

## Status

Proposed

## Context

AI coding agents (Claude Code, Cursor, etc.) frequently need to manipulate files inside zip archives. The most common case is Excel files (.xlsx, which are themselves zip archives) nested inside other zip archives sent as email attachments, exported reports, or data packages.

The current workaround is that agents write ad-hoc Python or Go scripts to extract, modify, and repack zip files. This wastes enormous token budgets on boilerplate code that is re-invented every session. This is a "token black hole."

The sister project excelize-mcp (xlq) solved the Excel manipulation problem but has a security guardrail restricting file access to the current working directory (pwd). Files at `/tmp` -- which account for approximately 99% of agent-created temporary archives -- cannot be accessed. This creates a second token black hole where agents must work around the path restriction.

The fundamental need is a transparent layer that makes zip file contents accessible as regular filesystem paths, enabling any tool to operate on them without bespoke extraction logic.

## Decision

### Technology

- Build zipfs as a **Go CLI + MCP server**
- Go is chosen for consistency with the xlq ecosystem and team expertise
- Dual interface: CLI for humans/scripts, MCP for AI agents

### Architecture

- zipfs extracts zip contents to managed workspace directories on real disk
- Any tool (grep, sed, xlq, cat, etc.) can operate on the extracted files using normal filesystem paths
- Changes are synced back to the original zip file on demand
- Session management allows multiple zip files open simultaneously

### What zipfs is NOT

- **NOT a FUSE filesystem**: FUSE is too complex, platform-dependent, and requires elevated privileges. Real extraction to real directories is simpler and more portable.
- **NOT an in-memory virtual FS**: Other tools need real filesystem paths they can access independently. An in-memory abstraction would require every tool to integrate with it.
- **NOT a zip library**: zipfs is an operational tool, not a programmatic API. It wraps standard library zip operations with lifecycle management.

## Goals

1. Zero-friction zip file manipulation for AI agents
2. Session-based workspace management with concurrent multi-zip support
3. Safe sync with backup (`.bak.zip`) before overwriting original archives
4. XDG Base Directory specification compliance
5. Integration with xlq/excelize-mcp via `--basepath`
6. Minimal token cost for the agent (simple, predictable commands)
7. Robust security model (zip slip prevention, zip bomb detection)

## Non-Goals

1. Replacing system-level zip utilities (`zip`, `unzip`, `7z`)
2. Streaming or partial extraction of massive archives (multi-GB)
3. FUSE or kernel-level filesystem mounting
4. Encryption or password-protected zip support (v1)
5. Nested zip support (zip within zip) -- the agent can open them as separate sessions
6. Compression algorithm selection beyond deflate default
7. Remote/network zip file access

## Consequences

### Positive

- Agents can manipulate zip contents with minimal token expenditure (open, path, sync -- three commands)
- Real filesystem paths mean ANY tool works without modification (grep, sed, xlq, jq, etc.)
- Backup strategy prevents data loss on sync
- XDG compliance makes workspaces discoverable and manageable

### Negative

- Disk space usage for extracted workspaces (mitigated by prune and configurable limits)
- Not atomic -- if the process crashes mid-sync, the `.bak.zip` provides recovery but manual intervention may be needed
- Extraction time scales with zip size (no partial extraction in v1)
