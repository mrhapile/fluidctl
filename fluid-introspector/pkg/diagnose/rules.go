package diagnose

import (
	"fmt"
	"strings"

	"github.com/fluid-cloudnative/fluid-introspector/fluid-introspector/pkg/types"
)

// Rule represents a single diagnostic condition that can check the resource graph.
type Rule interface {
	ID() string
	Evaluate(graph *types.ResourceGraph) *types.FailureHint
}

// Rules registry - deterministic order
var rules = []Rule{
	&DatasetNotBoundRule{},
	&RuntimeMissingRule{},
	&MasterNotReadyRule{},
	&WorkerPartiallyReadyRule{},
	&FuseMissingRule{},
	&PVCNotBoundRule{}, // Renamed from PVCPendingRule
}

// ----------------------------------------------------------------------------
// Rule Implementations
// ----------------------------------------------------------------------------

// DATASET_NOT_BOUND
type DatasetNotBoundRule struct{}

func (r *DatasetNotBoundRule) ID() string { return "DATASET_NOT_BOUND" }

func (r *DatasetNotBoundRule) Evaluate(g *types.ResourceGraph) *types.FailureHint {
	if g.Dataset.Status != "Bound" {
		return &types.FailureHint{
			ID:         r.ID(),
			Severity:   types.SeverityCritical,
			Component:  "Dataset",
			Evidence:   types.Evidence{Kind: "Dataset", Name: g.Dataset.Name, Detail: fmt.Sprintf("Phase: %s, Status: %s", g.Dataset.Phase, g.Dataset.Status)},
			Suggestion: "Check if a Runtime with the same name exists and is compatible.",
		}
	}
	return nil
}

// RUNTIME_MISSING
type RuntimeMissingRule struct{}

func (r *RuntimeMissingRule) ID() string { return "RUNTIME_MISSING" }

func (r *RuntimeMissingRule) Evaluate(g *types.ResourceGraph) *types.FailureHint {
	if g.Runtime == nil {
		return &types.FailureHint{
			ID:         r.ID(),
			Severity:   types.SeverityCritical,
			Component:  "Runtime",
			Evidence:   types.Evidence{Kind: "Runtime", Name: g.Dataset.Name, Detail: "Runtime object is missing from graph."},
			Suggestion: "Create a Runtime CR (e.g., AlluxioRuntime, JindoRuntime) matching the Dataset.",
		}
	}
	return nil
}

// MASTER_NOT_READY
type MasterNotReadyRule struct{}

func (r *MasterNotReadyRule) ID() string { return "MASTER_NOT_READY" }

func (r *MasterNotReadyRule) Evaluate(g *types.ResourceGraph) *types.FailureHint {
	if g.Runtime != nil && g.Runtime.Master != nil {
		if g.Runtime.Master.Ready != g.Runtime.Master.Replicas {
			return &types.FailureHint{
				ID:         r.ID(),
				Severity:   types.SeverityCritical,
				Component:  "Runtime/Master",
				Evidence:   types.Evidence{Kind: "StatefulSet", Name: g.Runtime.Master.Name, Detail: fmt.Sprintf("Ready replicas: %d/%d", g.Runtime.Master.Ready, g.Runtime.Master.Replicas)},
				Suggestion: "Check Master pod logs for startup errors or scheduling issues.",
			}
		}
	}
	return nil
}

// WORKER_PARTIALLY_READY
type WorkerPartiallyReadyRule struct{}

func (r *WorkerPartiallyReadyRule) ID() string { return "WORKER_PARTIALLY_READY" }

func (r *WorkerPartiallyReadyRule) Evaluate(g *types.ResourceGraph) *types.FailureHint {
	if g.Runtime != nil && g.Runtime.Worker != nil {
		if g.Runtime.Worker.Ready < g.Runtime.Worker.Replicas {
			return &types.FailureHint{
				ID:         r.ID(),
				Severity:   types.SeverityWarning,
				Component:  "Runtime/Worker",
				Evidence:   types.Evidence{Kind: "StatefulSet/DaemonSet", Name: g.Runtime.Worker.Name, Detail: fmt.Sprintf("Ready replicas: %d/%d", g.Runtime.Worker.Ready, g.Runtime.Worker.Replicas)},
				Suggestion: "Check individual Worker pods for OOMKilled or CrashLoopBackOff.",
			}
		}
	}
	return nil
}

// FUSE_MISSING
type FuseMissingRule struct{}

func (r *FuseMissingRule) ID() string { return "FUSE_MISSING" }

func (r *FuseMissingRule) Evaluate(g *types.ResourceGraph) *types.FailureHint {
	if g.Runtime != nil && g.Runtime.Fuse != nil {
		if g.Runtime.Fuse.Ready == 0 && g.Runtime.Fuse.Replicas > 0 {
			// If desired replicas > 0 but none are ready, it's considered missing or completely broken.
			return &types.FailureHint{
				ID:         r.ID(),
				Severity:   types.SeverityWarning,
				Component:  "Runtime/Fuse",
				Evidence:   types.Evidence{Kind: "DaemonSet", Name: g.Runtime.Fuse.Name, Detail: fmt.Sprintf("Ready replicas: %d/%d", g.Runtime.Fuse.Ready, g.Runtime.Fuse.Replicas)},
				Suggestion: "Check DaemonSet node selectors and tolerations. Ensure nodes have capacity.",
			}
		}
	}
	return nil
}

// PVC_NOT_BOUND
type PVCNotBoundRule struct{}

func (r *PVCNotBoundRule) ID() string { return "PVC_NOT_BOUND" }

func (r *PVCNotBoundRule) Evaluate(g *types.ResourceGraph) *types.FailureHint {
	if g.Infrastructure != nil && g.Infrastructure.PVC != nil {
		if !strings.EqualFold(g.Infrastructure.PVC.Status, "Bound") {
			return &types.FailureHint{
				ID:         r.ID(),
				Severity:   types.SeverityCritical,
				Component:  "Infrastructure/PVC",
				Evidence:   types.Evidence{Kind: "PersistentVolumeClaim", Name: g.Infrastructure.PVC.Name, Detail: fmt.Sprintf("Status: %s", g.Infrastructure.PVC.Status)},
				Suggestion: "Check PersistentVolume availability or StorageClass configuration.",
			}
		}
	}
	return nil
}
