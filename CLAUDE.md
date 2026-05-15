# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build -o git-next ./cmd/git-next/main.go   # build binary
go test ./...                                  # run all tests
go test ./internal/engine/...                  # run a single package's tests
go vet ./...                                   # static analysis
```

Releases are handled by GoReleaser (`.goreleaser.yaml`), which cross-compiles for Linux/macOS/Windows × AMD64/ARM64/ARM.

## Architecture

`git-next` is a CLI tool that inspects the current git repository state and surfaces prioritized advice — from "dangerous, stop immediately" down to "mild suggestion." It has zero runtime dependencies beyond the Go stdlib and `gopkg.in/yaml.v3`.

**Core data flow:**

```
cmd/git-next/main.go
  → internal/repo    CollectState()   → pkg/model.RepoState  (40+ fields)
  → internal/rules   AllRules()       → []Rule               (58 rules, R001–R058)
  → internal/engine  Evaluate()       → []Advice             (filtered, sorted)
  → internal/output  Format()         → stdout
```

### `pkg/model` — shared types

`RepoState` (~70 fields) is the single struct that flows through the whole pipeline. Grouped by concern: dangerous ops, integrity, workflow hygiene, suggestions, informational. `Advice` holds the rule ID, command, description, and priority for display.

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

58 rules split into five files by danger tier (priority value):

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

### `internal/config` — YAML configuration

Config file search order: `.git-next.yaml` (project root) → `~/.config/git-next/config.yaml` → built-in defaults. Supports: protected branch list, disabling specific rule IDs, per-rule parameters (e.g., `R020.max_commits`), and custom suppression pairs.

### `internal/output` — formatters

Three modes selected by CLI flag: human-readable (default), `--json`, `--compact`. `--action` flag invokes `internal/action/executor.go` for interactive menu-driven command execution.

### `cmd/git-next` — CLI entrypoint

Parses flags (`--version`, `--all`, `--json`, `--compact`, `--debug`, `--action`, `--config`), calls `CollectState → AllRules → Evaluate → Format`, and exits. No persistent state outside config files.
