Clean up the repository "{repository_name}": fold every branch and worktree into its default branch, then remove what is merged.

Repository: {repository_path}
Default branch: {default_branch}

In every command below, `<default>` stands for the default branch above. If the default branch is empty, resolve it with `git symbolic-ref refs/remotes/origin/HEAD` and use that branch name. If that also fails, stop and ask the owner which branch is the default; never guess it.

You are working in the owner's main checkout (the repository path above), not in a worktree. The owner has authorized the merges, the conflict resolution, and, after one confirmation, the push and the deletions described below. They have authorized nothing else.

Protected branches (attached to live Kandev tasks):
{protected_branches}

Protected worktrees (attached to live Kandev tasks):
{protected_worktrees}

Never commit to, merge, delete, or remove a protected branch or worktree, in any phase. Treat a branch checked out in a protected worktree as protected too. The value `none` means the list is empty.

## Phase 1: Inventory

Do not check out anything in this phase.

1. Run `git fetch --prune`.
2. List the local branches with `git branch --format='%(refname:short)'`, the remote branches that have no local counterpart with `git branch -r`, and the worktrees with `git worktree list`.
3. Drop every protected branch and worktree from the working lists. Everything below applies only to what remains.

## Phase 2: Snapshot

Before any checkout, commit the uncommitted changes of every remaining worktree, and of the main checkout if it is dirty, to its own current branch with the message `chore: wip snapshot before cleanup`. Include untracked files that are not ignored. If the main checkout's current branch is protected, do not snapshot or touch it: stop and report it to the owner.

Only after the snapshots, run `git checkout <default>` and then `git pull --ff-only`. If either fails, stop and tell the owner why; change nothing else.

## Phase 3: Merge

Merge into `<default>`, one branch at a time with `git merge --no-ff <branch>`, every remaining local branch and every remaining remote branch that has no local counterpart, when it has commits not in `<default>`. Merge a remote-only branch as `origin/<name>`; skip `origin/HEAD` and `origin/<default>`.

- Resolve conflicts yourself. After a conflicted merge, find the project's test command in AGENTS.md, CLAUDE.md, README, Makefile, or package scripts, and run it.
- If you are not confident in a conflict resolution, or the tests fail, run `git merge --abort` (or, once the merge commit exists, `git reset --hard <pre-merge commit>`), keep the branch, and note it with the reason for the report. Then continue with the next branch.
- A branch with no commits that are missing from `<default>` needs no merge; it is a candidate for removal in Phase 5.

## Phase 4: Confirmation

Post exactly one summary in this task conversation and end your turn waiting for the owner. The summary lists:

- the branches you merged;
- the branches you kept, each with the reason;
- the local branches and worktrees you intend to remove;
- the remote branches you intend to delete;
- the push target (`origin <default>`).

Do not push, delete, or remove anything before the owner's explicit confirmation arrives as a reply in this conversation. If the owner declines, stop: the local merges and snapshot commits stay on the local branches, and nothing is pushed or deleted.

## Phase 5: Apply

Only after the explicit confirmation:

1. Run `git push origin <default>`. If it is rejected, report the rejection and stop; delete nothing on the remote.
2. Before removing any worktree, run `git worktree list` again and skip any worktree that was not in the Phase 1 inventory. Remove the worktrees of merged, non-protected branches with `git worktree remove <path>`.
3. Delete merged local branches with `git branch -d <branch>`.
4. Delete remote branches only with `git push origin --delete <branch>`, and only for remote branches listed by `git branch -r --merged origin/<default>`, after the push succeeded. Skip any branch that is on the protected list or that is not in the Phase 3/Phase 4 list the owner confirmed.

## Phase 6: Forbidden operations

- Never use `--force` or `--force-with-lease`, on any command. If a push is rejected, report the rejection; do not retry with force.
- Never use `git branch -D`. If `git branch -d` refuses, the branch is not merged: keep it and report it.
- Never delete the default branch, locally or on the remote.
- Never rebase or amend commits that are already published.
- Never touch a protected branch or worktree.

## Phase 7: Report

Finish with a report covering: what was merged, what was kept and why, which branches were deleted locally, which were deleted on the remote, which worktrees removed, and what was pushed. Name every branch and path explicitly.
