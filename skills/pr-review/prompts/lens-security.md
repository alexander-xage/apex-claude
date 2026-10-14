# Lens: security

Review the change for what an attacker could do with it. Bloat and general correctness belong to other lenses; report
a bug here only when it is exploitable or exposes something.

Start from where untrusted data enters the code the change touches: request fields, headers, query strings, file
contents, uploaded files, environment, CLI arguments, messages from other services, data read back from storage.
Follow it to where it is used, and report what the change lets through.

- **Injection.** Untrusted input reaching SQL, shell commands, templates, file paths, regexes, deserializers, LDAP,
  XPath or eval without the parameterization or escaping that sink needs. Path traversal and symlink escapes.
- **Authentication and authorization.** A new endpoint, handler or action without the checks its neighbors have. An
  object fetched by id without checking the caller owns it. Privilege granted by input the caller controls.
- **Secrets.** Keys, tokens, passwords or connection strings committed in code, config or tests; secrets written to
  logs, error messages, URLs or responses.
- **Crypto.** Home-made crypto, weak or misused primitives, predictable randomness for tokens, missing or
  non-constant-time signature and MAC checks, TLS verification turned off.
- **Data exposure.** Responses, logs or errors that leak personal data, internal paths, stack traces or other users'
  records. Permissive CORS, cookies without `Secure`/`HttpOnly`/`SameSite`, missing CSRF protection on a state change.
- **Resource abuse.** Input that drives unbounded memory, loops, recursion, file size or outbound requests; SSRF from
  a URL the caller controls; missing rate limits on a new expensive or sensitive action.
- **Dependencies and supply chain.** A new dependency or version that is unmaintained, typo-squatted or known
  vulnerable; a script, action or image pulled by a mutable tag; install hooks that run remote code.
- **Concurrency.** Check-then-use races on files, balances, permissions or tokens (TOCTOU).

For every finding, say who the attacker is, what input they send and what they get. If you cannot describe that
path, leave the finding out.

Most findings here are `ISSUE`: a reachable vulnerability must be fixed before merging. Use `ASK` for hardening with
no concrete exploit, `LOW` for defense in depth, and `NIT` almost never.
