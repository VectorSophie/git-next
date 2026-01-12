# git-next

Git advice that doesn't lie.

## What It Does

`git-next` analyzes your repository state and tells you what to do next. No philosophy. No guessing. Just honest advice based on who has the history.

It understands:
- When you can safely reset vs. when you need to revert
- When to rebase vs. when to merge
- What order to do things in (commit before push, pull before rebase, etc.)
- When you're about to do something regrettable

## Installation

### From Source

```bash
go install github.com/VectorSophie/git-next/cmd/git-next@latest
```

### Manual Build

```bash
git clone https://github.com/VectorSophie/git-next
cd git-next
go build -o git-next cmd/git-next/main.go
```

### Using the install script

```bash
curl -sfL https://raw.githubusercontent.com/VectorSophie/git-next/main/scripts/install.sh | sh
```

## Usage

```bash
# Show what to do next
git-next

# Show all advice including suppressed items
git-next --all

# Output as JSON
git-next --json

# Compact one-line output (good for scripts/prompts)
git-next --compact

# Debug mode (show repo state)
git-next --debug

# Interactive mode - execute suggested actions
git-next --action
```

## Example Output

### Default Mode

```
Git Next - Suggested Actions
═══════════════════════════════

→ [R003] Staged files waiting for commit
  Command: git commit

→ [R004] Local commits ready to push
  Command: git push

───────────────────────────────
Active: 2  Suppressed: 0
```

### Interactive Mode (`--action`)

```
Git Next - Interactive Action Mode
═══════════════════════════════════

1. [R003] Staged files waiting for commit
   Command: git commit
   Priority: 38

2. [R004] Local commits ready to push
   Command: git push
   Priority: 50

Select action to execute (1-2, or 'q' to quit): 1

About to execute: git commit
Proceed? (y/N): y

───────────────────────────────
Executing...

[... git commit output ...]

✓ Command completed successfully
```

The `--action` flag enables interactive mode where you can:
- Browse all available actions with their priorities
- Select which action to execute
- Automatically resolve command placeholders (`<files>`, `<branch>`, `HEAD~N`)
- Confirm before execution
- See real-time command output

## How It Works

### Rule Priority

Rules are evaluated by priority. Higher priority rules can suppress lower priority ones.

