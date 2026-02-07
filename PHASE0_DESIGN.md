# Fluid Inspection & Diagnostic System - Phase 0 Design

## 1. Problem Definition

### The Gap in Operations
Current Fluid operations rely heavily on `kubectl get` and `kubectl logs`. While effective for standard Kubernetes resources, this approach is insufficient for Fluid because:
- **Abstraction Layer**: Fluid's `Dataset` and `Runtime` CRDs abstract away complex underlying infrastructure (StatefulSets, DaemonSets, PVCs, PVs, Fuse pods, Workers, etc.). A simple `kubectl get dataset` often hides the root cause of failures deep within these generated resources.
- **Distributed Complexity**: A single `Dataset` may involve a Master component, multiple Workers, and Fuse components distributed across nodes. Failures can occur in any of these layers (e.g., Fuse failure on a specific node, Worker OOM, Master scheduling issues).
- **Correlation Difficulty**: Manually correlating a `Dataset` status to a specific `Pod` failure in a `StatefulSet` or a `PersistentVolumeClaim` binding issue requires deep knowledge of Fluid's internal naming conventions and owner references, which is time-consuming and error-prone during incidents.

### Dataset vs Runtime Mental Model Gap
- **User Perspective**: Users think in terms of "Datasets" (logical data volumes). They expect data to be ready and accessible.
- **System Reality**: The system operates in terms of "Runtimes" (Alluxio, Jindo, JuiceFS) which are complex distributed systems (Master/Worker architecture) backed by Kubernetes primitives.
- **The Friction**: When a Dataset is "Pending", the user needs to know *why* in terms of the Runtime's health (e.g., "Worker 0 is CrashLoopBackOff due to OOM"), not just that the Dataset CRD is not ready.

## 2. CRD Relationship Analysis

The introspection system rests on understanding the ownership and reference hierarchy:

```mermaid
graph TD
    Dataset[Dataset CRD] -->|Ref| Runtime[Runtime CRD (Alluxio/Jindo/JuiceFS)]
    Runtime -->|Owns| Master[StatefulSet (Master)]
    Runtime -->|Owns| Worker[StatefulSet/DaemonSet (Worker)]
    Runtime -->|Owns| Fuse[DaemonSet (Fuse)]
    Runtime -->|Owns| Config[ConfigMaps/Secrets]
    
    Master -->|Owns| MasterPod[Master Pods]
    Worker -->|Owns| WorkerPod[Worker Pods]
    Fuse -->|Owns| FusePod[Fuse Pods]
    
    Dataset -->|Bound| PVC[PersistentVolumeClaim]
    PVC -->|Bound| PV[PersistentVolume]
```

### Resource Mapping Rules

The `fluid-introspector` must implement specific logic to map high-level resources to low-level K8s objects for each supported Runtime (Alluxio, Jindo, JuiceFS).

**Runtime Components:**
- **Master**: Typically a `StatefulSet`. Critical for metadata management.
- **Worker**: Can be a `StatefulSet` or `DaemonSet` depending on configuration. Responsible for data storage/caching.
- **Fuse**: A `DaemonSet`. Responsible for exposing the filesystem to client pods.

**Storage & Config:**
- **Storage**: `PersistentVolumeClaim` (PVC) and `PersistentVolume` (PV) are created to expose the dataset.
- **Configuration**: `ConfigMaps` and `Secrets` hold initialization scripts and credentials.

## 3. Failure Scenarios

The diagnostic engine must recognize and classify these common failure modes:

| Scenario | Symptom | probable Cause / Diagnostic Check |
| :--- | :--- | :--- |
| **Dataset Pending** | `Dataset` phase is `Bound` but not `Ready` | Master pods are not ready; PVC is not bound; Runtime controller is stalled. |
| **Runtime PartialReady** | Runtime is `Ready` but some replicas are down | Check Worker/Fuse DaemonSet counts. Identify nodes with failures. |
| **Missing Fuse** | Application pods cannot mount volume | Fuse DaemonSet not scheduled on application node (taints/tolerations issue) or Fuse pod crashed. |
| **Worker OOM** | Performance drop or partial data unavailability | Check `RESTARTS` count on Worker pods. Inspect `LastTerminationState` for `OOMKilled`. |
| **PVC Bound but Pod Pending** | App pod pending with "MountVolume.SetUp failed" | Check if Fuse pod is ready on the node where App pod is scheduled. |

