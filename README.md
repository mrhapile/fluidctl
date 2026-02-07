# Fluid Introspection & Diagnostic System (`fluid-introspector`)

A production-grade, read-only introspection engine for CNCF Fluid. This library powers `fluidctl` and other diagnostic tools by converting complex Kubernetes runtime states into deterministic, typed diagnostic reports.

## Installation & Usage

### Building from Source

To build and install `fluidctl`, use the provided `Makefile`. This ensures that version metadata is correctly injected into the binary.

```bash
# Build the binary to bin/fluidctl
make build

# Install to /usr/local/bin/fluidctl
make install
```

### Verification

Once installed, verify the CLI version:

```bash
fluidctl version
# Output:
# fluidctl version: v0.1.0
# git commit: a1b2c3d
# build date: 2026-02-08T10:00:00Z
# go version: go1.22.0
# platform: darwin/arm64
```

## CLI Modes

`fluidctl` supports two modes of operation:

### 1. Mock Mode (Offline)
Safe for demos, CI/CD, and logic verification without a cluster.

```bash
# Run a specific scenario
fluidctl inspect dataset demo-data --mock --scenario partial-ready

# Output as JSON
fluidctl inspect dataset demo-data --mock --scenario missing-runtime -o json
```

**Available Scenarios:** `healthy`, `partial-ready`, `missing-runtime`, `missing-fuse`, `failed-pods`.

### 2. Real Mode (Kubernetes)
Connects to the active Kubernetes cluster using `KUBECONFIG` or in-cluster config.

```bash
# Inspect a real dataset in your current context
fluidctl inspect dataset my-dataset -n default

# Inspect with JSON output for piping to jq
fluidctl inspect dataset my-dataset -o json | jq .isHealthy
```

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
