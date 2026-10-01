# Lens: general

Review the change for correctness and for how it fits the code structure that is already there.

- **Correctness.** Wrong logic, missed edge cases, off-by-one, nil or empty input, error paths that lose or hide the
  error, races, resource leaks.
- **Structure.** The change follows the patterns, layering and naming of the code around it. It reuses the helper
  that already exists instead of writing a second one. It lands in the file and module where a reader would look.
- **Contracts.** Callers of anything whose signature, return value or behavior changed still work. Public API,
  schema, config and CLI changes are deliberate and complete.
- **Tests.** New behavior and fixed bugs have a test that would fail without the change. Changed behavior did not
  leave a test asserting the old one.
- **Leftovers.** Dead code, debug output, commented-out code, stale comments and docs the change made untrue.
