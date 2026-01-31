# ADR-007: Integration with xlq (excelize-mcp)

## Status

Proposed

## Context

xlq (excelize-mcp) is a CLI/MCP tool for Excel file (.xlsx) manipulation. It provides commands for reading sheets, writing cells, searching, and other spreadsheet operations.

xlq has a deliberate security guardrail: it only accesses files within the current working directory (pwd). This prevents agents from accessing arbitrary files on the filesystem, which is important when running as an MCP server with the user's permissions.

However, this guardrail creates a problem: AI agents frequently create or extract zip archives to `/tmp`, and Excel files within those archives cannot be accessed by xlq because `/tmp` is not the agent's working directory. This is the "second token black hole" that zipfs was created to solve.

zipfs extracts zip contents to `~/.local/share/zipfs/workspaces/<session>/contents/`. xlq needs a way to operate on files in this directory without violating its security model.

## Decision

### Integration via --basepath

The primary integration mechanism is a `--basepath` flag on xlq (a new feature to be added to xlq):

```bash
# Current behavior (fails for files outside pwd):
xlq head --file /tmp/archive/financials.xlsx
# Error: file is outside working directory

# With zipfs + basepath:
zipfs open /tmp/archive.zip --name report
xlq --basepath $(zipfs path report) head --file financials.xlsx
# Success: xlq resolves "financials.xlsx" relative to the zipfs workspace
```

The `--basepath` flag tells xlq to resolve all file paths relative to the given base directory instead of pwd. xlq's security guardrail is preserved -- files must be within the basepath, which is a known, managed directory.

### MCP Integration Pattern

When both zipfs and xlq are configured as MCP servers for an AI agent:

```
Agent workflow:

1. zipfs_open(path="/tmp/reports.zip", name="q4")
   -> { workspace_path: "~/.local/share/zipfs/workspaces/q4/contents" }

2. zipfs_ls(session="q4")
   -> { entries: ["Q4-Report.xlsx", "summary.docx", "data/raw.csv"] }

3. zipfs_path(session="q4")
   -> { path: "/home/user/.local/share/zipfs/workspaces/q4/contents" }

4. xlq_head(basepath="/home/user/.../q4/contents", file="Q4-Report.xlsx", sheet="Revenue")
   -> { rows: [["Month", "Amount"], ["Jan", 50000], ...] }

5. xlq_write_cell(basepath="/home/user/.../q4/contents", file="Q4-Report.xlsx",
                  sheet="Revenue", cell="B13", value="600000")
   -> { written: true }

6. zipfs_sync(session="q4")
   -> { synced: true, backup: "/tmp/reports.bak.zip" }
```

The agent uses `zipfs_path` once to obtain the workspace path, then passes it as `basepath` to all subsequent xlq operations. The modified Excel file is written to the workspace's `contents/` directory, and `zipfs_sync` repacks it into the zip.

### Required Changes to xlq

For this integration to work, xlq needs the following additions:

1. **CLI**: `--basepath <path>` global flag that overrides pwd for all file path resolution
2. **MCP**: `basepath` optional string parameter on all tools that accept a `file` parameter
3. **Validation**:
   - basepath must be an absolute path
   - basepath must exist and be a directory
   - Resolved file paths must still be within the basepath (no `../` escape)
4. **Behavior**: When basepath is set, xlq operates as if its working directory is basepath. All relative file paths in tool parameters are resolved against basepath.

### Alternatives Considered

#### Symlink from pwd to zipfs workspace (Rejected)

Create a symlink in the agent's working directory pointing to the zipfs workspace:
```bash
ln -s $(zipfs path report) ./report-contents
xlq head --file report-contents/financials.xlsx  # within pwd via symlink
```

Rejected because:
- Fragile: symlinks have platform-specific edge cases
- Pollutes the working directory with symlinks
- Confusing for users who see unexpected symlinks
- Security: symlink following opens attack vectors

#### Change xlq's pwd (Rejected)

Have xlq `cd` to the zipfs workspace before operating:

Rejected because:
- Affects the entire process's working directory
- Not composable (can't operate on multiple workspaces in one session)
- Breaks if xlq needs to access its own config files relative to the original pwd

#### zipfs embeds xlq (Rejected)

Have zipfs directly provide Excel operations by embedding xlq's functionality:

Rejected because:
- Tight coupling between two independent tools
- Feature duplication and maintenance burden
- Violates Unix philosophy: do one thing well
- Users would need to update zipfs whenever xlq adds features

#### zipfs MCP provides basepath-aware resource URIs (Deferred)

MCP supports "resources" that expose data via URIs. zipfs could expose workspace files as MCP resources that xlq reads via the MCP resource protocol.

Deferred because:
- MCP resource protocol for cross-server communication is not mature
- The basepath approach is simpler and works today
- Can be revisited when MCP ecosystem matures

### Future Considerations

**Native zip support in xlq**: Long-term, xlq could potentially read xlsx files directly from zip archives without extraction. However:
- This requires xlsx-in-zip awareness in xlq's core
- It doesn't generalize to non-Excel files in zips
- zipfs serves a broader purpose beyond just Excel

The basepath integration is the right level of coupling for the foreseeable future.

**General basepath pattern**: The `--basepath` pattern is not specific to zipfs. It could be useful for any scenario where xlq needs to operate on files outside its pwd. This makes the xlq change independently valuable.

## Consequences

### Positive

- Clean separation of concerns: zipfs manages zip lifecycle, xlq manages Excel operations
- xlq's security model is preserved and extended, not bypassed
- The basepath pattern generalizes to ANY tool that accepts a basepath, not just xlq
- No tight coupling between the two projects
- Each tool can evolve independently
- The integration cost is minimal: one flag on xlq, one tool on zipfs

### Negative

- Requires a small but necessary change to xlq (adding `--basepath` support)
- Agent must coordinate two tools (extra tokens), though the overhead is minimal (~3 extra tool calls)
- The workspace path is long and not human-friendly (mitigated by session names and `zipfs_path`)
- If xlq's `--basepath` change is delayed, the integration doesn't work until it ships
