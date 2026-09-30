---
description: Work the follow-up ledger and reminders. Bare lists open follow-ups and due reminders; review walks stale entries (extend/close/promote/skip); remind files a reminder that surfaces at session start.
argument-hint: [review | remind <when> <what> | due <id>]
---

<flow>
1. **List** (bare). `apex followups list` + `apex reminder list` (and `apex reminder due` for fired ones). Present as one indexed list.
2. **Review** (`review`). Walk stale follow-ups one at a time — extend (re-date via `apex reminder`), close (`apex followups close <id> <reason>`), promote (a plan → spec via `/ax-plan`), or skip. Confirm each disposition.
3. **Remind** (`remind <when> <what>`). Resolve `<when>` ("tomorrow", "in 2h", "after the PR") to an absolute RFC3339 time; `apex reminder add "<what>" <due>`. It surfaces in the next session start at or after that time. For a nudge that must fire mid-session or with no session open, point the user at the built-in `/schedule`.
4. **Due** (`due <id>`). Surface that reminder and wait for a response.
</flow>

<notes>
- Two follow-up kinds: `finding` (loose ends, subject to staleness) and `plan` (deferred work, exempt). Plans render first.
- File new follow-ups with flags: `apex followups add "<title>" --kind finding|plan --severity <sev> --origin <origin>`.
</notes>
