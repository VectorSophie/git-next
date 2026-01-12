# Dangerous Operations (Priority 100-90)

## "Put the keyboard down"

These are critical warnings about operations that can break EVERYTHING.

---

## R037: Force-push to shared branch
**Priority: 100**

```
# DO NOT git push --force on shared branches!
```

**What it detects:**
- Rebase/amend on protected branches (main, master, develop, etc.)
- Changes that would require `--force` to push

**Why it matters**
You are breaking their local branches and creates merge conflicts for the entire team.

**What to do:**
- **On main/protected branches:** Use `git revert` instead of amending or rebasing
- **On feature branches:** Check with your team before force-pushing
- **If you must:** Use `--force-with-lease` (not on protected branches!)

---

## R038: Rewritten published tags
**Priority: 100**

```
# DO NOT rewrite published tags!
```

**What it detects:**
- Tags that exist remotely but point to different commits locally
- Attempts to move or delete pushed tags

**Why it matters:**
TAG YOUR FUCKING RELEASES

**What to do:**
- Create a NEW tag (e.g., v1.0.1) instead of moving v1.0.0
- Document what went wrong in release notes
- Accept that the old tag is permanent history

**Releases are now folklore:**
```bash
git tag -f v1.0.0 <new-commit>  
git push --force --tags          

```

---

## R039: Reset on protected branch
**Priority: 100**

```
# DO NOT reset on protected branches!
```

**What it detects:**
- `git reset` operations in reflog on main/master/develop/production

**Why it matters:**
Even `git reset --soft` on shared branches creates divergence.

**What to do:**
- Use `git revert <commit>` to undo changes safely
- Create a fix commit instead of erasing history
- Switch to a feature branch for experimental work

---

## R040: Submodule pointer rewrite without update
**Priority: 100**

```
git submodule update --remote
```

**What it detects:**
- Submodule pointer changes in the index
- Submodule commits that don't exist or haven't been pulled

**Why it matters:**
If you update the submodule pointer without pushing the submodule changes, you are pointing to a nonexistant submodule.

**What to do:**
1. `cd submodule/`
2. `git push` (push submodule changes first)
3. `cd ..`
4. `git add submodule/`
5. `git commit -m "Update submodule"`

---

## R041: Accidental history rewrite
**Priority: 100**

```
# Accidental history rewrite detected - you don't get to pretend this was fine
```

**What it detects:**
- Rebase or filter-branch operations AFTER commits were pushed
- Changes to commits that others have already pulled

**Why it matters:**
When you rebase commits that others have pulled, you create parallel timelines. Their work is now based on commits that "don't exist" in your rewritten history.

**What to do:**
- **If not pushed yet:** Rebase freely on your feature branch
- **If already pushed:** DON'T rebase shared commits
- **If you already did:** Communicate with your team immediately
- **On protected branches:** This should never happen

---

## R021: Last commit was pushed - use revert
**Priority: 100**

```
git revert HEAD
```

**What it detects:**
- Last commit exists on remote
- Attempts to amend or reset pushed commits

**Why it matters:**
Once a commit is pushed, it's public history. Other people may have pulled it. Use `git revert` to create a new commit that undoes the changes, rather than trying to erase the original commit.

**What to do:**
```bash
git revert HEAD              # Undo last commit safely
git revert HEAD~3..HEAD      # Undo last 3 commits
git revert --no-commit <sha> # Revert without auto-committing
```

---

## R009-R011: Active operations in progress
**Priority: 98-96**

```
git merge --continue OR git merge --abort
git rebase --continue OR git rebase --abort
git cherry-pick --continue OR git cherry-pick --abort
```

**What it detects:**
- `.git/MERGE_HEAD` exists (merge in progress)
- `.git/rebase-merge/` exists (rebase in progress)
- `.git/CHERRY_PICK_HEAD` exists (cherry-pick in progress)

**Why it matters:**
Cause leaving guns out open isnt a great move any time of the day

**What to do:**
Complete or abort the operation before doing anything else.

---

## R001: Detached HEAD
**Priority: 95**

```
git checkout <branch>
```

**What it detects:**
- HEAD points to a commit rather than a branch
- Often happens after `git checkout <sha>`

**Why it matters:**
Commits made in detached HEAD state aren't on any branch and can be lost.

**What to do:**
```bash
git checkout main         # Discard detached state
git checkout -b new-branch  # Keep work, create branch
git branch temp HEAD      # Save position before switching
```

---

## R032: Diverged on protected branch - use merge
**Priority: 90**

```
git merge origin/<branch>
```

**What it detects:**
- Local commits + remote commits on protected branch
- Both ahead and behind remote

**Why it matters:**
Protected branches should preserve merge commits. Rebasing on main/master creates a false linear history.
~~ definitely not personal ~~

**What to do:**
```bash
git merge origin/main  # Preserve merge history
```

---

