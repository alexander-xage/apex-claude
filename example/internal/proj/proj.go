// Package proj resolves project-scoped paths for the apex backbone.
package proj

import "os"

// Root resolves the project root: $APEX_REPO if set, else the working directory.
func Root() string {
	if r := os.Getenv("APEX_REPO"); r != "" {
		return r
	}
	wd, _ := os.Getwd()
	return wd
}
