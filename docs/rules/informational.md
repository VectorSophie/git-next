# Informational Trivia (Priority <10)

## Just FYI

These rules provide awareness about repository health and state. Nothing is wrong - these are just things you might want to know.

---

## R056: Repo size growing unusually fast
**Priority: 9**

```
git gc --aggressive
```

**What it detects:**
- Repository `.git` directory larger than 100MB
- Potential optimization opportunities

**What to do:**

**Option 1: Garbage collection**
```bash
# Light cleanup
git gc

# Aggressive cleanup (slower but more thorough)
git gc --aggressive

# Clean up unreachable objects
git reflog expire --expire=now --all
git gc --prune=now
```

**Option 2: Find large files**
```bash
# Find largest files in history
git rev-list --objects --all |
  git cat-file --batch-check='%(objecttype) %(objectname) %(objectsize) %(rest)' |
  sed -n 's/^blob //p' |
  sort --numeric-sort --key=2 |
  tail -20

# Use BFG Repo-Cleaner to remove them
# https://rtyley.github.io/bfg-repo-cleaner/
```

**Option 3: Shallow clone for CI**
```bash
# CI doesn't need full history
git clone --depth 1 https://github.com/user/repo
```

**Option 4: Use Git LFS**
```bash
# Track large files with LFS instead
git lfs track "*.zip"
git lfs track "*.pdf"
```

---

## R057: Inactive branches detected
**Priority: 8**

```
git branch -d <branch>
```

**What it detects:**
- Local branches with no commits in 90+ days
- Stale development work

**What to do:**
```bash
# Review old branches
git for-each-ref --sort=-committerdate refs/heads/ --format='%(committerdate:short) %(refname:short)'

# Delete if no longer needed
git branch -d old-branch-name

# Force delete if not merged (careful!)
git branch -D really-old-unmerged-branch
```

**Questions to ask:**
- Was this work ever completed?
- Is this still relevant?
- Did this ship in a different form?
- Can I remember what this was for?

**When to keep:**
- Long-running feature development
- On-hold work you plan to resume
- Experiments you reference occasionally
- Branches with valuable work not yet merged

**When to delete:**
- Work that shipped
- Abandoned experiments
- Duplicate efforts
- Branches you can't remember creating

**Archive instead of delete:**
```bash
# Create tag to preserve branch pointer
git tag archive/old-feature old-feature
git branch -d old-feature

# Can recreate branch later if needed
git checkout -b old-feature archive/old-feature
```

---

## R058: Detached HEAD but clean
**Priority: 5**

```
git checkout <branch>
```

**What it detects:**
- HEAD is detached (not on a branch)
- Working tree is clean
- Nothing staged

**What to do:**

**If you're done looking:**
```bash
# Return to your branch
git checkout main
```

**If you want to keep this state:**
```bash
# Create a branch here
git checkout -b new-branch-name
```

**If you made changes accidentally:**
```bash
# Stash them
git stash

# Or commit them to a new branch
git checkout -b save-my-work
git add .
git commit -m "Save work from detached HEAD"
```

**Why it matters:**
Detached HEAD makes changes lost to time.

