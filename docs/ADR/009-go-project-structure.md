# ADR-009: Go Project Structure

## Status

Proposed

## Context

zipfs is built in Go for consistency with the xlq (excelize-mcp) ecosystem and the team's primary language. The project needs to support both CLI and MCP server modes from a single binary. It should follow standard Go project layout conventions, enable clean separation between transport layers and business logic, and be installable via `go install`.

## Decision

### Module Path

```
module github.com/Fuabioo/zipfs
```

### Project Layout

```
zipfs/
├── cmd/
│   └── zipfs/
│       └── main.go                 # Entry point
├── internal/
│   ├── cli/                        # CLI command definitions (cobra)
│   │   ├── root.go                 # Root command, global flags (--json, --quiet)
│   │   ├── open.go                 # zipfs open
│   │   ├── close.go                # zipfs close
│   │   ├── ls.go                   # zipfs ls
│   │   ├── tree.go                 # zipfs tree
│   │   ├── read.go                 # zipfs read
│   │   ├── write.go                # zipfs write
│   │   ├── delete.go               # zipfs delete
│   │   ├── grep.go                 # zipfs grep
│   │   ├── sync_cmd.go             # zipfs sync (sync_cmd to avoid stdlib conflict)
│   │   ├── status.go               # zipfs status
│   │   ├── path.go                 # zipfs path
│   │   ├── sessions.go             # zipfs sessions
│   │   ├── prune.go                # zipfs prune
│   │   ├── mcp.go                  # zipfs mcp (starts MCP server)
│   │   ├── version.go              # zipfs version
│   │   └── completion.go           # zipfs completion (bash/zsh/fish)
│   ├── mcp/                        # MCP server implementation
│   │   ├── server.go               # Server setup, tool registration
│   │   ├── tools.go                # Tool handler definitions (maps to core)
│   │   └── transport.go            # stdio transport adapter
│   ├── core/                       # Business logic (transport-agnostic)
│   │   ├── session.go              # Session struct, create/get/list/delete
│   │   ├── workspace.go            # Workspace creation, directory management
│   │   ├── extract.go              # Zip extraction with security validation
│   │   ├── repack.go               # Zip repacking from workspace contents
│   │   ├── sync.go                 # Sync orchestration (conflict check, backup, repack)
│   │   ├── scanner.go              # Filesystem scanning (ls, tree, grep, status)
│   │   ├── config.go               # Configuration loading, defaults, env overrides
│   │   ├── lock.go                 # File-based locking (flock)
│   │   └── paths.go                # XDG path resolution, path construction
│   ├── security/                   # Security validation (isolated for testing)
│   │   ├── zipslip.go              # Path traversal validation
│   │   ├── zipslip_test.go
│   │   ├── zipbomb.go              # Zip bomb detection (pre-scan + runtime)
│   │   ├── zipbomb_test.go
│   │   ├── sanitize.go             # Input sanitization (names, paths, patterns)
│   │   └── sanitize_test.go
│   └── errors/                     # Typed errors with codes
│       └── errors.go               # Error types matching ADR-005 error codes
├── testdata/                       # Test fixtures
│   ├── valid/                      # Known-good zip files for testing
│   │   ├── simple.zip              # Basic zip with a few text files
│   │   ├── nested-dirs.zip         # Deep directory structure
│   │   ├── with-excel.zip          # Contains .xlsx files (xlq integration tests)
│   │   └── large-file-count.zip    # Many small files (limit testing)
│   ├── malicious/                  # Security test fixtures
│   │   ├── zipslip.zip             # Contains ../../../etc/passwd entry
│   │   ├── zipbomb-ratio.zip       # High compression ratio
│   │   ├── zipbomb-size.zip        # Large uncompressed size declared
│   │   ├── symlink-escape.zip      # Symlink pointing to /etc/passwd
│   │   └── null-byte-path.zip      # Path with embedded null bytes
│   └── README.md                   # Documents test fixture purposes
├── docs/
│   └── ADR/                        # Architecture Decision Records
│       ├── 001-project-scope.md
│       ├── 002-workspace-layout.md
│       ├── 003-session-management.md
│       ├── 004-sync-strategy.md
│       ├── 005-mcp-protocol.md
│       ├── 006-cli-interface.md
│       ├── 007-xlq-integration.md
│       ├── 008-security.md
│       └── 009-go-project-structure.md
├── .github/
│   └── workflows/
│       ├── ci.yml                  # Build, test, lint, security scan
│       └── release.yml             # GoReleaser on tagged commits
├── .goreleaser.yml                 # Multi-platform binary release config
├── .golangci.yml                   # Linter configuration
├── go.mod
├── go.sum
├── LICENSE                         # MIT
├── README.md
└── CLAUDE.md                       # Claude Code instructions for this repo
```

