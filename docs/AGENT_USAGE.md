# git-next for Coding Agents

`git-next` is designed to be consumed by coding agents, CI bots, and shell tools — not just humans.

## Output schema

```bash
git-next --agent
```

Output conforms to [`docs/schema/agent-v1.json`](schema/agent-v1.json). The `schema_version` field is `"1"` and will increment on breaking changes.

```json
{
  "schema_version": "1",
  "status": "unsafe",
  "risk_level": "high",
  "summary": "Branch has diverged - need to sync",
  "recommended_next_command": "git rebase origin/main OR git merge origin/main",
  "allowed_commands": [
    "git add", "git commit", "git diff", "git fetch",
    "git log", "git remote -v", "git show", "git status"
  ],
  "blocked_commands": [],
  "rules_triggered": [
    {
      "id": "R006",
      "severity": "high",
      "description": "Branch has diverged - need to sync",
      "destructive": false
    }
  ]
}
```

**Status values:**

| `status` | Meaning |
|---|---|
| `safe` | No active rules. Proceed normally. |
| `caution` | Low/informational rules active. Awareness only. |
| `unsafe` | High-severity rules active. Do not push or merge. |
| `critical` | Critical rules active. Stop all operations. |

---

## Checking a command before running it

```bash
git-next guard -- git push --force
# exit 0 = allowed, exit 1 = blocked
```

```bash
git-next guard --json -- git reset --hard HEAD~3
```

```json
{
  "command": "git reset --hard HEAD~3",
  "allowed": false,
  "risk_level": "danger",
  "reason": "commits have already been pushed; hard-resetting rewrites shared history",
  "alternative": "git revert HEAD   # safer: creates an undo commit rather than erasing history",
  "active_rules": []
}
```

---

## Claude Code integration

Add a pre-tool-use hook to `~/.claude/settings.json` that runs `git-next guard` before any shell command Claude Code executes:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "bash -c 'if echo \"$CLAUDE_TOOL_INPUT\" | grep -q \"^git \"; then git-next guard -- $CLAUDE_TOOL_INPUT 2>&1; fi'"
          }
        ]
      }
    ]
  }
}
```

Or use the simpler approach: run `git-next --agent` at the start of any agentic session and pass the JSON to the model as context about the repository state.

**System prompt snippet:**

```
Before running any git command, check the current repository state:
  git-next --agent

If status is "critical" or "unsafe", do not run any git operations that
appear in blocked_commands. Use the recommended_next_command if available.
Always prefer git revert over git reset when the last commit is pushed.
```

---

## GitHub Actions

### Basic CI gate

```yaml
- name: Git safety check
  run: git-next ci --fail-on high --format github
```

This exits 0 when the repository is clean or has only low/informational rules, and emits `::error::` annotations for high and critical rules.

### Severity-tiered jobs

```yaml
jobs:
  safety-gate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0   # shallow clones trigger R046

      - name: Install git-next
        run: |
          curl -sfL https://raw.githubusercontent.com/VectorSophie/git-next/main/scripts/install.sh | sh

      - name: Block critical issues
        run: git-next ci --fail-on critical --format github

      - name: Warn on high severity
        run: git-next ci --fail-on high --format github
        continue-on-error: true
```

---

## Pre-push hook

```bash
git-next hook install            # installs to .git/hooks/pre-push
git-next hook install --hook pre-commit
```

The hook content:

```sh
#!/bin/sh
# Managed by git-next. Remove this line to stop git-next from managing this hook.
git-next ci --fail-on high
```

To remove:

```bash
git-next hook uninstall
```

---

## Shell alias (guard every git command)

Add to `~/.bashrc` or `~/.zshrc`:

```bash
git() {
    git-next guard -- git "$@" || return $?
    command git "$@"
}
```

**Caveats:** This adds a `CollectState` call (~50ms) to every git invocation. Disable for bulk operations with `command git <cmd>`.

---

## Reading agent output in scripts

```bash
STATUS=$(git-next --agent | jq -r .status)
if [ "$STATUS" = "critical" ] || [ "$STATUS" = "unsafe" ]; then
    echo "Repository is not safe to deploy from"
    exit 1
fi
```

```bash
# Get the recommended command
NEXT=$(git-next --agent | jq -r '.recommended_next_command // empty')
[ -n "$NEXT" ] && echo "Suggested: $NEXT"
```

---

## Design guarantees for agent consumers

- **Deterministic**: same repository state always produces the same output.
- **Exit codes**: `git-next --agent` exits 0 when status is `safe`, exits 1 otherwise.
- **No network calls**: all analysis is local git commands.
- **Schema stability**: `schema_version` will increment on breaking changes; fields will not be removed within a version.
- **Suppression is applied**: `rules_triggered` contains only active (non-suppressed) rules. You will never see both `merge` and `rebase` recommended simultaneously.
