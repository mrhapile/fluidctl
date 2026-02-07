package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Build-time variables to be injected via -ldflags
var (
	Version   = "v0.1.0"
	Commit    = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long:  `Print the build version, git commit, build date, and Go runtime information for fluidctl.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("fluidctl version: %s\n", Version)
		fmt.Printf("git commit: %s\n", Commit)
		fmt.Printf("build date: %s\n", BuildDate)
		fmt.Printf("go version: %s\n", runtime.Version())
		fmt.Printf("platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
