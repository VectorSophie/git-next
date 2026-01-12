# Repo Integrity Issues (Priority 89-60)

## Issues that need immediate attention

These rules catch problems with repository health, data integrity, and configuration that can cause builds to fail, code to be lost, or development to grind to a halt.

---

## R042: Conflicted files staged
**Priority: 89**

**What it detects:**
- Staged files containing `<<<<<<<`, `=======`, or `>>>>>>>` markers
- Conflict markers that survived conflict resolution

**What to do:**
1. Edit the file and remove ALL conflict markers
2. Choose which code to keep (or merge both)
3. Test that it actually works
4. Then `git add` and `git commit`

**Why it matters:**
Conflict markers are not valid syntax in any language. Your code won't compile/run. CI will fail. Teammates will be sad.

---

## R043: Binary files changed without LFS
**Priority: 85**

```
git lfs track <pattern> && git add .gitattributes
```

**What it detects:**
- Binary files > 1MB being added to git
- Large files not tracked by Git LFS
- Binaries committed directly to repository

**What to do:**
```bash
# Install git-lfs
git lfs install

# Track binary file types
git lfs track "*.zip"
git lfs track "*.exe"
git lfs track "*.pdf"

# Commit the .gitattributes file
git add .gitattributes
git commit -m "Track binaries with LFS"

# Now add your binaries
git add bigfile.zip
git commit -m "Add binary (via LFS)"
```

**Why it matters:**
Please don't commit binaries at all. Use artifact storage (S3, GitHub Releases, package registries). They are huge and impossible to manage.

---

## R044: Line ending normalization conflict
**Priority: 82**

```
git config core.autocrlf true (or false)
```

**What it detects:**
- Git warnings about CRLF/LF line ending changes
- Mixed line endings causing constant modifications
- The eternal Windows vs Unix line ending war

**What to do:**

**Option 1: Team decision (recommended)**
Create `.gitattributes` in repo root:
```
# Force LF everywhere
* text=auto eol=lf

# Or force CRLF for Windows-specific files
*.bat text eol=crlf
*.ps1 text eol=crlf
```

**Option 2: Individual config**
```bash
# Windows: Convert to CRLF on checkout, LF on commit
git config core.autocrlf true

# Linux/Mac: Keep LF everywhere
git config core.autocrlf input
```

**Fix existing files:**
```bash
# Re-normalize line endings
git add --renormalize .
git commit -m "Normalize line endings"
```

---

## R045: Submodule detached HEAD
**Priority: 81**

```
cd <submodule> && git checkout <branch>
```

**What it detects:**
- Submodules pointing to specific commits (detached HEAD)
- Submodules not on any branch

**What to do:**
```bash
cd lib/vendor
git checkout main  # Get on a branch
git pull           # Get latest

# Now make your changes
git add .
git commit -m "Fix bug"
git push

cd ..
git add lib/vendor  # Update submodule pointer
git commit -m "Update vendor submodule"
git push
```

**Why it matters:**
Work in detached HEAD submodules can be lost. Submodule pointers should reference commits that exist on branches, not floating commits.

---

## R046: Shallow clone doing history ops
**Priority: 80**

```
git fetch --unshallow
```

**What it detects:**
- Shallow clone (limited history depth)
- Attempts to use `git rebase`, `git blame`, `git bisect`, or `git log --all`

**Git will lie to you politely:**
```bash
git clone --depth 1 repo  # Only fetch latest commit
cd repo

git log --all
# Shows: 1 commit
# Reality: 10,000 commits

git blame file.js
# Shows: Everything authored by "Initial commit"
# Reality: 15 different authors

git rebase main
# Error: unshallow first
# Reality: Can't rebase without history
```

**What to do:**
```bash
# Convert shallow clone to full clone
git fetch --unshallow

# Or clone with full history from the start
git clone repo  # No --depth flag
```

**Why it matters:**
Shallow clones save bandwidth but break history operations. If you need to rebase, blame, or bisect, you need full history.

---

## R006: Branch has diverged
**Priority: 80**

```
git rebase origin/<branch> OR git merge origin/<branch>
```

**What it detects:**
- Local commits + remote commits (both ahead and behind)
- Branch has diverged from its upstream

**What to do:**
```bash
# Check the situation
git status
# Your branch and 'origin/main' have diverged

# Option 1: Rebase (feature branches, linear history)
git rebase origin/main

# Option 2: Merge (protected branches, preserve history)
git merge origin/main

# Then push
git push
```
---

## R034: No upstream configured
**Priority: 75**

```
git branch --set-upstream-to=origin/<branch>
```

**What it detects:**
- Current branch doesn't track a remote branch
- Created a local branch but never pushed it

**What to do:**
```bash
# If remote branch exists
git branch --set-upstream-to=origin/feature-branch

# If remote branch doesn't exist, push and track
git push -u origin feature-branch

# Now git push/pull work without arguments
```

**Why it matters:**
Without an upstream, you have to specify remote and branch every time: `git push origin feature-branch`. Setting upstream makes `git push` and `git pull` just work.

---

## R031: Feature branch diverged - rebase
**Priority: 70**

```
git rebase origin/<branch>
```

**What it detects:**
- Feature branch has diverged from its tracking branch
- Not on a protected branch (main/master/develop)

**Keep linear history:**
```bash
# Teammate pushed to feature branch
git fetch
git status
# Your branch and 'origin/feature' have diverged

# Rebase to keep linear history
git rebase origin/feature

# Or pull with rebase
git pull --rebase
```

**Why it matters:**
- Keeps history linear and easy to read
- Makes bisecting and reverting easier
- Produces cleaner git log
- Avoids unnecessary merge commits

---

## R035: Merged branches ready for cleanup
**Priority: 65**

```
git branch -d <branch>
```

**What it detects:**
- Local branches that have been fully merged
- Excludes current branch and protected branches

**What to do:**
```bash
# See which branches are merged
git branch --merged

# Delete a merged branch
git branch -d old-feature

# Delete multiple merged branches
git branch --merged | grep -v "main\|master\|develop" | xargs git branch -d
```
---

## R036: Gone remote branches
**Priority: 62**

```
git branch -d <branch>
```

**What it detects:**
- Local branches whose upstream has been deleted on remote
- Typically happens after PR merge + branch deletion on GitHub

**What to do:**
```bash
# Fetch to update remote tracking
git fetch --prune

# Delete gone branches
git branch -d feature-1 feature-2

# Or delete all gone branches
git branch -vv | grep ': gone]' | awk '{print $1}' | xargs git branch -d
```

---

## R033: Existing merge history - continue with merge
**Priority: 60**

```
git merge origin/<branch>
```

**What it detects:**
- Branch has merge commits in its history
- Behind remote and needs to sync

**What to do:**
```bash
# Branch has merge commits, so continue with merge
git merge origin/main

# Don't rebase - would rewrite existing merges
```

**Why it matters:**
If your branch already has merge commits, rebasing would rewrite that history and create duplicate commits. Stick with merging for consistency.

---


