package diagnose_test

import (
	"testing"

	"github.com/fluid-cloudnative/fluid-introspector/fluid-introspector/pkg/diagnose"
	"github.com/fluid-cloudnative/fluid-introspector/fluid-introspector/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestDiagnose_Healthy(t *testing.T) {
	// Setup
	graph := &types.ResourceGraph{
		Dataset: &types.DatasetInfo{
			Status: "Bound",
			Phase:  "Ready",
		},
		Runtime: &types.RuntimeInfo{
			Master: &types.ComponentInfo{Ready: 1, Replicas: 1},
			Worker: &types.ComponentInfo{Ready: 3, Replicas: 3},
			Fuse:   &types.ComponentInfo{Ready: 5, Replicas: 5},
		},
		Infrastructure: &types.InfrastructureInfo{
			PVC: &types.PVCInfo{Status: "Bound"},
		},
	}

	// Act
	result := diagnose.Diagnose(graph)

	// Assert
	assert.NotNil(t, result)
	assert.True(t, result.IsHealthy)
	assert.Len(t, result.FailureHints, 0)
	assert.Equal(t, "Dataset is healthy and all components are ready.", result.Summary)
}

func TestDiagnose_RuntimeMissing(t *testing.T) {
	graph := &types.ResourceGraph{
		Dataset: &types.DatasetInfo{Status: "Bound"}, // Should trigger? No, rule checks if nil
		Runtime: nil,
	}

	result := diagnose.Diagnose(graph)

	assert.False(t, result.IsHealthy)
	assert.Len(t, result.FailureHints, 1)
	assert.Equal(t, types.SeverityCritical, result.FailureHints[0].Severity)
	assert.Equal(t, "RUNTIME_MISSING", result.FailureHints[0].ID)
}

func TestDiagnose_WorkerPartialReady(t *testing.T) {
	graph := &types.ResourceGraph{
		Dataset: &types.DatasetInfo{Status: "Bound"},
		Runtime: &types.RuntimeInfo{
			Worker: &types.ComponentInfo{Ready: 2, Replicas: 3}, // 2/3 Ready -> Warning
		},
	}

	result := diagnose.Diagnose(graph)

	assert.False(t, result.IsHealthy)
	hint := result.FailureHints[0]
	assert.Equal(t, types.SeverityWarning, hint.Severity)
	assert.Equal(t, "WORKER_PARTIALLY_READY", hint.ID)
	assert.Contains(t, hint.Evidence.Detail, "Ready replicas: 2/3")
}

func TestDiagnose_ComplexFailure(t *testing.T) {
	// Scenario: PVC Pending AND Worker Crash
	graph := &types.ResourceGraph{
		Dataset: &types.DatasetInfo{Status: "Bound"},
		Runtime: &types.RuntimeInfo{
			Worker: &types.ComponentInfo{Ready: 0, Replicas: 1},
		},
		Infrastructure: &types.InfrastructureInfo{
			PVC: &types.PVCInfo{Status: "Pending"},
		},
	}

	result := diagnose.Diagnose(graph)

	assert.Len(t, result.FailureHints, 2)
	// Expected Order:
	// 1. PVC_NOT_BOUND (Critical)
	// 2. WORKER_PARTIALLY_READY (Warning)

	assert.Equal(t, types.SeverityCritical, result.FailureHints[0].Severity)
	assert.Equal(t, "PVC_NOT_BOUND", result.FailureHints[0].ID)

	assert.Equal(t, types.SeverityWarning, result.FailureHints[1].Severity)
	assert.Equal(t, "WORKER_PARTIALLY_READY", result.FailureHints[1].ID)
}
