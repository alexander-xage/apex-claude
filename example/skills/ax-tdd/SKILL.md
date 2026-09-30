---
name: ax-tdd
description: >
  Test-first discipline. Auto-triggers on "let's implement", "add feature", "fix bug",
  "write a test for", "build out", and similar pre-code-change phrases. Iron rule: a failing
  test exists before production code. Explicit invocation: /ax-tdd.
---

<trigger>
Fires before writing production code: "implement X", "add feature Y", "fix bug Z",
"build out", "let's add". Skip only for pure docs/config changes — and then say
`skipped TDD because: <reason>`.
</trigger>

## Rules
1. Write the failing test first. Run it. Confirm it fails for the *right* reason (asserts the intended behavior, not a typo).
2. Write the minimum production code to pass. Run the test. Confirm green.
3. Refactor only with the test green.
4. The test encodes WHY — the intended behavior — not a mirror of the implementation. A test that passes when the logic is wrong is a liability.

## Boundary
This skill owns the *discipline*. Actual multi-file implementation is the job of the
`ax-builder` agent, which follows this same rule. Commit formatting is `ax-commit`.
