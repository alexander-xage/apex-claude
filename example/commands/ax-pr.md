---
description: Commit current changes, push, then open a PR via gh. Delegates message + body tone to the ax-commit and ax-review skills.
---

<flow>
1. **Pre-flight.** `git status --short`. Branch guard: not on base. If clean working tree but commits exist ahead of base, skip to the review gate.
   **Review gate.** If the branch's commits haven't been through `/ax-ship` review, dispatch `ax-reviewer` on `git diff <base>...HEAD`. Any 🟥 or low CONFIDENCE → stop and surface before publishing anything.
2. **Stage + message.** Stage by path. Invoke `ax-commit`; commit.
3. **Push.** `git push -u origin <branch>`.
4. **PR body.** Summarize the branch diff vs base (`git diff <base>...HEAD --stat`). Tone per `ax-review` (signal-first, no fluff). Structure: what + why + how-verified.
5. **Open.** `gh pr create --title <subject> --body <body> --base <base>`. Report the PR URL.
</flow>

<safety>
- Opening a PR publishes the diff and body externally. Confirm if the change is sensitive.
- End the PR body with the Generated-with-Claude-Code footer.
</safety>
