# Git-Next Rules Reference

Git-next analyzes your repository state and provides prioritized advice based on a comprehensive rule system.

## Rule Organization

Rules are organized by **danger level** (priority) into five categories:

### [Dangerous Operations](./dangerous.md) (Priority 100-90)
**"Put the keyboard down"**

Critical warnings about operations that can break things. These rules prevent destructive actions on shared code.

**Example Rules:**
- R037: Force-push to shared branch
- R038: Rewritten published tags
- R039: Reset on protected branch

[View all dangerous rules →](./dangerous.md)

---

### [Repo Integrity](./integrity.md) (Priority 89-60)
**Issues that need immediate attention**

Problems with repository health, data integrity, and configuration that can cause builds to fail or code to be lost.

**Example Rules:**
- R042: Conflicted files staged
- R043: Binary files without LFS
- R044: Line ending conflicts

[View all integrity rules →](./integrity.md)

---

### [Workflow Hygiene](./workflow.md) (Priority 59-30)
**Best practices and normal workflow**

Standard git workflow suggestions - merging, rebasing, committing, pushing.

**Example Rules:**
- R047: Working on main instead of feature branch
- R048: Long-lived feature branch
- R049: Squash recommended

[View all workflow rules →](./workflow.md)

---

### [Mild Suggestions](./suggestions.md) (Priority 29-10)
**Helpful hints and optimizations**

Nice-to-have improvements and quality-of-life suggestions.

**Example Rules:**
- R052: Commit message quality warning
- R053: Amend last commit suggested
- R054: Unpushed local tags

[View all suggestion rules →](./suggestions.md)

---

### [Informational](./informational.md) (Priority <10)
**Just FYI**

Informational notices about repository health - nothing wrong, just awareness.

**Example Rules:**
- R056: Repo size growing fast
- R057: Inactive branches detected
- R058: Detached HEAD but clean

[View all informational rules →](./informational.md)

---

## Rule Suppression

Higher priority rules automatically suppress lower priority rules when they apply to the same git operation. For example:

- "Merge in progress" (R009) suppresses "Pull updates" (R005)
- Dangerous operations suppress almost everything else
- This prevents contradictory or overwhelming advice

## Configuration

You can customize rule behavior in `.git-next.yaml`:

```yaml
rules:
  disabled:
    - R007  # Disable "untracked files" warnings

  parameters:
    R020:
      max_commits: 5  # Increase soft reset threshold
```

