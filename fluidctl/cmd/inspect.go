package main

import (
	"context"
	"fmt"
	"os"

	"github.com/fluid-cloudnative/fluid-introspector/fluid-introspector/pkg/diagnose"
	"github.com/fluid-cloudnative/fluid-introspector/fluid-introspector/pkg/k8s"
	"github.com/fluid-cloudnative/fluid-introspector/fluid-introspector/pkg/mapper"
	"github.com/fluid-cloudnative/fluid-introspector/fluidctl/pkg/printer"
	"github.com/fluid-cloudnative/fluid-introspector/fluidctl/pkg/scenarios"
	"github.com/spf13/cobra"
)

var (
	inspectNamespace string
	inspectOutput    string
	inspectMock      bool
	inspectScenario  string
)

// inspectCmd represents the inspect command
var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect Fluid resources",
}

var datasetCmd = &cobra.Command{
	Use:   "dataset <name>",
	Short: "Inspect a specific dataset",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		if inspectMock {
			runMock(name, inspectScenario, inspectOutput)
		} else {
			// Real Mode Path
			runReal(name, inspectNamespace, inspectOutput)
		}
	},
}

func init() {
	rootCmd.AddCommand(inspectCmd)
	inspectCmd.AddCommand(datasetCmd)

	// Flags logic - attach to dataset command or inspect command?
	// User example: fluidctl inspect dataset demo-data --mock
	// It's safer to attach to PersistentFlags of inspectCmd if they apply to all inspects,
	// or LocalFlags of datasetCmd. Let's put them on datasetCmd for now as requested.
	datasetCmd.Flags().StringVarP(&inspectNamespace, "namespace", "n", "default", "Kubernetes namespace")
	datasetCmd.Flags().StringVarP(&inspectOutput, "output", "o", "tree", "Output format: tree, json, wide")
	datasetCmd.Flags().BoolVar(&inspectMock, "mock", false, "Use mock data instead of live cluster")
	datasetCmd.Flags().StringVar(&inspectScenario, "scenario", "healthy", "Mock scenario: healthy, partial-ready, missing-runtime, missing-fuse, failed-pods")
}

func runMock(name, scenarioName, outputFormat string) {
	s := scenarios.Get(scenarioName)
	if s == nil {
		fmt.Printf("Error: Scenario '%s' not found. Available: healthy, partial-ready, missing-runtime, missing-fuse, failed-pods\n", scenarioName)
		os.Exit(1)
	}

	// In mock mode, we use the graph from the scenario, but we might want to override the name
	// to match the CLI arg for consistency, essentially simulating "I found *this* dataset".
	// However, the scenario graph has hardcoded names.
	// For now, let's just use the scenario graph as is.

	// Phase 2 Invoke: Diagnose
	result := diagnose.Diagnose(s.Graph)

	// Phase 3 Invoke: Print
	if outputFormat == "json" {
		printer.PrintJSON(result)
	} else {
		fmt.Printf("[MOCK MODE] Scenario: %s\n", s.Description)
		printer.PrintTree(result)
	}
}

func runReal(name, namespace, outputFormat string) {
	// 1. Initialize Client
	cli, err := k8s.NewClient()
	if err != nil {
		fmt.Printf("Error initializing K8s client: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize Mapper
	m := mapper.NewK8sMapper(cli)

	// 3. Map Dataset
	ctx := context.Background()
	graph, err := m.MapDataset(ctx, name, namespace)
	if err != nil {
		fmt.Printf("Error mapping dataset '%s' in namespace '%s': %v\n", name, namespace, err)
		os.Exit(1)
	}

	// 4. Diagnose
	result := diagnose.Diagnose(graph)

	// 5. Print
	if outputFormat == "json" {
		printer.PrintJSON(result)
	} else {
		printer.PrintTree(result)
	}
}
