package explain

// Explanation holds human-readable context for a single rule.
type Explanation struct {
	ID          string
	Title       string
	Why         string
	Alternative string
}

// Lookup returns the explanation for a rule ID, if one exists.
func Lookup(id string) (Explanation, bool) {
	ex, ok := explanations[id]
	return ex, ok
}

// AllIDs returns the IDs of all rules that have explanations.
func AllIDs() []string {
	ids := make([]string, 0, len(explanations))
	for id := range explanations {
		ids = append(ids, id)
	}
	return ids
}

var explanations = map[string]Explanation{
	"R001": {
		ID:    "R001",
		Title: "Detached HEAD detected",
		Why: `You are not on a branch. Commits made here are not referenced by any branch
and will be garbage-collected by Git after 30 days unless you create a branch.
This is almost never intentional during normal development.`,
		Alternative: `git checkout <branch>      # return to a branch
git checkout -b <newbranch>  # or create one from the current HEAD`,
	},
	"R002": {
		ID:    "R002",
		Title: "Modified files not staged",
		Why: `Changes exist in the working tree but have not been staged. They will not be
included in the next commit. This is frequently unintentional.`,
		Alternative: `git add <files>         # stage specific files
git add -p              # stage changes interactively`,
	},
	"R003": {
		ID:    "R003",
		Title: "Staged files waiting for commit",
		Why:   `Files have been staged but no commit has been created yet. The index is ahead of the last commit.`,
		Alternative: `git commit              # create the commit
git commit -m "..."     # with an inline message`,
	},
	"R004": {
		ID:    "R004",
		Title: "Local commits ready to push",
		Why:   `You have commits that exist locally but not on the remote. They are invisible to your team until pushed.`,
		Alternative: `git push
git push -u origin <branch>   # if no upstream is set`,
	},
	"R005": {
		ID:    "R005",
		Title: "Behind remote and clean",
		Why:   `The remote has commits your local branch does not. Your working tree is clean, so a pull is safe now.`,
		Alternative: `git pull
git pull --rebase   # if you prefer a linear history`,
	},
	"R006": {
		ID:    "R006",
		Title: "Branch has diverged",
		Why: `Both you and the remote have commits the other does not have. You must reconcile the two histories
before you can push. Every minute you wait makes this harder.`,
		Alternative: `git rebase origin/<branch>   # rebase your work on top of theirs
git merge origin/<branch>    # or merge (preserves merge commits)`,
	},
	"R007": {
		ID:    "R007",
		Title: "Untracked files present",
		Why:   `Files exist in the working directory that Git does not track. They will not be committed or backed up unless explicitly added or ignored.`,
		Alternative: `git add <files>       # track them
echo "<file>" >> .gitignore  # or ignore them`,
	},
	"R008": {
		ID:    "R008",
		Title: "Stash exists",
		Why:   `You have work saved in the stash. Stashes are easy to forget, and the stash is not backed up to the remote.`,
		Alternative: `git stash pop         # restore the most recent stash
git stash list        # see all stashes`,
	},
	"R009": {
		ID:    "R009",
		Title: "Merge in progress",
		Why: `A merge operation was started and has not been completed or aborted. Most git operations
are blocked until this is resolved. Unresolved merges left for days become context puzzles.`,
		Alternative: `git merge --continue   # after resolving conflicts
git merge --abort      # to abandon the merge entirely`,
	},
	"R010": {
		ID:    "R010",
		Title: "Rebase in progress",
		Why: `A rebase is in progress. Git is replaying your commits one at a time; a conflict occurred.
Resolve the conflict, stage the fix, then continue.`,
		Alternative: `git rebase --continue   # after resolving and staging conflicts
git rebase --abort      # to restore the branch to its pre-rebase state`,
	},
	"R011": {
		ID:    "R011",
		Title: "Cherry-pick in progress",
		Why:   `A cherry-pick hit a conflict and stopped. Resolve the conflict, stage the result, and continue.`,
		Alternative: `git cherry-pick --continue
git cherry-pick --abort`,
	},
	"R020": {
		ID:    "R020",
		Title: "Local commits can be soft-reset",
		Why:   `You have a small number of unpushed commits. A soft reset unstages the commits but preserves all changes, letting you recommit with a cleaner message or structure.`,
		Alternative: `git reset --soft HEAD~N    # N = number of commits to undo`,
	},
	"R021": {
		ID:    "R021",
		Title: "Last commit was pushed — use revert instead of reset",
		Why: `The tip of the branch has been pushed to the remote. Hard-resetting it would require a force-push,
which rewrites history that others may have already pulled. Revert creates an undo commit instead,
preserving history.`,
		Alternative: `git revert HEAD            # creates a new commit that undoes the last one
git revert HEAD~3..HEAD    # revert a range`,
	},
	"R022": {
		ID:    "R022",
		Title: "Too many local commits — use interactive rebase",
		Why:   `You have many unpushed commits, suggesting an opportunity to squash, reorder, or clean up before sharing.`,
		Alternative: `git rebase -i HEAD~N       # N = number of commits to review`,
	},
	"R030": {
		ID:    "R030",
		Title: "Can fast-forward",
		Why:   `The remote has new commits and your branch has none, so a pull can simply advance the pointer without a merge commit.`,
		Alternative: `git pull --ff-only`,
	},
	"R031": {
		ID:    "R031",
		Title: "Feature branch diverged — rebase recommended",
		Why: `Your feature branch has both local commits and upstream commits it doesn't have. Rebasing keeps
history linear and makes the eventual merge review easier.`,
		Alternative: `git fetch && git rebase origin/<base-branch>`,
	},
	"R032": {
		ID:    "R032",
		Title: "Diverged on protected branch — merge instead of rebase",
		Why: `Protected branches (main, master, etc.) typically preserve full merge history for audit purposes.
Rebasing here would rewrite commits that other developers or CI systems may reference.`,
		Alternative: `git merge origin/<branch>`,
	},
	"R033": {
		ID:    "R033",
		Title: "Existing merge commits detected",
		Why:   `The branch already has merge commits. Switching to rebase now would require rewriting history. Stick with merge to stay consistent.`,
		Alternative: `git merge origin/<branch>`,
	},
	"R034": {
		ID:    "R034",
		Title: "No upstream configured",
		Why:   `The current branch has no remote tracking reference. Push and pull have no default destination, so you must specify the remote each time.`,
		Alternative: `git branch --set-upstream-to=origin/<branch>
git push -u origin <branch>   # sets upstream while pushing`,
	},
	"R035": {
		ID:    "R035",
		Title: "Merged branches ready for cleanup",
		Why:   `Branches that have been fully merged to the current branch serve no purpose. Accumulated stale branches clutter the repository and confuse newcomers.`,
		Alternative: `git branch -d <branch>    # safe: refuses to delete unmerged branches`,
	},
	"R036": {
		ID:    "R036",
		Title: "Gone remote branches",
		Why:   `The remote tracking branch no longer exists on the remote. The local branch is an orphan. Clean it up or it will accumulate.`,
		Alternative: `git branch -d <branch>
git remote prune origin   # clean up all gone remote refs at once`,
	},
	"R037": {
		ID:    "R037",
		Title: "Force-push to shared branch",
		Why: `Force-pushing rewrites the remote branch history. Anyone who has already pulled commits that
you are about to overwrite will have a diverged local branch with no clean resolution path.
This is one of the most disruptive things you can do to a team.`,
		Alternative: `git revert HEAD            # create an undo commit instead
git push --force-with-lease  # at minimum use this instead of --force`,
	},
	"R038": {
		ID:    "R038",
		Title: "Rewritten published tags",
		Why: `Tags are meant to be immutable references to specific commits. Rewriting a published tag
means anyone who fetched the old tag now has a different object under the same name.
Package managers and release pipelines often cache tags.`,
		Alternative: `Create a new tag with a new version name instead of rewriting the existing one.`,
	},
	"R039": {
		ID:    "R039",
		Title: "Reset on protected branch",
		Why: `Resetting a protected branch (main, master, etc.) moves the branch pointer backwards or sideways.
If the commits being abandoned have been pushed, recovery requires a force-push
that will break every team member who has pulled those commits.`,
		Alternative: `git revert HEAD            # always prefer revert for public commits`,
	},
	"R040": {
		ID:    "R040",
		Title: "Submodule pointer rewrite without update",
		Why:   `A submodule directory was changed in the index, but git submodule update has not been run. Builds will check out the wrong submodule commit.`,
		Alternative: `git submodule update --remote --recursive`,
	},
	"R041": {
		ID:    "R041",
		Title: "Accidental history rewrite after push",
		Why: `A rebase or filter-branch was run after commits were pushed. The local and remote histories
now diverge. Everyone who pulled the original commits has a locally inconsistent view.`,
		Alternative: `If the rewrite was accidental: git reflog to find the pre-rebase hash, then git reset --hard <hash>
to restore. Then push normally. Do not force-push unless you have coordinated with the team.`,
	},
	"R042": {
		ID:    "R042",
		Title: "Conflicted files staged",
		Why:   `Files containing conflict markers (<<<<<<<) have been staged. Committing them will put literal conflict markers into the repository history.`,
		Alternative: `Resolve all conflicts first, then git add the resolved files, then commit.`,
	},
	"R043": {
		ID:    "R043",
		Title: "Binary files without LFS",
		Why:   `Large binary files committed directly bloat the repository permanently. Unlike text, binary diffs cannot be compressed efficiently by Git, and the size accumulates in every clone forever.`,
		Alternative: `git lfs install
git lfs track "*.psd"       # or whatever pattern
git add .gitattributes`,
	},
	"R044": {
		ID:    "R044",
		Title: "Line ending normalization conflict",
		Why:   `Mixed line endings (CRLF vs LF) cause spurious diffs and make blame history noisy. Usually caused by editors on different platforms.`,
		Alternative: `git config core.autocrlf input     # recommended on macOS/Linux
git config core.autocrlf true      # recommended on Windows
echo "* text=auto" >> .gitattributes`,
	},
	"R045": {
		ID:    "R045",
		Title: "Submodule in detached HEAD",
		Why:   `A submodule is checked out at a specific commit rather than a branch. Changes made inside will not be tracked by any branch and may be lost when the submodule is updated.`,
		Alternative: `cd <submodule-dir> && git checkout <branch>`,
	},
	"R046": {
		ID:    "R046",
		Title: "Shallow clone doing history operations",
		Why: `Shallow clones (--depth N) have truncated history. git log, git bisect, git merge-base, and
similar operations will silently return wrong or incomplete results.`,
		Alternative: `git fetch --unshallow    # convert to a full clone`,
	},
	"R047": {
		ID:    "R047",
		Title: "Working directly on a protected branch",
		Why: `Protected branches (main, master) are shared history. Committing directly bypasses code review,
makes rollback harder, and creates a commit message style the team cannot control.`,
		Alternative: `git checkout -b feature/<descriptive-name>`,
	},
	"R048": {
		ID:    "R048",
		Title: "Long-lived feature branch",
		Why:   `Branches that diverge from main for weeks accumulate merge debt. Every day the gap grows, conflicts become more likely and more expensive to resolve.`,
		Alternative: `git fetch && git rebase origin/main   # keep the branch current
# or merge main into the feature branch periodically`,
	},
	"R049": {
		ID:    "R049",
		Title: "Squash recommended before merge",
		Why:   `Many small commits (fix, oops, wip, temp) clutter the project history and make git bisect less useful. Squashing produces a single, well-described commit per feature.`,
		Alternative: `git rebase -i HEAD~N    # squash interactively
git merge --squash <branch>   # squash on merge`,
	},
	"R050": {
		ID:    "R050",
		Title: "WIP commit on shared branch",
		Why:   `A commit with a WIP, temp, or debug message was pushed to a shared branch. These are drafts, not history. They confuse readers and make git bisect unreliable.`,
		Alternative: `git commit --amend -m "<proper message>"   # before pushing
git revert HEAD && git stash           # or revert and continue later`,
	},
	"R051": {
		ID:    "R051",
		Title: "Rebase recommended instead of merge",
		Why:   `The latest commit has multiple parents, indicating a merge. On feature branches, merge commits make history harder to read and git bisect less precise. Rebase produces a linear history.`,
		Alternative: `git rebase origin/main`,
	},
	"R052": {
		ID:    "R052",
		Title: "Commit message quality warning",
		Why:   `The last commit message is too short or generic (e.g. "fix", "update", "."). Git logs are used for debugging production incidents at 2am. Poor messages help no one.`,
		Alternative: `git commit --amend -m "<verb>: <what and why in one sentence>"`,
	},
	"R053": {
		ID:    "R053",
		Title: "Amend last commit suggested",
		Why:   `The last commit looks like it should have been part of the previous one, or the message could be improved. This is only safe before the commit is pushed.`,
		Alternative: `git commit --amend`,
	},
	"R054": {
		ID:    "R054",
		Title: "Unpushed local tags",
		Why:   `Tags exist locally that have not been pushed. Releases and version references that exist only on your machine are not useful to anyone else.`,
		Alternative: `git push --tags`,
	},
	"R055": {
		ID:    "R055",
		Title: "Stash stack growing",
		Why:   `Multiple stash entries are accumulating. Stashes are easy to forget and not backed up remotely. Work in the stash is work in limbo.`,
		Alternative: `git stash list        # review what's there
git stash pop         # apply and remove the most recent
git stash drop stash@{N}  # discard a specific entry`,
	},
	"R056": {
		ID:    "R056",
		Title: "Repo size growing fast",
		Why:   `The repository size is increasing unusually quickly. This often indicates accidentally committed large files, build artifacts, or binaries.`,
		Alternative: `git gc --aggressive
git lfs migrate import --include="*.bin"   # move large files to LFS`,
	},
	"R057": {
		ID:    "R057",
		Title: "Inactive branches detected",
		Why:   `Branches with no recent commits may be abandoned work. They accumulate over time and make branch lists confusing.`,
		Alternative: `git branch -d <branch>          # delete merged branches
git push origin --delete <branch>  # remove from remote too`,
	},
	"R058": {
		ID:    "R058",
		Title: "Detached HEAD but clean",
		Why:   `You are in detached HEAD state, but the working tree is clean so nothing is at immediate risk. This is normal after checking out a tag or specific commit.`,
		Alternative: `git checkout <branch>           # return to a branch
git checkout -b <new-branch>    # or create one from here`,
	},
}
