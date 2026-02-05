## zipfs

Zip file virtual filesystem CLI + MCP server.

### Build

```bash
go build ./cmd/zipfs/
```

### Test

```bash
go test ./... -race -count=1
```

### Lint

```bash
golangci-lint run
```

### Architecture

See `docs/ADR/` for all design decisions.

- `internal/errors/` - Typed errors with codes (ADR-005)
- `internal/security/` - Zip slip, zip bomb, input sanitization (ADR-008)
- `internal/core/` - Business logic: sessions, workspaces, extraction, sync (ADR-002, 003, 004)
- `internal/cli/` - Cobra CLI commands (ADR-006)
- `internal/mcp/` - MCP server and tool handlers (ADR-005)
- `cmd/zipfs/` - Entry point

### Conventions

- Go-like error handling everywhere. Never ignore errors.
- All packages under `internal/` -- this is a CLI tool, not a library.
- Core is transport-agnostic. CLI and MCP are thin adapters.
- Security validations are in a separate package for isolated testing.
- Use `uint64` for any ID fields.
- Prefer standard library. External deps: cobra, mcp-go, uuid only.
