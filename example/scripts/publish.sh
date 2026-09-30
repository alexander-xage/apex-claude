#!/usr/bin/env bash
#
# publish.sh — cut a GitHub Release of Apex Claude with prebuilt bundles.
#
# The Linux/macOS counterpart to scripts/publish.ps1. Cross-compiles the apex
# backbone for the release matrix and, for each platform, bundles the binary
# together with the loose artifacts (commands / agents / skills / output-style)
# into a single zip:
#
#     apex-claude-<os>-<arch>.zip
#       apex(.exe)
#       commands/ax-*.md
#       agents/ax-*.md
#       skills/ax-*/...
#       output-styles/apex.md
#
# The zips are uploaded to a GitHub Release via `gh`, alongside both one-line
# installers (install.ps1 for Windows, install-release.sh for Linux/macOS), so
# a fresh box needs no Go, make, bash, or python — just the prebuilt zip.
#
# This is the maintainer "ship to prod" step. End users never run it.
#
# Usage:
#   scripts/publish.sh                 # tag from main.go, publish
#   scripts/publish.sh --version v0.3.0
#   scripts/publish.sh --dry-run       # build + bundle into dist/, no release
#   scripts/publish.sh --help
#
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO="FNGApex/apex-claude"
DIST="$REPO_ROOT/dist"
STAGE="$DIST/stage"

# Release matrix — mirrors RELEASE_TARGETS in the Makefile and $Targets in publish.ps1.
TARGETS="darwin/arm64 darwin/amd64 linux/amd64 linux/arm64 windows/amd64"

VERSION=""
DRYRUN=0

while [ $# -gt 0 ]; do
  case "$1" in
    --version) VERSION="${2:-}"; shift 2 ;;
    --version=*) VERSION="${1#*=}"; shift ;;
    --dry-run) DRYRUN=1; shift ;;
    -h|--help)
      sed -n '2,/^set -euo/p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//; /^set -euo/d'
      exit 0 ;;
    *) echo "publish.sh: unknown argument '$1' (try --help)" >&2; exit 2 ;;
  esac
done

