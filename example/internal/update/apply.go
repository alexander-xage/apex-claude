package update

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"apexclaude/internal/version"
)

// ApplyResult reports the outcome of a successful Apply call.
type ApplyResult struct {
	Old      string // running version tag before the update, e.g. "v0.2.0"
	New      string // installed version tag, e.g. "v0.3.0" (== Old when UpToDate)
	UpToDate bool   // true when no update was needed (latest == current, no --to pin)
}

// defaultDownloadBaseURL is the GitHub release-asset download prefix for
// RepoSlug: assets live at <defaultDownloadBaseURL>/<tag>/<asset>.
const defaultDownloadBaseURL = "https://github.com/" + RepoSlug + "/releases/download"

// downloadBaseURL resolves the base Apply downloads release assets from.
// APEX_UPDATE_BASE_URL overrides it wholesale — the same seam LatestTag
// respects — so a single httptest server can serve both the latest-redirect
// probe (at its root) and tagged asset downloads (at <base>/<tag>/<asset>).
func downloadBaseURL() string {
	if u := os.Getenv("APEX_UPDATE_BASE_URL"); u != "" {
		return u
	}
	return defaultDownloadBaseURL
}

func downloadURL(tag, asset string) string {
	return strings.TrimRight(downloadBaseURL(), "/") + "/" + tag + "/" + asset
}

// assetName is the release zip filename for the running platform, per the
// frozen naming contract: apex-claude-<GOOS>-<GOARCH>.zip.
func assetName() string {
	return fmt.Sprintf("apex-claude-%s-%s.zip", runtime.GOOS, runtime.GOARCH)
}

// ValidTag reports whether t matches the vMAJOR.MINOR.PATCH tag format the
// release pipeline publishes.
func ValidTag(t string) bool {
	return tagRe.MatchString(t)
}

// Apply downloads, verifies, and installs the release bundle for tag `to`
// (or the latest release when to is "") into root, replacing the binary and
// artifacts in lockstep. Callers own the layout guard — Apply does not check
// layout.IsLooseInstall itself. When to is "" and the resolved latest tag
// equals the running version, Apply is a no-op (UpToDate=true) and touches
// neither the network again nor the filesystem. An explicit --to pin is
// always applied, even when it names the running version (downgrades/
// reinstalls are allowed by design — no special-casing).
//
// On any failure before the binary/artifact writes begin (download, missing
// SHA256SUMS line, checksum mismatch, zip-slip, malformed zip), root is left
// completely untouched.
func Apply(root, to string) (ApplyResult, error) {
	// A malformed pin would be concatenated into the download URL — reject
	// before any network or disk activity.
	if to != "" && !tagRe.MatchString(to) {
		return ApplyResult{}, fmt.Errorf("update: invalid tag %q — expected vMAJOR.MINOR.PATCH", to)
	}

	// Sweep before the up-to-date early return: an already-current install
	// is exactly when a leftover .old from the last swap lingers longest.
	RemoveStaleWindowsOld(root)

	cur := "v" + version.Version
	tag := to
	if tag == "" {
		t, err := LatestTag()
		if err != nil {
			return ApplyResult{}, err
		}
		tag = t
		if tag == cur {
			return ApplyResult{Old: cur, New: cur, UpToDate: true}, nil
		}
	}

	asset := assetName()

	dlDir, err := os.MkdirTemp("", "apex-update-dl-*")
	if err != nil {
		return ApplyResult{}, fmt.Errorf("update: temp download dir: %w", err)
	}
	defer os.RemoveAll(dlDir)

	zipPath, err := downloadFile(downloadURL(tag, asset), dlDir, asset)
	if err != nil {
		return ApplyResult{}, err
	}

	sums, err := fetchText(downloadURL(tag, "SHA256SUMS"))
	if err != nil {
		return ApplyResult{}, err
	}
	if err := verifyChecksum(sums, asset, zipPath); err != nil {
		return ApplyResult{}, err
	}

	extractDir, err := os.MkdirTemp("", "apex-update-extract-*")
	if err != nil {
		return ApplyResult{}, fmt.Errorf("update: temp extract dir: %w", err)
	}
	defer os.RemoveAll(extractDir)

	if err := extractZip(zipPath, extractDir); err != nil {
		return ApplyResult{}, err
	}

	if err := applyArtifacts(extractDir, root); err != nil {
		return ApplyResult{}, err
	}
	if err := swapBinary(extractDir, root); err != nil {
		return ApplyResult{}, err
	}

	// Best-effort: a rewrite failure here does not undo an already-applied
	// update, it just leaves the cache stale until the next check/apply.
	_ = WriteCache(CachePath(), Cache{
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
		Latest:    tag,
	})

	return ApplyResult{Old: cur, New: tag}, nil
}

