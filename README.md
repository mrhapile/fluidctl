# Fluid Introspection & Diagnostic System (`fluid-introspector`)

A production-grade, read-only introspection engine for CNCF Fluid. This library powers `fluidctl` and other diagnostic tools by converting complex Kubernetes runtime states into deterministic, typed diagnostic reports.

## Architecture

The system operates in two phases:
1.  **Phase 1: Resource Mapper (`pkg/mapper`)**: Discovers and correlates Fluid Datasets with their underlying Kubernetes resources (StatefulSets, PVCs, Pods, ConfigMaps) into a `ResourceGraph`.
2.  **Phase 2: Diagnostic Engine (`pkg/diagnose`)**: Analyzes the `ResourceGraph` using a set of static, deterministic rules to identify failures and suggest remediations.

## How Diagnostics Work

The Diagnostic Engine is a pure function that takes a `ResourceGraph` and returns a `DiagnosticResult`. It performs no external API calls, ensuring speed and reliability.

### The Pipeline
1.  **Input**: A snapshot of the resource state (`ResourceGraph`).
2.  **Rule Evaluation**: The engine iterates through a fixed list of rules.
3.  **Aggregation**: Failure hints are collected.
4.  **Sorting**: Results are consistently sorted by Severity → Component → RuleID.
5.  **Output**: A JSON-serializable `DiagnosticResult`.

### Failure Rules
| ID | Severity | Description |
| :--- | :--- | :--- |
| `DATASET_NOT_BOUND` | Critical | The Dataset CR exists but is not in a Bound state. |
| `RUNTIME_MISSING` | Critical | No Runtime CR was found for the Dataset. |
| `MASTER_NOT_READY` | Critical | The Runtime Master StatefulSet is not fully ready. |
| `WORKER_PARTIALLY_READY` | Warning | Some Worker pods are not ready (e.g., OOMKilled, CrashLoop). |
| `FUSE_MISSING` | Warning | The Fuse DaemonSet has 0 ready replicas. |
| `PVC_NOT_BOUND` | Critical | The underlying PersistentVolumeClaim is not Bound. |

## Mock-Mode & Example Scenarios

The engine is tested against mock graphs to ensure correct behavior without a live cluster.

### Scenario 1: Partial Failure (`WORKER_PARTIALLY_READY`)

**Context:** A worker node ran out of memory, causing one worker pod to crash.

**Diagnostic Output:**
```json
{
  "timestamp": "2023-10-27T10:00:00Z",
  "isHealthy": false,
  "summary": "Found 1 issues: 0 critical, 1 warnings.",
  "failureHints": [
    {
      "id": "WORKER_PARTIALLY_READY",
      "severity": "Warning",
      "component": "Runtime/Worker",
      "evidence": {
        "kind": "StatefulSet/DaemonSet",
        "name": "demo-data-worker",
        "detail": "Ready replicas: 2/3"
      },
      "suggestion": "Check individual Worker pods for OOMKilled or CrashLoopBackOff."
    }
  ]
}
```

### Scenario 2: Missing Runtime (`RUNTIME_MISSING`)

**Context:** User created a Dataset CR but forgot to apply the corresponding AlluxioRuntime CR.

**Diagnostic Output:**
```json
{
  "timestamp": "2023-10-27T10:05:00Z",
  "isHealthy": false,
  "summary": "Found 1 issues: 1 critical, 0 warnings.",
  "failureHints": [
    {
      "id": "RUNTIME_MISSING",
      "severity": "Critical",
      "component": "Runtime",
      "evidence": {
        "kind": "Runtime",
        "name": "demo-data",
        "detail": "Runtime object is missing from graph."
      },
      "suggestion": "Create a Runtime CR (e.g., AlluxioRuntime, JindoRuntime) matching the Dataset."
    }
  ]
}
```

### Scenario 3: Fuse Failure (`FUSE_MISSING`)

**Context:** Fuse DaemonSet exists but no pods are scheduled (e.g., due to taints).

**Diagnostic Output:**
```json
{
  "timestamp": "2023-10-27T10:10:00Z",
  "isHealthy": false,
  "summary": "Found 1 issues: 0 critical, 1 warnings.",
  "failureHints": [
    {
      "id": "FUSE_MISSING",
      "severity": "Warning",
      "component": "Runtime/Fuse",
      "evidence": {
        "kind": "DaemonSet",
        "name": "demo-data-fuse",
        "detail": "Ready replicas: 0/5"
      },
      "suggestion": "Check DaemonSet node selectors and tolerations. Ensure nodes have capacity."
    }
  ]
}
```
