package scenarios

import (
	"github.com/fluid-cloudnative/fluid-introspector/fluid-introspector/pkg/types"
)

// Scenario represents a predefined mock scenario.
type Scenario struct {
	Name        string
	Description string
	Graph       *types.ResourceGraph
}

// Get finds a scenario by name. Returns nil if not found.
func Get(name string) *Scenario {
	for _, s := range All {
		if s.Name == name {
			return &s
		}
	}
	return nil
}

// All scenarios.
var All = []Scenario{
	{
		Name:        "healthy",
		Description: "A fully functional Dataset with ready Runtime and Infrastructure.",
		Graph: &types.ResourceGraph{
			Dataset: &types.DatasetInfo{Name: "demo-data", Status: "Bound", Phase: "Ready"},
			Runtime: &types.RuntimeInfo{
				Name:   "demo-data",
				Type:   "AlluxioRuntime",
				Master: &types.ComponentInfo{Name: "demo-data-master", Replicas: 1, Ready: 1, State: "Ready"},
				Worker: &types.ComponentInfo{Name: "demo-data-worker", Replicas: 3, Ready: 3, State: "Ready"},
				Fuse:   &types.ComponentInfo{Name: "demo-data-fuse", Replicas: 5, Ready: 5, State: "Ready"},
			},
			Infrastructure: &types.InfrastructureInfo{
				PVC: &types.PVCInfo{Name: "demo-data", Status: "Bound"},
				PV:  &types.PVInfo{Name: "pv-demo-data", Status: "Bound"},
			},
		},
	},
	{
		Name:        "missing-runtime",
		Description: "Dataset created but no runtime associated (Runtime == nil).",
		Graph: &types.ResourceGraph{
			Dataset: &types.DatasetInfo{Name: "demo-data", Status: "NotBound", Phase: "NotReady"},
			Runtime: nil, // Trigger RUNTIME_MISSING
		},
	},
	{
		Name:        "partial-ready",
		Description: "One worker pod is failing (2/3 Ready).",
		Graph: &types.ResourceGraph{
			Dataset: &types.DatasetInfo{Name: "demo-data", Status: "Bound", Phase: "Processing"},
			Runtime: &types.RuntimeInfo{
				Name:   "demo-data",
				Type:   "AlluxioRuntime",
				Master: &types.ComponentInfo{Name: "demo-data-master", Replicas: 1, Ready: 1, State: "Ready"},
				Worker: &types.ComponentInfo{Name: "demo-data-worker", Replicas: 3, Ready: 2, State: "PartialReady"}, // Trigger WORKER_PARTIALLY_READY
				Fuse:   &types.ComponentInfo{Name: "demo-data-fuse", Replicas: 5, Ready: 5, State: "Ready"},
			},
			Infrastructure: &types.InfrastructureInfo{
				PVC: &types.PVCInfo{Name: "demo-data", Status: "Bound"},
			},
		},
	},
	{
		Name:        "missing-fuse",
		Description: "Fuse daemonset has 0 ready replicas.",
		Graph: &types.ResourceGraph{
			Dataset: &types.DatasetInfo{Name: "demo-data", Status: "Bound", Phase: "Bound"},
			Runtime: &types.RuntimeInfo{
				Name:   "demo-data",
				Type:   "AlluxioRuntime",
				Master: &types.ComponentInfo{Name: "demo-data-master", Replicas: 1, Ready: 1},
				Worker: &types.ComponentInfo{Name: "demo-data-worker", Replicas: 3, Ready: 3},
				Fuse:   &types.ComponentInfo{Name: "demo-data-fuse", Replicas: 5, Ready: 0, State: "NotReady"}, // Trigger FUSE_MISSING
			},
			Infrastructure: &types.InfrastructureInfo{
				PVC: &types.PVCInfo{Name: "demo-data", Status: "Bound"},
			},
		},
	},
	{
		Name:        "failed-pods",
		Description: "Multiple components failing simultaneously.",
		Graph: &types.ResourceGraph{
			Dataset: &types.DatasetInfo{Name: "demo-data", Status: "Bound"},
			Runtime: &types.RuntimeInfo{
				Name:   "demo-data",
				Type:   "JindoRuntime",
				Master: &types.ComponentInfo{Name: "demo-data-master", Replicas: 1, Ready: 0}, // Fail Master
				Worker: &types.ComponentInfo{Name: "demo-data-worker", Replicas: 3, Ready: 1}, // Fail Worker
				Fuse:   &types.ComponentInfo{Name: "demo-data-fuse", Replicas: 2, Ready: 2},   // Fuse OK
			},
			Infrastructure: &types.InfrastructureInfo{
				PVC: &types.PVCInfo{Name: "demo-data", Status: "Bound"},
			},
		},
	},
}
