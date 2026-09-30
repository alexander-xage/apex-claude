package main

import (
	"fmt"
	"os"

	"apexclaude/internal/layout"
	"apexclaude/internal/update"
	"apexclaude/internal/version"
)

func init() {
	register("update", "check for Apex Claude releases, or apply one", runUpdate)
}

func runUpdate(args []string) int {
	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	switch {
	case sub == "check":
		return runUpdateCheck(args[1:])
	case sub == "" || sub == "--to":
		return runUpdateApply(args)
	default:
		fmt.Fprintln(os.Stderr, "usage: apex update [--to vX.Y.Z] | apex update check [--quiet]")
		return 2
	}
}

// runUpdateApply applies an update in place: downloads the release bundle
// for the target tag (--to vX.Y.Z, or latest when omitted), verifies it,
// and replaces artifacts + binary in root. The layout guard and exit-code
// interpretation live here; the download/verify/extract/swap mechanics live
// in internal/update.Apply, matching update check's cmd-thin/package-thick
// split.
func runUpdateApply(args []string) int {
	to := ""
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--to":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "apex update: --to requires a value")
				return 2
			}
			to = args[i+1]
			i++
		default:
			fmt.Fprintf(os.Stderr, "apex update: unknown argument %q\nusage: apex update [--to vX.Y.Z] | apex update check [--quiet]\n", args[i])
			return 2
		}
	}
	if to != "" && !update.ValidTag(to) {
		fmt.Fprintf(os.Stderr, "apex update: invalid tag %q — expected vMAJOR.MINOR.PATCH\n", to)
		return 2
	}

	root := layout.ArtifactRoot()
	if !layout.IsLooseInstall(root) {
		fmt.Fprintln(os.Stderr, "dev layout — use 'git pull && make install'")
		return 2
	}

	res, err := update.Apply(root, to)
	if err != nil {
		fmt.Fprintln(os.Stderr, "apex update:", err)
		return 1
	}
	if res.UpToDate {
		fmt.Printf("apex %s — up to date\n", res.Old)
		return 0
	}
	if !layout.ApexHooksWired(root) {
		fmt.Fprintln(os.Stderr, "warning: apex hooks are not wired in settings.json — rerun install to fix")
	}
	fmt.Printf("updated %s → %s\n", res.Old, res.New)
	return 0
}

// runUpdateCheck refreshes the update cache (a network fetch, regardless of
// TTL — TTL gating is the session-start hook's job) and reports whether a
// newer release is available. --quiet keeps the exit code but suppresses the
// stdout report, matching the form a hook spawns detached; failure
// diagnostics still go to stderr.
func runUpdateCheck(args []string) int {
	quiet := false
	for _, a := range args {
		if a == "--quiet" {
			quiet = true
		}
	}

	cur := "v" + version.Version
	c, err := update.Refresh(update.CachePath())
	if err != nil {
		fmt.Fprintln(os.Stderr, "apex update check:", err)
		return 2
	}
	if update.Compare(c.Latest, cur) > 0 {
		if !quiet {
			fmt.Printf("apex %s — latest %s (update available)\n", cur, c.Latest)
		}
		return 1
	}
	if !quiet {
		fmt.Printf("apex %s — up to date\n", cur)
	}
	return 0
}
