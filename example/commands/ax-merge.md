---
description: Land the current branch on base. Default merge commit (--no-ff); --squash collapses the branch into one commit. Reviews an unreviewed branch first, re-runs tests on the landed tip, prefers gh when a PR is open, offers to remove the worktree.
argument-hint: [--squash]
---

<flow>
1. **Pre-flight.** Working tree clean (commit or stash first). Identify base (main/master).
2. **Review gate.** If the branch's commits haven't been through `/ax-ship` review, dispatch `ax-reviewer` on `git diff <base>...HEAD`. Any 🟥 or low CONFIDENCE → stop and surface; don't land.
3. **Open PR?** If `gh pr view` finds one, land through GitHub: `gh pr merge --merge` (or `--squash`). Else local.
4. **Local land.**
   - default: `git checkout <base>` → `git merge --no-ff <branch>`.
   - `--squash`: confirm the user wants the branch history collapsed, then `git checkout <base>` → `git merge --squash <branch>` → `ax-commit` synthesizes one message for the whole branch → commit.
5. **Verify the landed tip.** Re-run the project test suite. On failure stop and report — never push a broken base.
6. **Worktree.** If the branch lived in `.worktrees/<branch>/`, offer to remove it.
7. **Report.** Resulting commit + test result. Push only if the user asks.
</flow>

<safety>
Landing on base is hard to reverse for collaborators. Roll back with `git revert`, never force-push base.
</safety>
