# Mild Suggestions (Priority 29-10)

## Helpful hints and optimizations

These are nice-to-have improvements - quality of life suggestions, housekeeping, and polish. Not urgent, but ignoring them forever will make your repository messy.

---

## R052: Commit message quality warning
**Priority: 25**

```
git commit --amend
```

**What it detects:**
- Commit message too short (< 5 characters)
- Message is just "." or ".."
- Message doesn't start with a verb

**Why it matters:**
Commit messages are the only documentation for why code changed. Good messages help with:
- Understanding changes during code review
- Debugging ("When did this break? What was the intent?")
- Generating changelogs
- Bisecting to find when bugs were introduced

---

## R053: Amend last commit suggested
**Priority: 23**

```
git commit --amend
```

**What it detects:**
- 2+ staged files waiting to commit
- Last commit was within 5 minutes of previous commit
- Looks like a quick follow-up fix

**What to do:**
```bash
# Stage your changes
git add forgotten-file.js

# Amend the previous commit
git commit --amend

# Or amend without changing message
git commit --amend --no-edit

# Or amend just the message
git commit --amend -m "Better message"
```

---

## R054: Unpushed local tags
**Priority: 21**

```
git push --tags
```

**What it detects:**
- Local tags that don't exist on remote
- Tags created but never shared

**Schrödinger's release:**
```bash
# You tagged a release
git tag v1.0.0
git push  # Pushes commits, but NOT tags

# Your machine: v1.0.0 exists
# Remote: v1.0.0 doesn't exist
# CI: Can't find v1.0.0
# Teammates: What's v1.0.0?
# Release: In superposition state
```

**What to do:**
```bash
# Push all tags
git push --tags

# Push a specific tag
git push origin v1.0.0

# Push commits and tags together
git push --follow-tags  # Only pushes annotated tags
```

**Why it matters:**
Tags mark important points (releases, milestones). If they only exist locally, they're useless for:
- CI/CD pipelines
- Release automation
- Team coordination
- Docker image tagging
- Package versioning

---

## R007: Untracked files present
**Priority: 20**

```
git add <files>
```

**What it detects:**
- Files in working tree not tracked by git
- New files that might need to be committed

**What to do:**
```bash
# Review untracked files
git status

# Add specific files
git add newfile.js

# Add all untracked files
git add .

# Interactively choose what to add
git add -p
```

---

## R055: Big ass stash stack
**Priority: 18**

```
git stash pop OR git stash clear
```

**What it detects:**
- More than 3 stashes
- Oldest stash is more than 7 days old

**What to do:**
```bash
# Review stashes
git stash list
git stash show stash@{0}  # See what's in a stash

# Apply and remove
git stash pop  # Apply most recent, remove from stack

# Apply without removing
git stash apply stash@{2}

# Delete specific stash
git stash drop stash@{3}

# Delete all stashes (dangerous!)
git stash clear
```

---

## R008: Stash exists
**Priority: 15**

```
git stash pop
```

**What it detects:**
- At least one stash exists
- Might want to resume that work

**What to do:**
```bash
# See what's stashed
git stash list
git stash show

# Apply most recent stash
git stash pop

# Apply specific stash
git stash apply stash@{1}

# Create a branch from stash
git stash branch new-branch-name
```

**Why it matters:**
- Quick context switch
- Pull with uncommitted changes
- Try something experimental
- Clean working tree temporarily