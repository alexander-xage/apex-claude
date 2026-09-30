#!/usr/bin/env bash
#
# install.sh — deploy Apex Claude as loose user-level artifacts.
#
# Apex installs its commands/agents/skills/output-style directly into
# ~/.claude/ (NOT as a Claude Code plugin). User-level artifacts are not
# namespaced, so commands appear as bare /ax-* instead of /apex-claude:ax-*.
# The tradeoff: no plugin enable/disable/update lifecycle — this script owns
# install, and scripts/uninstall.sh owns removal.
#
# What it does (all idempotent, safe to re-run):
#   1. Builds the apex backbone binary.
#   2. Removes any prior PLUGIN install of Apex (migration — avoids /ax-* and
#      /apex-claude:ax-* showing up as duplicates).
#   3. Copies artifacts into ~/.claude/{commands,agents,skills,output-styles}.
#   4. Installs the binary into ~/.claude/bin/apex.
#   5. Wires the SessionStart hook into ~/.claude/settings.json, stripping any
#      legacy Apex PreToolUse group and preserving every other setting.
#
# It does NOT touch ~/.claude/CLAUDE.md — the Apex spine is opt-in. See README.
#
# Usage:
#   scripts/install.sh            # build + install
#   scripts/install.sh --release  # build the full cross-compile matrix first
#   scripts/install.sh --no-build # skip the build, install the existing binary
#   scripts/install.sh --help
#
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_DIR="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"

# On Windows (Git Bash / MSYS / Cygwin) the Go toolchain emits `apex.exe`, so
# every file op and the wired hook command must use that exact name. Detecting
# the host here keeps the build check, the copy, and the hook in agreement.
# $CONFIG_DIR is already an MSYS forward-slash path (/c/Users/...), so the hook
# command we write is bash-safe; a backslash Windows path would be mangled when
# Claude Code runs the hook through bash (\U \d ... get eaten as escapes).
EXE=""
case "$(uname -s)" in
  MINGW*|MSYS*|CYGWIN*) EXE=".exe" ;;
esac
APEX_BIN="$CONFIG_DIR/bin/apex$EXE"

PLUGIN_ID="apex-claude@apex-claude"
MARKETPLACE="apex-claude"

BUILD=1
RELEASE=0

for arg in "$@"; do
  case "$arg" in
    --release)  RELEASE=1 ;;
    --no-build) BUILD=0 ;;
    -h|--help)
      sed -n '2,/^set -euo/p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//; /^set -euo/d'
      exit 0 ;;
    *)
      echo "install.sh: unknown argument '$arg' (try --help)" >&2
      exit 2 ;;
  esac
done

say()  { printf '\033[1m==>\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }
have() { command -v "$1" >/dev/null 2>&1; }

# --- preflight ---------------------------------------------------------------
have python3 || die "'python3' is required to merge hooks into settings.json"
if [ "$BUILD" -eq 1 ]; then
  have go   || die "'go' is not on PATH — needed to build the apex binary (or pass --no-build)"
  have make || die "'make' is not on PATH — needed to build (or pass --no-build)"
fi

cd "$REPO_ROOT"

# --- 1. build ----------------------------------------------------------------
if [ "$BUILD" -eq 1 ]; then
  say "Cleaning bin/ then building"
  rm -rf bin
  make build
  [ "$RELEASE" -eq 1 ] && make release
  [ -x "bin/apex$EXE" ] || die "build did not produce an executable bin/apex$EXE"
else
  say "Skipping build (--no-build)"
  [ -x "bin/apex$EXE" ] || die "--no-build set but bin/apex$EXE is missing — build it first"
fi

# --- 2. drop any prior plugin install (migration) ----------------------------
if have claude && claude plugin list 2>/dev/null | grep -q "$PLUGIN_ID"; then
  say "Removing prior plugin install of Apex (switching to loose artifacts)"
  claude plugin uninstall "$PLUGIN_ID" || true
  claude plugin marketplace remove "$MARKETPLACE" 2>/dev/null || true
fi

# --- 3. copy artifacts -------------------------------------------------------
say "Installing artifacts into $CONFIG_DIR/"
mkdir -p "$CONFIG_DIR"/{commands,agents,skills,output-styles,bin}
# Prune ax-* artifacts this release no longer ships, so a command cut from Apex
# disappears from the install instead of lingering in the slash menu. ax-* is
# Apex's namespace (uninstall removes the whole prefix); other files are the
# user's and are never touched. Skips a kind the source doesn't ship at all, so
# a malformed bundle can never wipe the installed set.
prune_unshipped() { # <src-dir> <dst-dir> <glob>
  ls -d "$1"/$3 >/dev/null 2>&1 || return 0
  for f in "$2"/$3; do
    [ -e "$f" ] || continue
    [ -e "$1/$(basename "$f")" ] || rm -rf "$f"
  done
}
prune_unshipped "$REPO_ROOT/commands" "$CONFIG_DIR/commands" 'ax-*.md'
prune_unshipped "$REPO_ROOT/agents"   "$CONFIG_DIR/agents"   'ax-*.md'
prune_unshipped "$REPO_ROOT/skills"   "$CONFIG_DIR/skills"   'ax-*'
cp commands/ax-*.md          "$CONFIG_DIR/commands/"
cp agents/ax-*.md            "$CONFIG_DIR/agents/"
cp output-styles/protocol.md "$CONFIG_DIR/output-styles/apex.md"
# Repair residue from a prior buggy install: a trailing-slash `cp -r src/ dest/`
# collapsed every skill into a single top-level skills/SKILL.md. Skills are
# always dir/SKILL.md, so a file directly under skills/ is invalid layout —
# remove it so the orphan doesn't linger.
rm -f "$CONFIG_DIR/skills/SKILL.md"
for d in skills/ax-*/; do
  dest="$CONFIG_DIR/skills/$(basename "$d")"
  rm -rf "$dest"
  # Copy to an explicit dest dir, NOT into skills/ with a trailing-slash source:
  # BSD/macOS `cp -r src/ dest/` copies src's *contents*, collapsing every skill
  # into one. Naming the dest dir copies the directory itself on both BSD + GNU.
  cp -r "$d" "$dest"
done

# --- 4. binary ---------------------------------------------------------------
say "Installing binary into $APEX_BIN"
cp "bin/apex$EXE" "$APEX_BIN"
chmod +x "$APEX_BIN"

# Detect whether the install dir is reachable on PATH. The binary lives under
# ~/.claude/bin, which is not on a default PATH — so `apex` won't resolve in a
# fresh shell unless the user wires it. We never mutate the user's rc; we only
# tell them the exact line to add. Empty ON_PATH => print guidance at the end.
# Match trailing-slash PATH entries and a symlinked bin dir too — an exact
# string compare false-negatives on both.
BIN_DIR="$(dirname "$APEX_BIN")"
REAL_BIN_DIR="$(cd "$BIN_DIR" 2>/dev/null && pwd -P)" || REAL_BIN_DIR="$BIN_DIR"
ON_PATH=""
case ":$PATH:" in
  *":$BIN_DIR:"* | *":$BIN_DIR/:"* | *":$REAL_BIN_DIR:"* | *":$REAL_BIN_DIR/:"*) ON_PATH=1 ;;
