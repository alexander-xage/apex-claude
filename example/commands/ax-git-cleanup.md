---
description: Scan and clean stale git state — worktrees, local branches, optionally remote tracking refs. Dispatches ax-git-scout for the inventory, presents an indexed report, asks which to clean. No destructive ops without explicit confirmation.
---

<flow>
1. **Scan.** Dispatch `ax-git-scout` (read-only). Default scope: local + worktrees; include remote only if asked. Default staleness: 30 days.
2. **Report.** Present its indexed candidate table (ref · kind · reason · evidence).
3. **Confirm.** Ask which rows to clean by number. No deletion without explicit per-item confirmation.
4. **Act.** YOU run the mutating ops (`git branch -d`, `git worktree remove`, `git push --delete`) only for confirmed rows. Skip dirty trees unless forced.
5. **Report** what was removed and what was kept.
</flow>

<safety>
Branch/worktree deletion is hard to reverse. The scout never mutates — you do, only on explicit confirmation. Never delete a branch with unmerged work without flagging it first.
</safety>
