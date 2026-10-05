Adapt the plan files of the repository "{repository_name}" to the Kandev plan file format.

Repository: {repository_path}
Current branch: {current_branch}
Plan directories (relative to the repository root): {plan_directories}

You are working in the owner's main checkout, not in a worktree. Other changes the owner has made may be present. Touch only the plan files described below.

A plan file is a Markdown file in a plan directory. It is "unadapted" when its frontmatter has no `board` key. Adapting a file means adding the frontmatter block described in rule 2, so the plan appears on the plans board with the right status.

Board statuses (the closed set for `board`): {board_statuses}

Rules:

1. Classify every unadapted plan file into exactly one board status. Decide from repository evidence, not from the file alone: explicit status sections or banners in the file, merged branches and commits that mention the plan, the presence of the code the plan creates or changes, and documents that supersede the plan. Treat checkbox counts as unreliable evidence: completed plans often have no ticked boxes, and abandoned plans often have some. Look at the git history and the working tree before you decide.

2. Add a frontmatter block at the very top of each unadapted file with the keys `board`, `priority`, and `order`. Add `depends_on` only when the plan states an explicit dependency on a plan that is not finished; it is a list of plan file names in the same directory. Do not change any other byte of the file: the body, the line endings, the trailing newline, and any text that was already there stay exactly as they are. If the file already starts with a frontmatter block that lacks `board`, add the keys inside that block and keep its other keys unchanged. `priority` is one of `critical`, `high`, `medium`, `low`. `order` is a decimal number that sorts the plans within one status.

3. Leave files that already declare `board` unchanged, even when you disagree with their status.

4. Create exactly one commit on the current branch that contains only the changed plan files. Use a path-limited commit: `git commit -- <directories>` with the plan directories above (or the explicit changed paths), so that changes the owner already staged stay staged and untouched. Verify the result with `git show --name-only --format= HEAD`: the list must contain only the plan files you changed. If it contains any other path, undo the commit with `git reset --soft HEAD~1`, fix the problem, and commit again. Do not push, and do not create or switch branches.

5. Finish with a Markdown table of every file you adapted, with the columns file, status, and one line of evidence. After the table, list the files whose status is uncertain, each with the reason, so the owner can review them first.