// --- download + verify -------------------------------------------------

func downloadFile(url, dir, name string) (string, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("update: download %s: %w", name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("update: download %s: HTTP %d", name, resp.StatusCode)
	}

	path := filepath.Join(dir, name)
	out, err := os.Create(path)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		return "", fmt.Errorf("update: save %s: %w", name, err)
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	return path, nil
}

func fetchText(url string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("update: fetch SHA256SUMS: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("update: fetch SHA256SUMS: HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// verifyChecksum parses sums (coreutils `sha256sum` two-space format) and
// checks the sha256 of the file at path against the line naming asset.
func verifyChecksum(sums, asset, path string) error {
	want := ""
	for _, line := range strings.Split(sums, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) != 2 {
			continue
		}
		if parts[1] == asset {
			want = parts[0]
			break
		}
	}
	if want == "" {
		return fmt.Errorf("update: no SHA256SUMS entry for %s", asset)
	}
	got, err := sha256File(path)
	if err != nil {
		return err
	}
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("update: checksum mismatch for %s", asset)
	}
	return nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// --- extract (zip-slip guarded) -----------------------------------------

// extractZip extracts the zip at zipPath into destDir. Any entry whose
// cleaned path would resolve outside destDir (a "zip-slip" entry, e.g.
// "../evil") is rejected with an error before anything is written for that
// entry.
func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("update: open bundle zip: %w", err)
	}
	defer r.Close()

	destAbs, err := filepath.Abs(destDir)
	if err != nil {
		return err
	}

	for _, f := range r.File {
		cleaned := filepath.Clean(f.Name)
		target := filepath.Join(destAbs, cleaned)
		targetAbs, err := filepath.Abs(target)
		if err != nil {
			return err
		}
		if targetAbs != destAbs && !strings.HasPrefix(targetAbs, destAbs+string(os.PathSeparator)) {
			return fmt.Errorf("update: zip entry %q escapes extract root", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetAbs, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(targetAbs), 0o755); err != nil {
			return err
		}
		if err := extractZipEntry(f, targetAbs); err != nil {
			return err
		}
	}
	return nil
}

func extractZipEntry(f *zip.File, targetAbs string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	mode := f.Mode()
	if mode == 0 {
		mode = 0o644
	}
	out, err := os.OpenFile(targetAbs, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, rc); err != nil {
		return err
	}
	return nil
}

// --- apply artifacts (installer semantics) -------------------------------

// applyArtifacts copies the extracted bundle into root with the installers'
// replace semantics: overwrite commands/ax-*.md, agents/ax-*.md and
// output-styles/apex.md; wholesale-replace each skills/ax-* dir.
//
// It then prunes ax-* artifacts the bundle no longer ships, so a release that
// cuts a command actually removes it from every updated install. ax-* is
// Apex's namespace — uninstall already deletes the whole prefix — so files
// without it are the user's and are never touched. Pruning is keyed on what
// the bundle ships: a directory the bundle leaves absent or empty is treated
// as malformed, not as "delete everything".
func applyArtifacts(extractDir, root string) error {
	for _, sub := range []string{"commands", "agents"} {
		shipped, err := overwriteGlob(extractDir, root, sub, "ax-*.md")
		if err != nil {
			return err
		}
		if err := pruneUnshipped(filepath.Join(root, sub), "ax-*.md", shipped); err != nil {
			return err
		}
	}

	src := filepath.Join(extractDir, "output-styles", "apex.md")
	if _, err := os.Stat(src); err == nil {
		if err := copyFile(src, filepath.Join(root, "output-styles", "apex.md"), 0o644); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	skillsSrc := filepath.Join(extractDir, "skills")
	entries, err := os.ReadDir(skillsSrc)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	shipped := map[string]bool{}
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "ax-") {
			continue
		}
		shipped[e.Name()] = true
		dst := filepath.Join(root, "skills", e.Name())
		if err := os.RemoveAll(dst); err != nil {
			return err
		}
		if err := copyDir(filepath.Join(skillsSrc, e.Name()), dst); err != nil {
			return err
		}
	}
	return pruneUnshipped(filepath.Join(root, "skills"), "ax-*", shipped)
}

