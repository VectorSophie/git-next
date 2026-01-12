# Workflow Hygiene (Priority 59-30)

## Best practices and normal workflow

These rules guide your day-to-day git operations - branching, committing, merging, rebasing, pushing. This is where git-next helps you follow best practices without thinking about it.

---

## R047: Work on main instead of feature branch
**Priority: 58**

```
git checkout -b feature/<name>
```

**What it detects:**
- Local commits on protected branches (main/master/develop)
- Commits that aren't merge commits

**What to do:**
```bash
# Move your work to a feature branch
git branch feature/my-work      # Create branch at current commit
git reset --hard origin/main    # Reset main to match remote
git checkout feature/my-work    # Switch to feature branch

# Or use git reset --soft to uncommit
git reset --soft origin/main    # Uncommits but keeps changes staged
git checkout -b feature/my-work # Create and switch to feature branch
git commit -m "Add feature"     # Re-commit on feature branch
```

**Why it matters:**
Protected branches should only receive changes via merge/PR. Direct commits bypass code review, CI checks, and team collaboration.

---

## R048: Long-lived feature branch
**Priority: 56**

```
git merge main (or rebase)
```

**What it detects:**
- Feature branch older than 14 days (configurable)
- Branch is behind main/master
- Active development continues on main

**What to do:**
```bash
# Option 1: Merge main into your branch regularly
git checkout feature/big-project
git merge main
# Resolve conflicts incrementally

# Option 2: Rebase on main (rewrites history)
git checkout feature/big-project
git rebase main
# Keep linear history, but requires force push

# Best practice: Sync at least weekly
```

**Config:**
```yaml
# .git-next.yaml
rules:
  parameters:
    R048:
      max_days: 21  # Flag branches older than 21 days
```

---

## R005: Behind remote and clean 
**Priority: 55**

```
git pull
```

**What it detects:**
- Branch is behind its upstream
- Working tree is clean (no uncommitted changes)

**What to do:**
```bash
git pull
```

**Why it matters:**
Your working tree is clean, so there's nothing to lose. Pulling remote changes will fast-forward your branch or create a merge commit.

---

## R049: Squash recommended before merge
**Priority: 52**

```
git rebase -i HEAD~N
```

**What it detects:**
- Many small commits with "noisy" messages
- Commits like: "fix", "oops", "wip", "typo", ".", "update", "test"
- More than 30% of commits are noisy
- More than 3 commits total

**What to do:**
```bash
# Interactive rebase to squash
git rebase -i HEAD~10

# Editor opens, change 'pick' to 'squash' or 'fixup'
pick abc123 Add feature
fixup def456 fix typo
fixup 789abc oops
fixup 123def actually fix it
fixup 456ghi test

# Result: One clean commit
* Add feature
```

**Why it matters:**
Git history is for understanding what changed and why, not documenting every keystroke. Clean history helps with:
- Code review
- Bisecting bugs
- Understanding project evolution
- Generating changelogs

---

## R050: WIP commit on shared branch
**Priority: 51**

```
git commit --amend
```

**What it detects:**
- Commits on protected branches with WIP markers
- Messages containing: "wip", "temp", "debug", "todo", "fixme"

**This is not your personal notebook:**
```bash
git checkout main
git commit -m "WIP: debugging production issue"
git push

# Congrats, production now has a commit titled "WIP"
# Release notes: "WIP: debugging production issue"
# Future developers: "What was the issue? Is it fixed?"
```

**What to do:**
```bash
# If not pushed yet
git commit --amend
# Write a proper commit message

# If already pushed to feature branch
git commit --amend
git push --force-with-lease

# If already pushed to protected branch
# You can't amend (history is public)
# You're stuck with it
# Learn from this
```

---

## R051: Rebase recommended instead of merge
**Priority: 50**

```
git rebase main
```

**What it detects:**
- Merge commit on a feature branch
- Not on a protected branch

**What to do:**
```bash
# You did:
git merge main

# Creates merge commit:
*   Merge branch 'main' into feature
|\
| * (main updates)
* | (your work)

# Better:
git rebase main

# Linear history:
* (your work, rebased)
* (main updates)
* (main history)
```

---

## R004: Local commits ready to push
**Priority: 50**

```
git push
```

**What it detects:**
- Branch is ahead of upstream
- Branch is not behind (no remote changes)
- Working tree is clean

**What to do:**
```bash
git push
```

---

## R030: Can fast-forward - safe to pull
**Priority: 48**

```
git pull --ff-only
```

**What it detects:**
- Behind remote
- Not ahead of remote (no local commits)
- Working tree is clean

**What to do:**
```bash
git pull --ff-only
# Or just
git pull
```

**Why `it matters:**
Guarantees a clean, fast-forward merge. If there are any complications, it will abort rather than creating a merge commit.

---

## R020: Local commits (≤3) can be soft reset
**Priority: 45**

```
git reset --soft HEAD~N
```

**What it detects:**
- Local unpushed commits (≤3)
- Commits haven't been pushed to remote

**What to do:**
```bash
# Undo commits but keep changes staged
git reset --soft HEAD~2  # Undo last 2 commits

# Changes are still staged
git status
# Changes to be committed: ...

# Now you can:
# - Recommit differently
# - Add more changes
# - Split into different commits
```

---

## R022: Too many local commits - use interactive rebase
**Priority: 42**

```
git rebase -i HEAD~N
```

**What it detects:**
- More than 3 local unpushed commits (configurable)
- Commits haven't been pushed

**What to do:**
```bash
git rebase -i HEAD~10

# Interactive editor opens
pick abc123 Commit 1
pick def456 Commit 2
pick 789ghi Commit 3
...

---

## R003: Staged files waiting for commit
**Priority: 38**

```
git commit
```

**What it detects:**
- Files in staging area
- Ready to commit

**What to do:**
```bash
git commit
# Or
git commit -m "Description of changes"
```

---

## R002: Modified files not staged
**Priority: 35**

```
git add <files> && git commit
```

**What it detects:**
- Modified files in working tree
- Nothing staged yet

**What to do:**
```bash
# Stage specific files
git add file1.js file2.js
git commit -m "Update files"

# Stage all changes
git add .
git commit -m "Update all files"

# Stage and commit in one step
git commit -am "Update files"  # Only for tracked files
```

---

