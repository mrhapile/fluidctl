package types

import (
	"time"
)

// DiagnosticResult encapsulates the overall health assessment and diagnosis.
type DiagnosticResult struct {
	Timestamp     time.Time      `json:"timestamp"`               // Time of diagnosis
	IsHealthy     bool           `json:"isHealthy"`               // High-level health indicator
	Summary       string         `json:"summary"`                 // Brief sentence like "Dataset is Bound but Runtime partially unready."
	FailureHints  []FailureHint  `json:"failureHints"`            // Specific findings
	ResourceGraph *ResourceGraph `json:"resourceGraph,omitempty"` // Context
}

// FailureHint describes a detected issue with severity and remediation suggestions.
type FailureHint struct {
	ID         string        `json:"id"`                // Unique identifier for the rule (e.g., DATASET_NOT_BOUND)
	Severity   SeverityLevel `json:"severity"`          // E.g., CRITICAL, WARNING, INFO
	Component  string        `json:"component"`         // E.g., "Worker", "PVC"
	Evidence   Evidence      `json:"evidence"`          // Concrete data supporting the finding
	Suggestion string        `json:"suggestion"`        // Actionable step for the user
	Context    string        `json:"context,omitempty"` // Additional explanation
}

type SeverityLevel string

const (
	SeverityCritical SeverityLevel = "Critical"
	SeverityWarning  SeverityLevel = "Warning"
	SeverityInfo     SeverityLevel = "Info"
)

// Evidence provides structured data linking a diagnosis to a resource.
type Evidence struct {
	Kind   string   `json:"kind"`           // e.g., Pod, Dataset
	Name   string   `json:"name"`           // e.g., runtime-master-0
	Detail string   `json:"detail"`         // e.g., "ExitCode 137, OOMKilled"
	Logs   []string `json:"logs,omitempty"` // Relevant log snippet
}

// DiagnosticContext is a holder for data passed between pipeline stages or for AI consumption.
type DiagnosticContext struct {
	Graph     *ResourceGraph
	K8sEvents []string          // Simplified events list
	PodLogs   map[string]string // Key: PodName, Value: Tail logs
	Config    map[string]interface{}
}