say()  { printf '\033[1m==>\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }
have() { command -v "$1" >/dev/null 2>&1; }

# --- preflight ---------------------------------------------------------------
have go || die "'go' is not on PATH — needed to build the release matrix"
have zip || have python3 || die "need 'zip' or 'python3' to create the release bundles"
if [ "$DRYRUN" -eq 0 ]; then
  have gh || die "'gh' is not on PATH — needed to create the GitHub Release (or pass --dry-run)"
fi

# --- resolve version ---------------------------------------------------------
# The const lives in internal/version (it moved out of cmd/apex/main.go). A
# release tag that disagrees with the embedded const would ship binaries whose
# `apex version` and update check report the wrong version, so an explicit
# --version must match it — the same guard publish.ps1 enforces.
version_go="$REPO_ROOT/internal/version/version.go"
raw="$(sed -n 's/^const Version = "\([^"]*\)".*/\1/p' "$version_go" | head -1)"
[ -n "$raw" ] || die "could not read Version const from $version_go"
CONST_VERSION="v$raw"
if [ -z "$VERSION" ]; then
  VERSION="$CONST_VERSION"
elif [ "$VERSION" != "$CONST_VERSION" ]; then
  die "--version $VERSION does not match the const in $version_go ($CONST_VERSION) — bump the const or drop --version"
fi
case "$VERSION" in
  v[0-9]*.[0-9]*.[0-9]*) ;;
  *) die "version '$VERSION' should look like v1.2.3" ;;
esac
say "Publishing $VERSION"

# zip the contents of a directory into an archive, preferring `zip`, falling
# back to python3's zipfile module so a stock box without `zip` still works.
make_zip() {
  src_dir="$1"; out_zip="$2"
  if have zip; then
    ( cd "$src_dir" && zip -qr "$out_zip" . )
  else
    python3 - "$src_dir" "$out_zip" <<'PY'
import os, sys, zipfile
src, out = sys.argv[1], sys.argv[2]
with zipfile.ZipFile(out, "w", zipfile.ZIP_DEFLATED) as z:
    for root, _, files in os.walk(src):
        for name in files:
            full = os.path.join(root, name)
            z.write(full, os.path.relpath(full, src))
PY
  fi
}

# --- clean dist --------------------------------------------------------------
rm -rf "$DIST"
mkdir -p "$STAGE"

cd "$REPO_ROOT"
for t in $TARGETS; do
  os="${t%/*}"; arch="${t#*/}"
  ext=""; [ "$os" = "windows" ] && ext=".exe"
  say "building $os/$arch"

  sdir="$STAGE/$os-$arch"
  mkdir -p "$sdir"/{commands,agents,skills,output-styles}

  # Build straight with the Go toolchain — no make dependency. Flags mirror the
  # Makefile release target and publish.ps1.
  GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 \
    go build -trimpath -ldflags "-s -w" -o "$sdir/apex$ext" ./cmd/apex \
    || die "go build failed for $os/$arch"

  # Bundle the platform-independent artifacts alongside the binary.
  cp "$REPO_ROOT"/commands/ax-*.md        "$sdir/commands/"
  cp "$REPO_ROOT"/agents/ax-*.md          "$sdir/agents/"
  cp -r "$REPO_ROOT"/skills/ax-*          "$sdir/skills/"
  cp "$REPO_ROOT"/output-styles/protocol.md "$sdir/output-styles/apex.md"

  out_zip="$DIST/apex-claude-$os-$arch.zip"
  make_zip "$sdir" "$out_zip"
  say "  bundled -> $out_zip"
done

# resolve into an array of zip paths
ZIPS=()
for z in "$DIST"/apex-claude-*.zip; do ZIPS+=("$z"); done
say "Built ${#ZIPS[@]} bundles into $DIST"

# SHA256SUMS in coreutils format (lower-hex, two spaces, bare filename), the
# contract `apex update` and install.ps1 verify against. Without it a release
# published from this script makes every `apex update` fail verification.
# macOS ships shasum, not sha256sum.
if have sha256sum; then hasher="sha256sum"; elif have shasum; then hasher="shasum -a 256"; else die "need 'sha256sum' or 'shasum' to write SHA256SUMS"; fi
SUMS="$DIST/SHA256SUMS"
( cd "$DIST" && $hasher apex-claude-*.zip > SHA256SUMS ) || die "could not write $SUMS"
say "checksums -> $SUMS"

if [ "$DRYRUN" -eq 1 ]; then
  printf '\n\033[1;32m✔ Dry run — bundles in %s, no release created.\033[0m\n' "$DIST"
  exit 0
fi

# --- create / update the GitHub Release --------------------------------------
# Both one-line installers ride along as assets so their download URLs
# (releases/latest/download/<name>) resolve.
INSTALLERS=("$REPO_ROOT/scripts/install.ps1" "$REPO_ROOT/scripts/install-release.sh")

NOTES="Apex Claude $VERSION

Linux / macOS install:
\`\`\`bash
curl -fsSL https://github.com/$REPO/releases/latest/download/install-release.sh | bash
\`\`\`

Windows install:
\`\`\`powershell
irm https://github.com/$REPO/releases/latest/download/install.ps1 | iex
\`\`\`
"

if gh release view "$VERSION" >/dev/null 2>&1; then
  say "Release $VERSION exists — uploading assets (--clobber)"
  gh release upload "$VERSION" "${ZIPS[@]}" "$SUMS" "${INSTALLERS[@]}" --clobber || die "gh release upload failed"
else
  say "Creating release $VERSION"
  gh release create "$VERSION" "${ZIPS[@]}" "$SUMS" "${INSTALLERS[@]}" \
    --title "Apex Claude $VERSION" --notes "$NOTES" || die "gh release create failed"
fi

printf '\n\033[1;32m✔ Published %s.\033[0m\n' "$VERSION"
echo "  Linux/macOS: curl -fsSL https://github.com/$REPO/releases/latest/download/install-release.sh | bash"
echo "  Windows    : irm https://github.com/$REPO/releases/latest/download/install.ps1 | iex"
