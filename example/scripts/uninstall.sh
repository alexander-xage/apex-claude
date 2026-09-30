#!/usr/bin/env bash
#
# uninstall.sh — remove the loose Apex Claude artifacts installed by install.sh.
#
# Removes the ax-* commands/agents/skills, the apex output style, the binary,
# and the apex hook entries from ~/.claude/settings.json. Every other setting
# and any non-Apex artifact is left untouched. It does NOT touch ~/.claude/CLAUDE.md.
#
# Usage:
#   scripts/uninstall.sh
#   scripts/uninstall.sh --help
#
set -euo pipefail

CONFIG_DIR="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"

# On Windows the installed binary is apex.exe; on Unix it is apex. Detect the
# host so removal targets the right name (the hook-stripping below already
# matches both `apex hooks` and `apex.exe hooks`).
EXE=""
case "$(uname -s)" in
  MINGW*|MSYS*|CYGWIN*) EXE=".exe" ;;
esac

for arg in "$@"; do
  case "$arg" in
    -h|--help)
      sed -n '2,/^set -euo/p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//; /^set -euo/d'
      exit 0 ;;
    *) echo "uninstall.sh: unknown argument '$arg'" >&2; exit 2 ;;
  esac
done

say() { printf '\033[1m==>\033[0m %s\n' "$*"; }
have() { command -v "$1" >/dev/null 2>&1; }

say "Removing Apex artifacts from $CONFIG_DIR/"
rm -fv "$CONFIG_DIR"/commands/ax-*.md 2>/dev/null || true
rm -fv "$CONFIG_DIR"/agents/ax-*.md   2>/dev/null || true
rm -rfv "$CONFIG_DIR"/skills/ax-*     2>/dev/null || true
rm -fv  "$CONFIG_DIR/output-styles/apex.md" 2>/dev/null || true
rm -fv  "$CONFIG_DIR/bin/apex$EXE"          2>/dev/null || true

if [ -f "$CONFIG_DIR/settings.json" ] && have python3; then
  say "Stripping apex hooks from settings.json"
  python3 - "$CONFIG_DIR/settings.json" <<'PY'
import json, sys
p = sys.argv[1]
try:
    with open(p) as f:
        data = json.load(f)
except (FileNotFoundError, json.JSONDecodeError):
    sys.exit(0)
hooks = data.get("hooks", {})
def is_apex(group):
    # Match both `apex hooks` (Unix) and `apex.exe hooks` (Windows).
    return any("apex hooks" in (c := h.get("command", "")) or "apex.exe hooks" in c
               for h in group.get("hooks", []))
for event in ("PreToolUse", "SessionStart"):
    if event in hooks:
        hooks[event] = [g for g in hooks[event] if not is_apex(g)]
        if not hooks[event]:
            del hooks[event]
if "hooks" in data and not data["hooks"]:
    del data["hooks"]
with open(p, "w") as f:
    json.dump(data, f, indent=2)
    f.write("\n")
PY
fi

printf '\033[1;32m✔ Apex Claude removed.\033[0m Restart Claude Code to drop /ax-*.\n'
