# git-next

[![CI](https://github.com/VectorSophie/git-next/actions/workflows/ci.yml/badge.svg)](https://github.com/VectorSophie/git-next/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/VectorSophie/git-next)](https://github.com/VectorSophie/git-next/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/VectorSophie/git-next)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**A deterministic Git safety oracle for humans, CI, and coding agents.**

`git-next` analyzes your repository state and tells you exactly what to do — and stops you before you do the wrong thing. No philosophy. No guessing. Same state, same advice, every time.

```
$ git-next
Git Next - Suggested Actions
═══════════════════════════════

→ [R006] Branch has diverged - need to sync
  Command: git rebase origin/main OR git merge origin/main

→ [R002] Modified files not staged
  Command: git add <files> && git commit

───────────────────────────────
Active: 2  Suppressed: 0
```

---

## Why not just `git status`?

`git status` tells you what is. `git-next` tells you what to do — and in what order.

| | `git status` | pre-commit | git-next |
|---|:---:|:---:|:---:|
| Shows current state | ✓ | ✗ | ✓ |
| Tells you what to do next | ✗ | ✗ | ✓ |
| Blocks dangerous operations | ✗ | partial | ✓ |
| Suppresses conflicting advice | ✗ | ✗ | ✓ |
| CI/CD gate with severity tiers | ✗ | ✓ | ✓ |
| Machine-readable output for AI agents | ✗ | ✗ | ✓ |
| Zero config, works in any repo | ✓ | ✗ | ✓ |

---

## Installation

```bash
# Using the install script
curl -sfL https://raw.githubusercontent.com/VectorSophie/git-next/main/scripts/install.sh | sh

# Or with Go
go install github.com/VectorSophie/git-next/cmd/git-next@latest

# Or clone and build
git clone https://github.com/VectorSophie/git-next
cd git-next
go build -o git-next cmd/git-next/main.go
```

---

## Usage

```bash
git-next                          # Show what to do next (medium+ severity)
git-next --verbose                # Show all severities including low/info hints
git-next --all                    # Include suppressed advice
git-next --explain                # Add a "why" line under each rule
git-next --json                   # Structured JSON output
git-next --compact                # One-line summary (good for PS1)
git-next --agent                  # Machine-readable JSON for AI agents
git-next --action                 # Interactive: select and run an action
git-next --network                # Include unpushed-tags check (network call)
```

By default `git-next` shows **medium and above** (skipping low-severity and informational hints). Use `--verbose` or `--all` to see everything.

### For coding agents

```bash
git-next --agent
```

```json
{
  "schema_version": "1",
  "status": "unsafe",
  "risk_level": "high",
  "summary": "Branch has diverged - need to sync; Modified files not staged",
  "recommended_next_command": "git rebase origin/main OR git merge origin/main",
  "allowed_commands": ["git add", "git commit", "git diff", "git fetch", "git log", "git status"],
  "blocked_commands": [],
  "rules_triggered": [
    { "id": "R006", "severity": "high", "description": "Branch has diverged - need to sync", "destructive": false },
    { "id": "R002", "severity": "medium", "description": "Modified files not staged", "destructive": false }
  ]
}
```

See [`docs/AGENT_USAGE.md`](docs/AGENT_USAGE.md) for integration examples with Claude Code, GitHub Actions, and shell aliases.

### MCP server (Claude Code / AI agents)

`git-next` ships as an MCP server, letting Claude Code and other agents inspect your repo and run git commands safely without leaving the AI session.

**Install for a single project** — add `.mcp.json` to your repo root (already included in this repo):

```json
{
  "mcpServers": {
    "git-next": { "command": "git-next", "args": ["mcp"] }
  }
}
```

**Install globally** (available in all your projects):

```bash
claude mcp add --transport stdio --scope user git-next -- git-next mcp
```

**Tools exposed to the agent:**

| Tool | What it does |
|---|---|
| `git_status` | Full repo state + prioritized advice. Call this first. |
| `git_guard` | Check if a command is safe. Never executes. |
| `git_run` | Run a git command with SAFE / GUARDED / BLOCKED enforcement. |
| `git_explain` | Full explanation of any rule ID (e.g. `R037`). |
| `git_diff` | Staged or unstaged diff (optionally filtered to one file). |
| `git_log` | Recent commit history as structured JSON. |
| `git_remote_status` | Ahead/behind vs remote (explicit network call). |

**`git_run` safety model:**

- **SAFE** (status, log, diff, add, fetch, commit) → executes immediately
- **GUARDED** (push, merge, rebase, commit --amend, stash pop) → returns `requires_confirmation: true`; agent calls again with `confirmed: true` to proceed
- **BLOCKED** (force-push, filter-branch, reset --hard on protected branch, etc.) → never executes, returns reason + safer alternative

```bash
git-next mcp    # start the MCP server (Claude Code manages the process)
```

### Guard mode

Check whether a command is safe before running it:

```bash
$ git-next guard -- git reset --hard HEAD~3
BLOCKED: git reset --hard HEAD~3
Reason:  commits have already been pushed; hard-resetting rewrites shared history
Safer:   git revert HEAD   # safer: creates an undo commit rather than erasing history

$ git-next guard -- git push --force
BLOCKED: git push --force
Reason:  force-pushing rewrites remote history and will break other developers' local branches
Safer:   git revert HEAD   # create an inverse commit instead

$ git-next guard -- git commit
ALLOWED: git commit
```

Exits 0 when allowed, 1 when blocked. Supports `--json` for scripting.

### Hook installer

```bash
git-next hook install            # installs a pre-push hook
git-next hook install --hook pre-commit
git-next hook list               # shows hook status
git-next hook uninstall          # removes the git-next managed hook
```

The hook calls `git-next ci --fail-on high` before every push, blocking pushes when high or critical severity rules are active.

### CI gate

```bash
git-next ci --fail-on high
git-next ci --fail-on critical --format github
```

`--fail-on` sets the minimum severity that causes exit 1:

| Flag | Blocks on |
|---|---|
| `--fail-on critical` | Force-push, tag rewrite, protected branch reset, history rewrite |
| `--fail-on high` | All of the above + diverged branches, integrity issues |
| `--fail-on medium` | All of the above + workflow hygiene (WIP commits, etc.) |
| `--fail-on all` | Everything including informational |

`--format github` emits GitHub Actions annotations:

```
::error::git-next R037: Force-push to shared branch
::warning::git-next R048: Long-lived feature branch - merge debt accumulating interest
```

**GitHub Actions example:**

```yaml
- name: Git safety check
  run: git-next ci --fail-on high --format github
```

### Rule explanations

```bash
git-next explain R037
```

```
R037: Force-push to shared branch

Why it matters:
  Force-pushing rewrites the remote branch history. Anyone who has already pulled commits that
  you are about to overwrite will have a diverged local branch with no clean resolution path.
  This is one of the most disruptive things you can do to a team.

Safer alternative:
  git revert HEAD            # create an undo commit instead
  git push --force-with-lease  # at minimum use this instead of --force
```

```bash
git-next rules list            # tabular view of all 58 rules
git-next --explain             # inline why for each active rule
```

### Shell completion

```bash
eval "$(git-next completion bash)"   # bash
eval "$(git-next completion zsh)"    # zsh
git-next completion fish | source    # fish
```

---

## How It Works

### Rule priority tiers

Rules are evaluated by priority. Higher priority rules can suppress contradictory lower-priority ones.

**100–90: "Put the keyboard down"**
- R037: Force-push to shared branch - this is how trust dies
- R038: Rewritten published tags - releases are now folklore
- R039: Reset on protected branch - muscle memory is not a justification
- R040: Submodule pointer rewrite without update - builds will fail creatively
- R041: Accidental history rewrite detected
- R021: Revert public commits (don't reset)
- R009: Merge in progress - complete or abort
- R010: Rebase in progress - complete or abort
- R011: Cherry-pick in progress - complete or abort
- R001: Detached HEAD detected
- R032: Merge on protected branches (not rebase)

**89–60: Repo integrity issues**
- R042: Conflicted files staged
- R043: Binary files without LFS
- R044: Line ending normalization conflict
- R045: Submodule detached HEAD
- R046: Shallow clone doing history ops
- R006: Branch diverged
- R034: No upstream configured
- R031: Rebase feature branches
- R035: Merged branches ready for cleanup
- R036: Gone remote branches
- R033: Continue with merge if merge history exists

**59–30: Workflow hygiene**
- R047: Work on main instead of feature branch
- R048: Long-lived feature branch
- R049: Squash recommended before merge
- R050: WIP commit on shared branch
- R051: Rebase recommended instead of merge
- R005: Pull when behind and clean
- R004: Push local commits
- R030: Fast-forward pull
- R020: Soft reset local commits (≤3)
- R022: Interactive rebase for many commits
- R003: Commit staged files
- R002: Stage and commit modified files

**29–10: Mild suggestions**
- R052: Commit message quality warning
- R053: Amend last commit suggested
- R054: Unpushed local tags
- R007: Add untracked files
- R055: Stash stack growing
- R008: Apply stash

**<10: Informational trivia**
- R056: Repo size growing unusually fast
- R057: Inactive branches detected
- R058: Detached HEAD but clean

**[Full rule documentation →](docs/rules/README.md)**

### Suppression logic

When one command is already the correct action, `git-next` suppresses conflicting lower-priority suggestions so you get one clear path forward — not a menu of contradictions.

**Active operations:**
- `merge --continue/--abort` suppresses almost all other operations
- `rebase --continue/--abort` suppresses almost all other operations
- `cherry-pick --continue/--abort` suppresses almost all other operations

**Normal operations:**
- `revert` suppresses `reset` (can't reset public commits)
- `merge` suppresses `rebase` (chosen strategy wins)
- `rebase` suppresses `pull` (rebase handles sync)
- `reset` suppresses `commit` (undoing commits)
- `commit` suppresses `add` (staged files take priority over untracked files)

---

## Configuration

Create `.git-next.yaml` in your repo root (or `~/.config/git-next/config.yaml` for user-level defaults):

```yaml
protected_branches:
  - main
  - master
  - develop
  - staging
  - production

rules:
  disabled:
    - R007  # don't nag about untracked files
    - R055  # don't suggest stash cleanup

  parameters:
    R020:
      max_commits: 5   # allow soft reset up to 5 commits (default: 3)
    R048:
      max_days: 21     # flag branches older than 21 days (default: 14)
```

See [`.git-next.yaml.example`](.git-next.yaml.example) for the full template.

---

## Exit Codes

| Code | Meaning |
|---|---|
| `0` | Repository is clean, no actions needed |
| `1` | Actions suggested (or an error occurred) |

```bash
if ! git-next --compact; then
    echo "Repository needs attention before proceeding"
fi
```

---

## Design Principles

1. **Never lie** — if we don't know, we don't guess
2. **Conservative** — suggest the safest move
3. **Explainable** — every suggestion has a reason (`git-next explain <rule>`)
4. **Deterministic** — same state always gives same advice
5. **No vibes** — pure state analysis, no heuristics

---

## Development

### Build and test

```bash
go build -o git-next ./cmd/git-next/
go test ./...
go vet ./...
```

### Adding rules

1. Add state fields to `pkg/model/types.go`
2. Populate them in the matching `internal/repo/state_*.go` file
3. Add a rule function in the appropriate `internal/rules/rules_*.go` file (choose the tier by danger level)
4. Set `Severity` and `Destructive` on the `RuleDef`
5. Update the suppression map in `internal/engine/engine.go` if needed
6. Document in `docs/rules/` and add an explanation in `internal/explain/explain.go`

### Project structure

```
cmd/git-next/           CLI entrypoint + subcommand handlers
internal/
  config/               YAML config loading
  repo/                 Repository state collection (modular by concern)
  rules/                Rule definitions (modular by danger tier)
  engine/               Rule evaluation + suppression logic
  output/               Human, JSON, compact, and agent formatters
  action/               Interactive action executor
  policy/               Command safety policy for guard mode
  hook/                 Git hook installer
  explain/              Rule explanation text (all 58 rules)
  completion/           Shell completion scripts
  mcp/                  MCP server (classifier, executor, tools, JSON-RPC server)
  testutil/             Synthetic git repo builder for tests
pkg/model/              Shared types (RepoState, Advice, AgentOutput)
docs/
  rules/                Comprehensive rule documentation
  schema/               JSON Schema for --agent output
```

---

## License

MIT

## Why This Exists

...Don't fuck up?
