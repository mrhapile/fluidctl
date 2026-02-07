package diagnose

import (
	"fmt"
	"sort"
	"time"

	"github.com/fluid-cloudnative/fluid-introspector/fluid-introspector/pkg/types"
)

// Diagnose evaluates the provided ResourceGraph against a set of registered rules.
// It returns a deterministic DiagnosticResult.
func Diagnose(graph *types.ResourceGraph) *types.DiagnosticResult {
	if graph == nil {
		return nil
	}

	result := &types.DiagnosticResult{
		Timestamp:     time.Now(),
		ResourceGraph: graph,
		IsHealthy:     true,
	}

	var allHints []types.FailureHint

	// 1. Iterate Rules
	// We use the 'rules' slice defined in rules.go, which guarantees order.
	for _, rule := range rules {
		// Evaluate
		hint := rule.Evaluate(graph)
		if hint != nil {
			allHints = append(allHints, *hint)
			result.IsHealthy = false
		}
	}

	// 2. Sort Hints for Determinism
	// Rules are already executed in order, but we can sort by severity as requested:
	// Severity (Critical > Warning) -> Component -> ID
	sort.SliceStable(allHints, func(i, j int) bool {
		hi, hj := allHints[i], allHints[j]
		if hi.Severity != hj.Severity {
			// e.g. "Critical" (8 chars) vs "Warning" (7 chars). Crude.
			// Better: define numeric precedence.
			return severityRank(hi.Severity) > severityRank(hj.Severity)
		}
		if hi.Component != hj.Component {
			return hi.Component < hj.Component
		}
		// Tie-breaker: Evidence Name or Detail?
		return hi.Evidence.Name < hj.Evidence.Name
	})

	result.FailureHints = allHints
	result.Summary = generateSummary(result.IsHealthy, allHints)

	return result
}

// severityRank helps sort FailureHints by importance.
func severityRank(s types.SeverityLevel) int {
	switch s {
	case types.SeverityCritical:
		return 3
	case types.SeverityWarning:
		return 2
	case types.SeverityInfo:
		return 1
	default:
		return 0
	}
}

func generateSummary(healthy bool, hints []types.FailureHint) string {
	if healthy {
		return "Dataset is healthy and all components are ready."
	}
	// Simple summary: "Found X issues: Y critical, Z warnings."
	crit := 0
	warn := 0
	for _, h := range hints {
		if h.Severity == types.SeverityCritical {
			crit++
		} else if h.Severity == types.SeverityWarning {
			warn++
		}
	}
	return fmt.Sprintf("Found %d issues: %d critical, %d warnings.", len(hints), crit, warn)
}
