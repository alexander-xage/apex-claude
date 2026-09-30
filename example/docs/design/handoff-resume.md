# Design — handoff / resume

> **Status:** implemented, then partly superseded on 2026-09-12: the binary no longer renders or writes the handoff document — `apex handoff scan` is a read-only fact report and the model writes `handoff.md`. This is the decision record as written; for current behavior read `docs/spec/handoff-resume.md` and its change log.

Session continuity for Apex: capture enough state at a stopping point that a fresh-context
session can pick the work back up. Binary owns the deterministic scan + staleness verdict;
skills and commands compose the narrative on top. Respects the determinism boundary.

## Problem

A session ends — context limit, time, or a clean project boundary — and the next session starts
blind. Git history shows *what* changed but not *why we stopped here* or *what comes next*. The gap
is intent: the locked decisions, the half-finished thread, the "resume by doing X". That is model
judgment and must be written by the model; everything mechanically derivable (branch, HEAD, dirty
files, open followups, due reminders, health, signals staleness, active brief) is the binary's job.

## Decision — Strategy B (binary-backed)

`apex handoff` Go subcommand owns the deterministic layer; `/ax-handoff`, `/ax-resume`, and the
`ax-handoff` skill compose the narrative. Validated consistent with every existing backbone pattern:

| Concern | Existing precedent | Cite |
|---|---|---|
| Cmd dispatch (`register` + `switch args[0]`, exit-code return) | followups | `cmd/apex/cmd_followups.go:14,23`; `cmd/apex/registry.go` |
| State under `.claude/project/` via `proj.Root`/`StateDir` | proj | `internal/proj/proj.go:10,19` |
| Frontmatter Parse/Render, ordered keys, zero-dep | fm | `internal/fm/fm.go:9,32` |
| `%03d` next id over a state dir | followups, reminder | `internal/followups/followups.go:33` |
| Archive-on-close (move out of active set + ledger line) | followups CLOSED | `internal/followups/followups.go:96` |
| RFC3339 `created` stamp | reminder | `internal/reminder/reminder.go` |
| Deterministic-scan → model-context handoff | session-start hook | `internal/hooks/hooks.go:20` |
| Capture sources readable as library calls | health/signals/reminder | `internal/health/health.go:24`, `internal/signals/signals.go:141`, `internal/reminder/reminder.go:103` |

Artifact registration: doctor + validate scan the command/skill dirs by **glob, not per-file**
(`internal/doctor/doctor.go`, `internal/validate/validate.go`), so new `commands/*.md` and
`skills/ax-handoff/SKILL.md` need **no doctor/validate code edit** — they're picked up automatically.
Spec lint requires a `# ` title and a `## Change log` section (`internal/validate/validate.go`).

**First git shell-out in the backbone.** `grep exec.Command|git rev-parse` over `internal/` is empty
today — `internal/handoff` is the first `os/exec` git caller. Keep it isolated behind small helpers
(`gitHead`, `gitBranch`, `gitDirty`) so the rest of the package stays testable without a repo.

## Pressure-test 1 — staleness model

**Question:** is "HEAD moved past recorded `head` sha" the right staleness signal?

**Conclusion — keep the sha anchor; rule = `live HEAD != recorded head → exit 2`.** "Moved past"
means "no longer matches", which collapses three drift cases into one verdict: new commits, amend/
rebase rewriting the sha, and branch switches. Rejected alternatives and why:

- **Ancestry math** (`git merge-base --is-ancestor recorded HEAD`) — brittle after rebase/amend (the
  recorded sha no longer exists) and after a branch switch. Adds failure modes for no gain.
- **File-relevance diffing** (stale only if commits touched files named in the handoff) — that is
  model judgment, not a deterministic exit code. Belongs in the resume reconciliation step, not the
  gate.
- **Dirty-tree at capture** — recorded in frontmatter as a note, never folded into the exit code.

**Bias toward stale is correct.** Over-reporting stale costs only a rescan — which the fresh-path
(`exit 0`) already does anyway during reconciliation. Under-reporting builds a resume on a lie. So
when in doubt, return `2`. Branch drift (HEAD matches but branch name differs, or branch gone) is
surfaced as a human-readable note alongside the code, not as a separate exit code — keeps the routing
contract at three values.

Exit-code contract:

| Code | Meaning | Resume route |
|---|---|---|
| `0` | active handoff present, `head` == live HEAD | read doc → reconcile vs read-only live state (`git status`, followups, health) → confirm → `handoff archive` |
| `2` | active handoff present, `head` != live HEAD | warn drift → same read-only reconcile as 0 → confirm → archive |
| `1` | no active handoff | re-prompt: rescan-reconstruct (`handoff scan`) \| tell me where \| start fresh |

Note: `apex handoff scan` *writes* the skeleton (like `apex signals scan`), so it is used only for
capture (`/ax-handoff`) and route-1 reconstruct — never during a route-0/2 consume, where it would
overwrite the doc being read. Route-0/2 reconciliation is read-only.

## Pressure-test 2 — auto-fire phrase scoping

**Question:** are the auto-fire phrases tight enough to avoid false fires yet catch real intent?

**Conclusion — bare "resume" exclusion CONFIRMED; locked phrases are well-scoped.** Each locked
phrase carries an explicit object ("the session", "for later", "the work") so it can't fire on
incidental usage ("resume the build", "I'll resume after lunch"). Recommended additions, all
object-bearing: handoff — "hand off the session", "write a handoff"; resume — "pick up where we left
off", "resume our session". No removals.

**Guardrail (recommended):** skill auto-fire confirms before doing anything destructive (overwriting
an un-consumed active handoff, or archiving on resume). Explicit commands (`/ax-handoff`,
`/ax-resume`) skip the confirm — the user already stated intent by typing the verb.

## Resolved at the approval gate

1. **`handoff write` over an un-consumed active doc → archive old first.** Never a silent overwrite;
   the prior un-consumed doc moves to `handoffs/<id>.md` before the new one is written.
2. **`head` anchor → short sha** (matches `git log` usage across the repo).
3. **Phrase set → adopt the recommended object-bearing additions.**
4. **Auto-fire guardrail → confirm-before-destructive on skill auto-fire only** (explicit commands
   skip confirm).

## Evidence trail

All citations above verified by direct read this session. Notable negative result: no pre-existing
git shell-out in `internal/` — the staleness scan introduces it.