esac

# --- 5. wire hooks (preserve all other settings) -----------------------------
say "Wiring hooks into $CONFIG_DIR/settings.json"
python3 - "$CONFIG_DIR/settings.json" "$APEX_BIN" <<'PY'
import json, sys
settings_path, apex_bin = sys.argv[1], sys.argv[2]
try:
    with open(settings_path) as f:
        data = json.load(f)
except FileNotFoundError:
    data = {}
except json.JSONDecodeError as e:
    sys.exit(f"settings.json is not valid JSON ({e}); fix it by hand and re-run")

hooks = data.setdefault("hooks", {})

def is_apex(group):
    # Match both `apex hooks` (Unix) and `apex.exe hooks` (Windows) so re-runs
    # on Windows strip the prior group instead of stacking a duplicate.
    return any("apex hooks" in (c := h.get("command", "")) or "apex.exe hooks" in c
               for h in group.get("hooks", []))

# Strip any prior apex groups so re-runs don't stack duplicates. PreToolUse is
# stripped but never re-added: Apex no longer ships a bash guard (Claude Code's
# own auto-mode owns that), so an upgrade from an older Apex must clean the
# stale PreToolUse entry rather than leave it pointing at a removed subcommand.
for event in ("PreToolUse", "SessionStart"):
    if event in hooks:
        hooks[event] = [g for g in hooks[event] if not is_apex(g)]
        if not hooks[event]:
            del hooks[event]

hooks.setdefault("SessionStart", []).append({
    "hooks": [{"type": "command", "command": f"{apex_bin} hooks session-start"}],
})

with open(settings_path, "w") as f:
    json.dump(data, f, indent=2)
    f.write("\n")
print(f"  hooks wired -> {apex_bin}")
PY

# --- done --------------------------------------------------------------------
cat <<EOF

$(printf '\033[1;32m✔ Apex Claude installed (loose artifacts).\033[0m')

  commands : $(ls commands/ax-*.md | wc -l | tr -d ' ')  → $CONFIG_DIR/commands/
  agents   : $(ls agents/ax-*.md | wc -l | tr -d ' ')  → $CONFIG_DIR/agents/
  skills   : $(ls -d skills/ax-*/ | wc -l | tr -d ' ')   → $CONFIG_DIR/skills/
  style    : Apex → $CONFIG_DIR/output-styles/apex.md

Next steps:
  • Restart Claude Code so /ax-* commands, agents, skills, and hooks load.
  • Activate the output style: /output-style Apex
  • Remove later with: scripts/uninstall.sh

Re-run this script after any code change to refresh the installed binary.
EOF

if [ -z "$ON_PATH" ]; then
  cat <<EOF
$(printf '\033[1;33m! %s is not on your PATH\033[0m' "$BIN_DIR")
  The 'apex' command resolves for Claude Code's hooks (they call the full path),
  but to run 'apex' yourself, add this line to your shell rc (~/.bashrc or
  ~/.zshrc) and start a new shell:

    export PATH="$BIN_DIR:\$PATH"

EOF
fi