## 4. Architecture & Output Model

### Output Hierarchy (Graph-Style)
The output should be a structured, deterministic tree that represents the logical hierarchy, not just a flat list of resources.

**Example Conceptual Output:**
```text
Dataset: demo-data (Namespace: default)
├── Status: Bound (Phase: NotReady)
├── Runtime: AlluxioRuntime/demo-data
│   ├── Master: 1/1 Ready
│   │   └── Pod: demo-data-master-0 (Running)
│   ├── Worker: 2/3 Ready [WARNING]
│   │   ├── Pod: demo-data-worker-0 (Running)
│   │   ├── Pod: demo-data-worker-1 (Running)
│   │   └── Pod: demo-data-worker-2 (CrashLoopBackOff) -> [ERROR] OOMKilled
│   └── Fuse: 5/5 Ready
└── Infrastructure
    ├── PVC: Bound
    └── PV: Bound
```

### Deterministic Ordering
- Maps and lists in the output structure must be sorted (e.g., by name, by kind) to ensure that two runs of `fluidctl` on the same state produce identical output (essential for diffing and testing).

### Machine-Readable JSON
Every command must support `--output json`. The JSON structure should mirror the internal Go structs of the `fluid-introspector` core, ensuring that external tools (and future AI agents) can parse the full state without scraping text.

## 5. Design Principles

1.  **Read-Only & Safe**: The `fluid-introspector` library and `fluidctl` must strictly be read-only. They should never mutate the cluster state (no Create/Update/Delete operations).
2.  **No Side Effects**: Executing a diagnostic command should not trigger reconciliations or leave artifacts in the cluster.
3.  **Deterministic Output**: The same cluster state must always yield the same diagnostic report structure and hash.
4.  **No `kubectl` Shelling**: All K8s interactions must be done via `client-go` or `controller-runtime`. Shelling out to `kubectl` is forbidden to ensure portability (e.g., executing inside a scratch container or a constrained environment).
5.  **Offline-First (Mock Mode)**: The design must prioritize a "Mock Provider" implementation to allow development, testing, and demos without a live cluster.

## 6. Implementation Plan

### Phase 1: Core Resource Mapper
- **Package Layout**: `pkg/mapper`, `pkg/types`, `pkg/k8s`.
- **Core API**: `MapDataset(ctx, name, ns)` returning a `ResourceGraph`.
- **Scope**: Discovery and correlation of Dataset, Runtime, PVC, PV, StatefulSets, DaemonSets, ConfigMaps.

### Phase 2: Diagnostic Engine
- **Pipeline**: Snapshot -> Health Check -> Event Correlation -> Log Sampling -> Failure Analysis.
- **Result**: `DiagnosticResult` with `FailureHint`s.

### Phase 3: Mock Mode
- **Component**: `MockProvider` implementation of the K8s client interface.
- **Usage**: Enables `fluidctl diagnose --mock`.

### Phase 4: CLI (fluidctl)
- **Framework**: `spf13/cobra`.
- **Commands**: `inspect`, `diagnose`.
- **Features**: Colorized output, JSON support, Archive generation.

## 7. AI Readiness (Future Proofing)
- **Context Struct**: A dedicated `DiagnosticContext` struct will aggregate all findings, logs, and specs.
- **Sanitization**: Secrets and sensitive data must be redacted before being serialized for any potential external analysis.
- **Explicit Evidence**: The data model will include an "Evidence" field for every diagnosis, linking a high-level conclusion (e.g., "Worker OOM") to low-level proof (e.g., "Pod X, Container Y, ExitCode 137").