### Key Design Principles

#### 1. internal/ Only

All packages are under `internal/` to prevent external Go imports. zipfs is a CLI tool, not a library. The `internal/` boundary is enforced by the Go compiler.

#### 2. Core is Transport-Agnostic

`internal/core/` has no knowledge of CLI flags, cobra commands, MCP JSON schemas, or I/O formatting. Both `internal/cli/` and `internal/mcp/` are thin adapters that:
1. Parse input from their respective formats (CLI args or MCP JSON)
2. Call core functions with typed Go parameters
3. Format core return values into their output format (human text or JSON)

This ensures feature parity and consistent behavior between CLI and MCP.

#### 3. Explicit Error Propagation

Following the project's CLAUDE.md directive (go-like error handling everywhere):
- All functions return `error` as the last return value
- No `panic()` except for truly unrecoverable programmer errors
- Errors are wrapped with context using `fmt.Errorf("operation: %w", err)`
- Typed errors in `internal/errors/` map to ADR-005 error codes

#### 4. Security as a Separate Package

`internal/security/` is isolated from core logic so that:
- Security validation is independently testable with focused test fixtures
- Security functions have no dependencies on session state or configuration
- They accept raw inputs and return errors, nothing more

### Dependencies

Minimal dependency set, standard library preferred:

| Dependency | Purpose | Justification |
|-----------|---------|---------------|
| `github.com/spf13/cobra` | CLI framework | Industry standard for Go CLIs |
| `github.com/mark3labs/mcp-go` | MCP SDK | Go MCP server implementation |
| `github.com/google/uuid` | Session UUIDs | Standard UUID generation |
| `archive/zip` (stdlib) | Zip operations | No external zip library needed |
| `crypto/sha256` (stdlib) | Hash computation | Conflict detection |
| `os`, `path/filepath` (stdlib) | Filesystem | Core operations |
| `regexp` (stdlib) | Grep (RE2) | Linear-time regex |
| `syscall` (stdlib) | File locking | `flock(2)` |

Total external dependencies: 3 (cobra, mcp-go, uuid). This minimizes supply chain risk.

### Build and Release

**CI (GitHub Actions):**
- Go build for linux/amd64 (primary)
- `go test ./...` with race detector
- `golangci-lint run` (static analysis)
- `govulncheck ./...` (vulnerability scanning)
- Test against Go current and previous stable versions

**Release (GoReleaser):**
- Triggered by git tags (`v*`)
- Produces binaries for: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64
- Publishes to GitHub Releases with checksums
- Users install via: `go install github.com/Fuabioo/zipfs@latest` or direct binary download

### Testing Strategy

| Level | Package | What |
|-------|---------|------|
| Unit | `security/` | Path traversal, bomb detection, input sanitization |
| Unit | `core/` | Session CRUD, config loading, path resolution |
| Integration | `core/` | Full workflows: open real zip, modify, sync, verify |
| Integration | `cli/` | CLI command execution, output format verification |
| Fixtures | `testdata/` | Pre-built zips (valid and malicious) |

Security tests are the highest priority. Every entry in the threat model (ADR-008) must have at least one corresponding test.

### CLAUDE.md for This Repo

The repo will include its own `CLAUDE.md` to guide AI agents working on the zipfs codebase:

```markdown
## zipfs

Zip file virtual filesystem CLI + MCP server.

### Build
go build ./cmd/zipfs/

### Test
go test ./...

### Lint
golangci-lint run

### Architecture
See docs/ADR/ for design decisions.
Core business logic is in internal/core/.
CLI and MCP are thin adapters in internal/cli/ and internal/mcp/.
Security validations are in internal/security/.
```

## Consequences

### Positive

- Clean separation between CLI, MCP, and core logic ensures feature parity
- `internal/` prevents accidental API surface exposure
- Standard Go project layout is familiar to any Go developer
- Minimal dependencies reduce supply chain risk and build times
- Isolated security package enables rigorous, focused testing
- GoReleaser provides cross-platform binaries without manual effort

### Negative

- Cobra is the heaviest dependency (but is the universal Go CLI standard)
- MCP SDK choice (`mcp-go`) may need evaluation as the MCP ecosystem is young
- `internal/` means the code cannot be imported as a library (intentional, but limits reuse)
- Test fixtures (malicious zips) must be carefully crafted and maintained