// overwriteGlob copies every extractDir/sub/pattern match into root/sub and
// returns the set of base names it shipped.
func overwriteGlob(extractDir, root, sub, pattern string) (map[string]bool, error) {
	matches, err := filepath.Glob(filepath.Join(extractDir, sub, pattern))
	if err != nil {
		return nil, err
	}
	shipped := map[string]bool{}
	for _, m := range matches {
		name := filepath.Base(m)
		if err := copyFile(m, filepath.Join(root, sub, name), 0o644); err != nil {
			return nil, err
		}
		shipped[name] = true
	}
	return shipped, nil
}

// pruneUnshipped removes dir/pattern entries whose names are not in shipped.
// An empty shipped set prunes nothing (see applyArtifacts).
func pruneUnshipped(dir, pattern string, shipped map[string]bool) error {
	if len(shipped) == 0 {
		return nil
	}
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return err
	}
	for _, m := range matches {
		if shipped[filepath.Base(m)] {
			continue
		}
		if err := os.RemoveAll(m); err != nil {
			return err
		}
	}
	return nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// --- binary swap ----------------------------------------------------------

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "apex.exe"
	}
	return "apex"
}

// RemoveStaleWindowsOld best-effort removes a leftover apex.exe.old from a
// prior interrupted Windows swap (see swapBinaryWindows). Silent on
// failure; a no-op on non-Windows. Exported so internal/hooks.SessionStart
// can reuse the same sweep instead of duplicating it (checkpoint 5).
func RemoveStaleWindowsOld(root string) {
	if runtime.GOOS != "windows" {
		return
	}
	_ = os.Remove(filepath.Join(root, "bin", "apex.exe.old"))
}

// writeBinary copies src to dst with the given mode. It is a package-level
// var (not a plain call to copyFile) so tests can substitute a failing
// implementation to exercise the Windows rename-dance rollback path without
// relying on OS-level permission tricks.
var writeBinary = copyFile

func swapBinary(extractDir, root string) error {
	name := binaryName()
	src := filepath.Join(extractDir, name)
	dst := filepath.Join(root, "bin", name)

	if runtime.GOOS == "windows" {
		return swapBinaryWindows(src, dst)
	}
	return swapBinaryUnix(src, dst)
}

// swapBinaryUnix writes the new binary alongside the old one, then renames
// over it — atomic on the same filesystem.
func swapBinaryUnix(src, dst string) error {
	tmp := filepath.Join(filepath.Dir(dst), ".apex.new")
	if err := writeBinary(src, tmp, 0o755); err != nil {
		return fmt.Errorf("update: stage new binary: %w", err)
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp) // don't strand the staged binary
		return fmt.Errorf("update: swap binary: %w", err)
	}
	return nil
}

// swapBinaryWindows implements the rename dance: a running exe cannot be
// opened for write on Windows but CAN be renamed. apex.exe -> apex.exe.old,
// write new apex.exe; on write failure, rename .old back (rollback). The
// .old file is intentionally left behind on success — it still backs the
// running process — and is cleaned up best-effort at the start of the NEXT
// apex update (RemoveStaleWindowsOld) or session-start hook (CP5).
func swapBinaryWindows(src, dst string) error {
	old := dst + ".old"
	if err := os.Rename(dst, old); err != nil {
		return fmt.Errorf("update: rename running binary aside: %w", err)
	}
	if err := writeBinary(src, dst, 0o755); err != nil {
		if rerr := os.Rename(old, dst); rerr != nil {
			return fmt.Errorf("update: write new binary: %w (rollback also failed: %v — restore manually: rename %s to %s)", err, rerr, old, dst)
		}
		return fmt.Errorf("update: write new binary (rolled back): %w", err)
	}
	return nil
}
