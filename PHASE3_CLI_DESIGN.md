# Phase 3 - CLI Design (`fluidctl`)

## 1. Objective
Build a thin, production-ready CLI wrapper (`fluidctl`) that orchestrates the `fluid-introspector` engine. It must support both real cluster interaction and a "Mock Mode" for offline demos and CI.

## 2. Command Architecture

The CLI is built using `cobra`.

```text
fluidctl
├── inspect
│   └── dataset <name> [flags]
└── version
```

### Flags for `inspect dataset`
| Flag | Short | Default | Description |
| :--- | :--- | :--- | :--- |
| `--namespace` | `-n` | `default` | Kubernetes namespace of the dataset. |
| `--output` | `-o` | `tree` | Output format: `tree`, `json`, `wide`. |
| `--mock` | | `false` | Enable offline mock mode (bypasses K8s). |
| `--scenario` | | `healthy` | (Mock only) Scenario to simulate: `healthy`, `partial-ready`, `missing-runtime`, `missing-fuse`. |

## 3. Execution Pipeline

### Real Mode
```mermaid
graph LR
    Args[CLI Args] --> Client[K8s Client]
    Client --> Mapper[mapper.MapDataset]
    Mapper --> Graph[ResourceGraph]
    Graph --> Diagnoser[diagnose.Diagnose]
    Diagnoser --> Result[DiagnosticResult]
    Result --> Printer[Printer (Tree/JSON)]
```

### Mock Mode
```mermaid
graph LR
    Args[CLI Args] --> Scenario[scenarios.GetMockGraph]
    Scenario --> Graph[ResourceGraph]
    Graph --> Diagnoser[diagnose.Diagnose]
    Diagnoser --> Result[DiagnosticResult]
    Result --> Printer[Printer (Tree/JSON)]
```

## 4. Implementation Details

### 4.1. Mock Scenarios (`pkg/scenarios`)
We will create a new package `pkg/scenarios` to host the predefined `ResourceGraph` generators. This keeps test data out of the core `mapper` logic but makes it available to the CLI and tests.

**Scenarios:**
1.  **Healthy**: All components `Ready == Replicas`. Status Bound.
2.  **Missing Runtime**: `Runtime == nil`.
3.  **Partial Ready**: Worker `Ready < Replicas`.
4.  **Missing Fuse**: Fuse `Ready == 0`.

### 4.2. Output Printers (`pkg/printer`)
To ensure clean code, printing logic is isolated.

-   **TreePrinter**: Uses indentation and emoji (`✓`, `⚠`, `❌`). Colorized (Green/Yellow/Red).
-   **JSONPrinter**: Standard `json.MarshalIndent`.

### 4.3. Directory Structure
```text
fluidctl/
├── cmd/
│   ├── root.go       # Root command
│   └── inspect.go    # Inspect command logic
├── pkg/
│   ├── scenarios/    # Mock graph providers
│   └── printer/      # Output formatting
```

## 5. Deliverables
-   `fluidctl` binary.
-   `pkg/scenarios` implementation.
-   `pkg/printer` implementation.
-   Updated README with usage.
