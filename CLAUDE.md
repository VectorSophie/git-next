# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build -o git-next ./cmd/git-next/             # build binary
go test ./...                                         # run all tests
go test ./internal/engine/...                         # run a single package's tests
go test ./internal/engine/... -run TestEvaluateReturns  # run a single test by name
go vet ./...                                          # static analysis
```

Releases are handled by GoReleaser (`.goreleaser.yaml`), which cross-compiles for Linux/macOS/Windows × AMD64/ARM64/ARM.

## Architecture

`git-next` is a CLI tool that inspects the current git repository state and surfaces prioritized advice — from "dangerous, stop immediately" down to "mild suggestion." It has zero runtime dependencies beyond the Go stdlib and `gopkg.in/yaml.v3`.

**Core data flow:**

```
cmd/git-next/main.go
  → subcommand dispatch (guard/hook/ci/explain/rules/completion)  OR
  → internal/repo    CollectState()   → pkg/model.RepoState
  → internal/rules   AllRules()       → []Rule               (rules R001–R058)
  → internal/engine  Evaluate()       → []Advice             (filtered, sorted)
  → internal/output  Format*()        → stdout
```

### `pkg/model` — shared types

`RepoState` (~50 fields) is the single struct flowing through the whole pipeline. Grouped by concern: dangerous ops, integrity, workflow hygiene, suggestions, informational. `Advice` holds the rule ID, command, description, priority, severity, and a `Destructive` flag. `AgentOutput` and `GuardOutput` are the machine-readable response types for `--agent` mode and the `guard` subcommand respectively.

### `internal/repo` — state collection

Runs `git` subprocesses and populates `RepoState`. Split by concern, not by priority:

| file | what it collects |
|---|---|
| `state.go` | orchestrator; working-tree basics |
| `state_dangerous.go` | R037–R041 signals (rebasing, detached HEAD, stash depth, etc.) |
| `state_integrity.go` | R042–R046 signals (broken refs, corrupted objects, etc.) |
| `state_workflow.go` | R047–R051 signals (unpushed commits, untracked files, etc.) |
| `state_suggestions.go` | R052–R058 signals (branch naming, commit message hygiene, etc.) |

To add a new check: add fields to `RepoState`, populate them in the appropriate `collect*` file, then register a rule in the matching `rules_*.go` file.

### `internal/rules` — rule definitions

Rules split into five files by danger tier (priority value):

| file | tier | priority |
|---|---|---|
| `rules_dangerous.go` | dangerous | 100–90 |
| `rules_integrity.go` | integrity | 89–60 |
| `rules_workflow.go` | workflow | 59–30 |
| `rules_suggestions.go` | suggestions | 29–10 |
| `rules_informational.go` | informational | <10 |

Each rule is a `Rule{ID, Check func(RepoState) bool, Command, Description, Priority}`. `AllRules()` concatenates all tiers in order.

### `internal/engine` — evaluation & suppression

`Evaluate()` runs every rule's `Check` against `RepoState`, sorts survivors by priority, then applies a **suppression map**: if rule A fires, it silences lower-priority contradictory rules (e.g., a `merge` suggestion suppresses any `rebase` suggestion). This prevents mutually exclusive advice from appearing together.

### `internal/policy` — command guard

Defines `Policies`: an ordered list of `CommandPolicy` entries that map dangerous git command patterns (e.g. `push --force`, `reset --hard`) to a `BlockWhen func(RepoState) bool` predicate. `Evaluate(proposed, state)` returns the first matching blocking policy or nil if safe. Used by the `guard` subcommand. First matching policy wins.

### `internal/hook` — git hook manager

`Install`, `Uninstall`, and `List` manage git hook scripts in `.git/hooks/`. Default hook is `pre-push`.

### `internal/explain` — rule explanations

Provides `Lookup(id)` → `Explanation{ID, Title, Why, Alternative}` backed by a static map of all rules. Used by both the `explain <rule>` subcommand and the `--explain` flag on the main command.

### `internal/completion` — shell completion

Generates shell completion scripts for bash, zsh, and fish via the `completion` subcommand.

### `internal/config` — YAML configuration

Config file search order: `.git-next.yaml` (project root) → `~/.config/git-next/config.yaml` → built-in defaults. Supports: protected branch list, disabling specific rule IDs, per-rule parameters (e.g., `R020.max_commits`), and custom suppression pairs.

### `internal/output` — formatters

Output modes selected by CLI flag: human-readable (default), `--json`, `--compact`, `--agent` (machine-readable JSON designed for coding agents and CI bots). `--action` flag invokes `internal/action/executor.go` for interactive menu-driven command execution.

### `internal/testutil` — test helpers

`testutil.NewRepo(t)` creates a temporary git repository with isolated config (no GPG signing). Use `CloneFrom(t, origin)` for push/upstream tests. Tests call `r.ChDir()` so `CollectState()` discovers the temp repo as the working directory.

### `internal/mcp` — MCP server

Exposes git-next as an MCP tool server over stdio (JSON-RPC 2.0). Started via `git-next mcp`. Claude Code discovers it through `.mcp.json` in the project root.

| file | responsibility |
|---|---|
| `classifier.go` | `Classify(command, state) Tier` — SAFE / GUARDED / BLOCKED |
| `executor.go` | `Run()` / `RunResult` — executes or gates commands based on tier |
| `tools.go` | `Registry []Tool` + all 7 `Handle*` handler functions |
| `server.go` | `Serve(repoPath)` — JSON-RPC request/response loop |

**Tools:** `git_status`, `git_guard`, `git_run`, `git_explain`, `git_diff`, `git_log`, `git_remote_status`.

To add a tool: append one `Tool{}` entry to `Registry` in `tools.go`. The server discovers it automatically via `tools/list`.

`git_run` execution model: SAFE commands run immediately; GUARDED commands (push, merge, rebase, commit --amend, stash pop) return `requires_confirmation: true` until called again with `confirmed: true`; BLOCKED commands (matched by `policy.Evaluate`) are never executed regardless of `confirmed`.

### `cmd/git-next` — CLI entrypoint

Dispatches subcommands (`guard`, `hook`, `ci`, `explain`, `rules`, `completion`, `mcp`) before flag parsing so they can manage their own flags. Main flags: `--version`/`-v`, `--all`/`-a`, `--verbose`/`-V` (show info-tier), `--json`, `--compact`, `--agent`, `--debug`, `--explain`, `--action`, `--network` (enables `git ls-remote` for R054), `--config`. Exits with status 1 if any unsuppressed advice is active.
