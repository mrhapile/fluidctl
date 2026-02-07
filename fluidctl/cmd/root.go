package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "fluidctl",
	Short: "fluidctl controls the fluid-introspector engine",
	Long: `fluidctl is a CLI wrapper for the Fluid Introspection Engine.
It allows you to inspect, map, and diagnose Fluid datasets and runtimes
without needing extensive grep/awk knowledge.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
