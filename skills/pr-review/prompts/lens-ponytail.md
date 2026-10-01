# Lens: ponytail

Review the change for bloat only: what could be deleted or made smaller with the same result. Correctness belongs to
another lens.

If the `ponytail:ponytail-review` skill is available, load it and apply it to your slice. Otherwise walk this ladder
for each thing the change adds and report where an earlier rung would have held:

1. It does not need to exist: a speculative option, a config for a value that never changes, scaffolding for later.
2. The codebase already has it: a helper, type or pattern a few files away.
3. The standard library does it.
4. A native platform feature covers it.
5. A dependency that is already installed does it; or a new dependency was added for what a few lines would do.
6. It could be one line.
7. It is more code than the problem needs: an interface with one implementation, a factory for one product, a layer
   that only forwards.

Name what to cut and what replaces it. Most findings here are `ASK` or `LOW`; use `ISSUE` only when the bloat itself
would be costly to remove after merging, such as a new public API or a new dependency.
