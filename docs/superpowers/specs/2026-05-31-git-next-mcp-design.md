# git-next MCP Server — Design Spec

**Date:** 2026-05-31
**Status:** Approved

---

## Goal

Turn `git-next` into an MCP server that AI agents (Claude Code, Codex) can use to inspect repo state, evaluate command safety, and execute git commands with policy-enforced guardrails. The MCP server ships as a new `mcp` subcommand on the existing binary — no new binary, no new release artifact.

---

## Architecture

### New files (nothing existing is restructured)

```
cmd/git-next/cmd_mcp.go          # thin subcommand, calls mcp.Serve()
internal/mcp/server.go           # MCP JSON-RPC 2.0 stdio transport + tool registry loop
internal/mcp/tools.go            # tool definitions (registry) + handlers
internal/mcp/executor.go         # smart git command runner (SAFE/GUARDED/BLOCKED)
internal/mcp/classifier.go       # command classification logic
internal/mcp/classifier_test.go
internal/mcp/tools_test.go
.mcp.json                        # example config for Claude Code
```

### Transport

stdio (newline-delimited JSON-RPC 2.0). Claude Code spawns `git-next mcp` as a child process. No HTTP, no ports, no daemon.

### Tool registry

```go
type Tool struct {
    Name        string
    Description string
    InputSchema json.RawMessage
    Handler     func(ctx context.Context, params map[string]any, repoPath string) (any, error)
}
```

`server.go` iterates the registry for `tools/list` and `tools/call`. It is unaware of what tools exist.

### Repo resolution

Every tool handler calls `resolveRepo(params map[string]any) string`:
- If `repo_path` is present in params, use it.
- Otherwise use cwd captured at server start.
Single implementation, shared by all handlers.

---

## Pre-MCP Project Improvements

These changes touch existing code minimally and make the MCP layer cleaner.

### 1. Severity constants (`pkg/model/types.go`)

Add `type Severity string` with constants `SeverityCritical`, `SeverityHigh`, `SeverityMedium`, `SeverityLow`, `SeverityInfo`. Update `Advice.Severity` field type. Update all existing assignments in `rules_*.go` files.

### 2. RuleByID (`internal/rules/rules.go`)

Add `RuleByID(id string) (Rule, bool)` — builds a map from `AllRules()` on first call. Used by `git_explain` handler.

### 3. Context on CollectState (`internal/repo/state.go`)

Change signature to `CollectState(ctx context.Context, cfg *config.Config) (model.RepoState, error)`. Thread context through to all `exec.Command` calls via `exec.CommandContext`. Update all callers: `main.go`, `cmd_guard.go`, `cmd_ci.go` (pass `context.Background()`).

---

## MCP Tools

### `git_status`
- **Params:** `repo_path` (optional string)
- **Action:** `CollectState` + `engine.Evaluate` → returns full advice list
- **Output:** `{ advice: [{rule_id, severity, description, command, suppressed, destructive}], summary, risk_level }`

### `git_guard`
- **Params:** `command` (string, required), `repo_path` (optional string)
- **Action:** `policy.Evaluate(command, state)` — never executes anything
- **Output:** `{ allowed, risk_level, reason, alternative }`

### `git_run`
- **Params:** `command` (string, required), `repo_path` (optional string), `confirmed` (bool, default false)
- **Action:** Classify command → execute or return guard result
  - SAFE: execute immediately → `{ stdout, stderr, exit_code }`
  - GUARDED + confirmed=false: → `{ requires_confirmation: true, risk_level, reason, command }`
  - GUARDED + confirmed=true: execute → `{ stdout, stderr, exit_code }`
  - BLOCKED: → `{ blocked: true, reason, alternative }` (confirmed=true is ignored)
- **Output:** one of the above shapes

### `git_explain`
- **Params:** `rule_id` (string, required)
- **Action:** `explain.Lookup(rule_id)`
- **Output:** `{ id, title, why, alternative }` or error if not found

### `git_diff`
- **Params:** `repo_path` (optional), `staged` (bool, default false), `file` (optional string)
- **Action:** runs `git diff [--staged] [-- file]`
- **Output:** `{ diff: string, files_changed: int }`

### `git_log`
- **Params:** `repo_path` (optional), `n` (int, default 10), `oneline` (bool, default true)
- **Action:** runs `git log`
- **Output:** `{ commits: [{hash, subject, author, date}] }`

### `git_remote_status`
- **Params:** `repo_path` (optional)
- **Action:** `git fetch --dry-run` + `git rev-list` ahead/behind
- **Output:** `{ ahead, behind, remote_branch, last_fetch }`
- **Note:** network call — slow. Agents should use this explicitly, not on every status check.

---

## Command Classification (`internal/mcp/classifier.go`)

Three tiers, evaluated in order:

**BLOCKED** — policy.Evaluate fires AND RiskLevel == "danger" for the given state. Never execute regardless of `confirmed`.

**GUARDED** — not blocked, but matches a pattern in the GUARDED list:
`push` (non-force), `merge`, `rebase`, `stash pop`, `commit --amend` (always, regardless of push state), `branch -d`, non-force `tag` creation

**SAFE** — everything else that starts with `git`. Whitelist of known-safe prefixes:
`status`, `log`, `diff`, `show`, `add`, `fetch` (no force flags), `commit` (no amend), `branch` (list/create), `stash list`, `stash show`

Commands that don't start with `git` are BLOCKED unconditionally.

---

## MCP Protocol Implementation

Minimal JSON-RPC 2.0 over stdio. Handles:
- `initialize` → respond with server capabilities
- `initialized` (notification, no response)
- `tools/list` → iterate registry
- `tools/call` → dispatch to handler, wrap result in `content: [{type:"text", text: json}]`
- Unknown methods → `-32601` error

No external MCP SDK. The protocol is simple enough to implement in ~150 lines.

---

## Error Handling

- Tool handler returns `error` → MCP response has `isError: true`, content contains the error message
- `CollectState` fails (not a git repo, git not installed) → descriptive error in tool result, not a server crash
- Unknown `repo_path` → error in tool result
- Context cancelled → tool returns immediately with cancellation error

---

## Testing

- `internal/mcp/classifier_test.go` — table tests: command string → expected tier, covering all SAFE/GUARDED/BLOCKED patterns
- `internal/mcp/tools_test.go` — unit tests for each handler using `testutil.NewRepo(t)`, verifying output shape
- Manual integration test: `go build && echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ./git-next mcp`

---

## Release

1. `.mcp.json` in repo root (users copy to project or `~/.claude/`):
```json
{ "mcpServers": { "git-next": { "command": "git-next", "args": ["mcp"] } } }
```
2. `go install github.com/VectorSophie/git-next/cmd/git-next@latest` — no GoReleaser changes needed
3. Validate with Claude Code locally before submitting to registry
4. Submit to Claude Code MCP marketplace after validation

---

## Out of Scope (v1)

- HTTP/SSE transport
- Multi-repo workspace mode
- PR/issue awareness (requires `gh` CLI)
- Codex marketplace submission (after Claude Code validation)
