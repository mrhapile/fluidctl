package printer

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fluid-cloudnative/fluid-introspector/fluid-introspector/pkg/types"
)

// PrintTree renders a human-readable tree of the diagnostic result.
func PrintTree(result *types.DiagnosticResult) {
	fmt.Printf("\n DIAGNOSTIC REPORT \n")
	fmt.Printf("===================\n")
	if result.IsHealthy {
		fmt.Printf("✓ Dataset is Healthy\n")
	} else {
		fmt.Printf("❌ Dataset is Unhealthy\n")
	}
	fmt.Printf("Summary: %s\n\n", result.Summary)

	if len(result.FailureHints) > 0 {
		fmt.Printf("FINDINGS:\n")
		for _, hint := range result.FailureHints {
			icon := "ℹ"
			if hint.Severity == types.SeverityCritical {
				icon = "❌"
			} else if hint.Severity == types.SeverityWarning {
				icon = "⚠"
			}
			fmt.Printf(" %s [%s] %s\n", icon, hint.Component, hint.ID)
			fmt.Printf("    Evidence: %s (%s)\n", hint.Evidence.Detail, hint.Evidence.Name)
			fmt.Printf("    Suggestion: %s\n\n", hint.Suggestion)
		}
	}

	// Simple graph print (could be more elaborate)
	g := result.ResourceGraph
	if g == nil {
		return
	}
	fmt.Printf("RESOURCE GRAPH:\n")
	fmt.Printf("Dataset: %s (Status: %s)\n", g.Dataset.Name, g.Dataset.Status)
	if g.Runtime != nil {
		fmt.Printf("└── Runtime: %s (%s)\n", g.Runtime.Name, g.Runtime.Type)
		printComponent("Master", g.Runtime.Master)
		printComponent("Worker", g.Runtime.Worker)
		printComponent("Fuse  ", g.Runtime.Fuse)
	} else {
		fmt.Printf("└── Runtime: <Missing>\n")
	}
}

func printComponent(label string, c *types.ComponentInfo) {
	if c == nil {
		return
	}
	status := "✓"
	if c.Ready < c.Replicas {
		status = "⚠"
		if c.Ready == 0 && c.Replicas > 0 {
			status = "❌"
		}
	}
	fmt.Printf("    ├── %s: %s %d/%d Ready\n", status, label, c.Ready, c.Replicas)
}

// PrintJSON renders the full result as JSON.
func PrintJSON(result *types.DiagnosticResult) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(result)
}
