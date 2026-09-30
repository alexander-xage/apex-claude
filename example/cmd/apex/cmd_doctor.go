package main

import (
	"os"

	"apexclaude/internal/doctor"
)

func init() {
	register("doctor", "integrity check on the plugin layout + project state", func(args []string) int {
		return doctor.Run(os.Stdout)
	})
}