**100–90: "Put the keyboard down"**
- R037: Force-push to shared branch - this is how trust dies
- R038: Rewritten published tags - releases are now folklore
- R039: Reset on protected branch - muscle memory is not a justification
- R040: Submodule pointer rewrite without update - builds will fail creatively
- R041: Accidental history rewrite detected - you don't get to pretend this was fine
- R021: Revert public commits (don't reset)
- R009: Merge in progress - complete or abort
- R010: Rebase in progress - complete or abort
- R011: Cherry-pick in progress - complete or abort
- R001: Detached HEAD detected
- R032: Merge on protected branches (not rebase)

**89–60: Repo integrity issues**
- R042: Conflicted files staged - if <<<<<<< is in the diff, stop pretending
- R043: Binary files without LFS - Git is not a landfill
- R044: Line ending normalization conflict - someone's editor declared war
- R045: Submodule detached HEAD - time capsule mode engaged
- R046: Shallow clone doing history ops - Git will lie to you politely
- R006: Branch diverged - need to sync
- R034: No upstream configured for current branch
- R031: Rebase feature branches - keep linear history
- R035: Merged branches ready for cleanup
- R036: Gone remote branches - local cleanup needed
- R033: Continue with merge if merge history exists

**59–30: Workflow hygiene**
- R047: Work on main instead of feature branch - you skipped the whole process part
- R048: Long-lived feature branch - merge debt accumulating interest
- R049: Squash recommended before merge - many noisy commits
- R050: WIP commit on shared branch - this is not your personal notebook
- R051: Rebase recommended instead of merge - keep linear history
- R005: Pull when behind and clean
- R004: Push local commits
- R030: Fast-forward pull
- R020: Soft reset local commits (≤3)
- R022: Interactive rebase for many commits
- R003: Commit staged files
- R002: Stage and commit modified files

**29–10: Mild suggestions**
- R052: Commit message quality warning - Git logs are for humans
- R053: Amend last commit suggested - you knew this already
- R054: Unpushed local tags - Schrödinger's release
- R055: Stash stack growing - you're hoarding unfinished thoughts
- R007: Add untracked files
- R008: Apply stash

**<10: Informational trivia**
- R056: Repo size growing unusually fast - just so you're aware
- R057: Inactive branches detected - archaeology opportunity
- R058: Detached HEAD but clean - nothing wrong, just vibes

**[View comprehensive rule documentation →](docs/rules/README.md)**

### Suppression Logic

Commands suppress other commands to prevent conflicting advice:

**Active Operations (highest priority):**
- `merge --continue/--abort` suppresses almost all operations (must finish or abort first)
- `rebase --continue/--abort` suppresses almost all operations
- `cherry-pick --continue/--abort` suppresses almost all operations

**Normal Operations:**
- `revert` suppresses `reset` (can't reset public commits)
- `merge` suppresses `rebase` (chosen strategy wins)
- `rebase` suppresses `pull` (rebase handles sync)
- `reset` suppresses `commit` (undoing commits)

This ensures you get **one clear path forward**, not a menu of contradictions.

## Configuration

git-next supports YAML-based configuration for customization (v0.2.0+). Create a `.git-next.yaml` file in your repository root or `~/.config/git-next/config.yaml` for user-level defaults.

### Example Configuration

```yaml
# Protected branches - branches that should use merge instead of rebase
protected_branches:
  - main
  - master
  - develop
  - staging
  - production

# Rule configuration
rules:
  # Disable specific rules
  disabled:
    - R007  # Don't nag about untracked files
    - R055  # Don't suggest stash cleanup

  # Customize rule parameters
  parameters:
    R020:
      max_commits: 5  # Allow soft reset up to 5 commits (default: 3)
    R048:
      max_days: 21    # Flag feature branches older than 21 days (default: 14)
```

See `.git-next.yaml.example` for a full configuration template.

### Protected Branches

By default, these branches are considered protected:
- `main`
- `master`
- `develop`
- `production`

On protected branches, `git-next` will always suggest merge over rebase to preserve merge history. You can customize this list in `.git-next.yaml`.

## Exit Codes

- `0`: Repository is clean, no actions needed
- `1`: Actions suggested (or error occurred)

This makes it easy to use in scripts:

```bash
if ! git-next --compact; then
    echo "Repository needs attention"
fi
```

## Design Principles

1. **Never lie** - If we don't know, we don't guess
2. **Conservative** - Suggest the safest move
3. **Explainable** - Every suggestion has a reason
4. **Deterministic** - Same state always gives same advice
5. **No vibes** - Pure state analysis, no heuristics

## Repository State

The tool examines:

- Working tree status (dirty, staged, modified, untracked)
- Branch relationship to remote (ahead, behind)
- Commit history (pushed vs local)
- Branch type (protected vs feature)
- Merge history (has merge commits)
- Stash status
- Detached HEAD state

## Development

### Project Structure

```
git-next/
├── cmd/git-next/       # CLI entrypoint
├── internal/
│   ├── config/         # Configuration system (YAML)
│   ├── repo/           # Repository state collection (modular)
│   │   ├── state.go              # Main collector
│   │   ├── state_dangerous.go   # Dangerous operation detection
│   │   ├── state_integrity.go   # Repo integrity checks
│   │   ├── state_workflow.go    # Workflow hygiene
│   │   └── state_suggestions.go # Suggestions + informational
│   ├── rules/          # Rule definitions (modular by danger level)
│   │   ├── rules.go               # Main aggregator
│   │   ├── rules_dangerous.go     # Priority 100-90
│   │   ├── rules_integrity.go     # Priority 89-60
│   │   ├── rules_workflow.go      # Priority 59-30
│   │   ├── rules_suggestions.go   # Priority 29-10
│   │   └── rules_informational.go # Priority <10
│   ├── engine/         # Rule evaluation + suppression
│   ├── output/         # Output formatters
│   └── action/         # Interactive action executor
├── pkg/model/          # Public types
├── docs/rules/         # Comprehensive rule documentation
└── .git-next.yaml.example  # Configuration template
```

### Building

```bash
go build -o git-next cmd/git-next/main.go
```

### Testing

```bash
go test ./...
```

### Adding Rules

The rule system is modular - each danger level has its own file:

1. **Choose the appropriate danger level:**
   - `rules_dangerous.go` (100-90): Dangerous operations
   - `rules_integrity.go` (89-60): Repo integrity issues
   - `rules_workflow.go` (59-30): Workflow hygiene
   - `rules_suggestions.go` (29-10): Mild suggestions
   - `rules_informational.go` (<10): Informational trivia

2. **Add state collection** in corresponding `internal/repo/state_*.go` file

3. **Add state fields** in `pkg/model/types.go` (grouped by danger level)

4. **Add rule function** in the appropriate `internal/rules/rules_*.go` file

5. **Add rule definition** to the module's function (e.g., `DangerousRules()`)

6. **Update suppression map** in `internal/engine/engine.go` if needed

7. **Document the rule** in `docs/rules/*.md`

Example: Adding a new workflow rule (priority 55):
```go
// In internal/rules/rules_workflow.go
func R999(state model.RepoState) bool {
    return state.SomeNewCondition
}

// Add to WorkflowRules()
{
    ID:          "R999",
    Check:       R999,
    Command:     "git do-something",
    Description: "Your new rule description",
    Priority:    55,
},
```

## License

MIT

## Why This Exists

...Dont fuck up?
