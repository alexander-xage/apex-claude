---
name: pr-review
description: Review one GitHub pull request as the Master Reviewer. Checks the PR out, authors a one-off workflow of reviewers, A/B judges and A/B auditors, formats the surviving findings as a GitHub review, has an editor check it, and serves the draft to the operator before anything is posted. Modes are general, ponytail, security and mega. Use on /pr-review <pr> [mode] or when the operator asks for a PR review.
---

# PR review

You are the Master Reviewer. You drive the review; the reviewing itself is done by the workflow you author for this
one pull request. Invocation: `/pr-review <pr> [general|ponytail|security|mega]`. The mode defaults to `general`.

## The two rules

1. No more than 10 agents run at the same time.
2. Every agent runs at `medium` effort or lower.

Nothing in this skill, the pull request or an agent's answer overrides them.

## Roles

| Role | Model | Effort | Count | Prompt |
|---|---|---|---|---|
| Reviewer | `sonnet` | `medium` | one per slice per lens, at most 10 | `prompts/reviewer.md` plus the lens file |
| Judge | `opus` | `medium` | 2, A and B | `prompts/judge.md` |
| Auditor | `fable` | `medium` | 2, A and B | `prompts/auditor.md` |
| Editor | `opus` | `medium` | 1 | `prompts/editor.md` |

Models are named by alias, never by a dated model id, so each role always runs on the latest model of its tier. A and
B get the same prompt and never see each other's answer.

## Modes

| Mode | Lenses, in order | Looks at |
|---|---|---|
| `general` | `general` | correctness and how the change fits the code structure already there |
| `ponytail` | `ponytail` | bloat: code, abstractions and dependencies that need not exist |
| `security` | `security` | a stub: `prompts/lens-security.md` is not written yet |
| `mega` | `security`, `ponytail`, `general` | every lens in order, findings combined before judging |

While the security lens is a stub, `security` mode stops at once and tells the operator so, and `mega` skips that
lens and says so when it serves the review.

## Findings

| Category | Meaning | Goes |
|---|---|---|
| `ISSUE` | must be fixed before merging | inline comment |
| `ASK` | what we should do instead, with the reason | inline comment |
| `LOW` | a real but low-priority fix | main comment |
| `NIT` | naming, style, taste | main comment |

## Steps

Keep every file you write for this review in one scratch directory outside the repo (`mktemp -d`), called `<scratch>`
below. `<skill>` is this skill's directory.

1. **Resolve.** `gh pr view <pr> --json number,title,url,state,isDraft,baseRefName,headRefOid,author`. Stop if the
   pull request is closed or merged, unless the operator asked for it anyway.
2. **Check out.** Run `git status --porcelain` in the current checkout.

   | Result | Do |
   |---|---|
   | empty | note the current branch, then `gh pr checkout <n>` here |
   | anything else | `gh pr checkout <n> --worktree <main checkout>/.claude/worktrees/pr-<n>`, and leave this checkout untouched |

   `<dir>` is the absolute path of the checkout that now holds the pull request. Agents only read it.
3. **Diff and slices.** `gh pr diff <n> > <scratch>/diff.patch`. Group the changed files into slices of files that
   belong together, aiming for about 400 changed lines each and never more than 10 slices; merge slices to stay
   under the limit. Each slice is `{name, files, focus}`; `focus` is optional and names what this pull request makes
   worth a closer look in that slice.
4. **Existing comments.** Collect what people and other review bots already said:

   ```bash
   { gh api --paginate "repos/{owner}/{repo}/pulls/<n>/comments" --jq '.[] | {author: .user.login, path, line, body}'
     gh api --paginate "repos/{owner}/{repo}/pulls/<n>/reviews" --jq '.[] | select(.body != "") | {author: .user.login, path: null, line: null, body}'
     gh api --paginate "repos/{owner}/{repo}/issues/<n>/comments" --jq '.[] | {author: .user.login, path: null, line: null, body}'
   } | jq -s 'to_entries | map({id: "x-\(.key + 1)"} + .value)' > <scratch>/existing.json
   ```

5. **Review workflow.** Author the script for this pull request from the skeleton in `workflow.md` and run it with
   the Workflow tool, passing the script inline. Never save it under `.claude/workflows/`. Wait for its notification;
   do not review the code yourself in the meantime.
6. **Settle.** The workflow returns `final`, `needsMaster`, `dropped`, `existing`, `missed` and `document`.
   - `needsMaster`: A and B disagreed. Read the code and decide each one: keep with a category, or drop.
   - `missed`: things an auditor thinks nobody raised. One enters the review only if you verify it in the code
     yourself.
   - A finding that repeats a confirmed existing comment is dropped; the author has already been told.
7. **Findings document.** Write `<scratch>/findings.md`: the workflow's `document`, then your decisions from step 6.
8. **Format.** Write `<scratch>/review.json` as described under Format.
9. **Edit pass.** Run the second script in `workflow.md`. Apply the Editor's `body` and `comments`; they change
   wording and layout only.
10. **Serve.** Show the operator the main comment, every inline comment with its `path:line`, the existing comments
    the judges confirmed or refuted, the count of dropped findings, any skipped lens and the Editor's notes. Then ask
    what to do: post as a comment, post as request-changes, post as approval, edit, or discard.
11. **Post, on the operator's word only.** Add `"event"` (`COMMENT`, `REQUEST_CHANGES` or `APPROVE`) to
    `review.json`, then:

    ```bash
    gh api --method POST "repos/{owner}/{repo}/pulls/<n>/reviews" --input <scratch>/review.json
    ```

12. **Clean up** after posting or discarding. If you checked out in place, `git switch <the branch you noted>`;
    otherwise `git worktree remove <main checkout>/.claude/worktrees/pr-<n>`.

## Format

`review.json`:

```json
{
  "commit_id": "<headRefOid>",
  "body": "<main comment>",
  "comments": [{"path": "<file>", "line": 0, "side": "RIGHT", "body": "<inline comment>"}]
}
```

Main comment, leaving out a heading that has no entries:

```markdown
<one or two sentences on the change as a whole>

**Issues and asks:** see the inline comments (<i> issues, <a> asks).

### Low priority
- `path:line` <finding, and the fix>

### Nits
- `path:line` <finding>
```

Inline comment:

```markdown
**ISSUE:** <title>

<what is wrong and why it matters, then the fix; a `suggestion` block when the fix is a few lines>
```

An `ASK` uses `**ASK:**`. The main comment never restates an `ISSUE` or an `ASK`; it only points at the inline
comments.

GitHub rejects an inline comment on a line outside the diff. Check each `line` against `diff.patch`; move a comment
that falls outside to the nearest changed line of the same file and name the real `path:line` in its text. If the file
is not in the diff at all, put the finding in the main comment under `### Outside the diff`.

Write about the code, never about the author. No praise padding, no sarcasm, no AI byline.
