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

**100–90: You are about to do something dangerous**
- R021: Revert public commits (don't reset)
- R009: Merge in progress - complete or abort
- R010: Rebase in progress - complete or abort
- R011: Cherry-pick in progress - complete or abort
- R001: Detached HEAD
- R032: Merge on protected branches

**89–60: Repo integrity issues**
- R006: Branch diverged
- R034: No upstream configured for current branch
- R031: Rebase feature branches
- R035: Merged branches ready for cleanup
- R036: Gone remote branches - local cleanup needed
- R033: Continue with merge if merge history exists

**59–30: Workflow hygiene**
- R005: Pull when behind and clean
- R004: Push local commits
- R030: Fast-forward pull
- R020: Soft reset local commits (≤3)
- R022: Interactive rebase for many commits
- R003: Commit staged files
- R002: Stage and commit modified files

**29–10: Mild suggestions**
- R007: Add untracked files
- R008: Apply stash

**<10: Informational trivia**
- (Reserved for future low-priority rules)

### New Rules Detail

**Active Operations (R009-R011)**

These rules detect when you have an incomplete git operation. They have the highest priority because finishing (or aborting) these operations is critical before doing anything else:

- **R009: Merge in progress** - Detects when a merge has conflicts or is paused. You must either resolve conflicts and continue, or abort to return to a clean state.
- **R010: Rebase in progress** - Detects when a rebase is paused (due to conflicts or editing commits). Complete the rebase or abort it.
- **R011: Cherry-pick in progress** - Detects when a cherry-pick operation is incomplete. Either continue or abort.

**Branch Health (R034-R036)**

These rules help maintain repository cleanliness and proper tracking:

- **R034: No upstream configured** - Your current branch doesn't track a remote branch. This is common after creating a new branch locally. You should set an upstream to enable push/pull operations.
- **R035: Merged branches ready for cleanup** - Detects local branches that have already been merged into your current branch (excluding protected branches like main/master). These can be safely deleted to reduce clutter.
- **R036: Gone remote branches** - Detects local branches whose upstream has been deleted on the remote (usually after a PR is merged and branch deleted on GitHub/GitLab). These orphaned local branches should be cleaned up.

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

## Protected Branches

By default, these branches are considered protected:
- `main`
- `master`
- `develop`
- `production`

On protected branches, `git-next` will always suggest merge over rebase.

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
│   ├── repo/           # Repository state collection
│   ├── rules/          # Rule definitions
│   ├── engine/         # Rule evaluation + suppression
│   └── output/         # Output formatters
└── pkg/model/          # Public types
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

1. Add the rule function in `internal/rules/rules.go`
2. Add the rule definition to `AllRules()`
3. Set appropriate priority
4. Update suppression map if needed

## License

MIT

## Why This Exists

Git gives you 47 ways to do everything. Most advice says "it depends" or gives you vibes-based heuristics. This tool looks at your actual repository state and tells you what to do. That's it.

No blog posts. No flame wars. Just: "Given who has this history, here's the least harmful move."
