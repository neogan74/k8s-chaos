# Architecture Overview

Comprehensive architectural documentation for k8s-chaos.

## Table of Contents

- [Introduction](#introduction)
- [System Architecture](#system-architecture)
- [Core Components](#core-components)
- [Data Flow](#data-flow)
- [Custom Resource Definitions](#custom-resource-definitions)
- [Controller Pattern](#controller-pattern)
- [Chaos Actions](#chaos-actions)
- [Safety Architecture](#safety-architecture)
- [Observability Architecture](#observability-architecture)
- [Security Architecture](#security-architecture)
- [Deployment Architecture](#deployment-architecture)
- [Architecture Decision Records](#architecture-decision-records)
- [Design Principles](#design-principles)

## Introduction

k8s-chaos is a Kubernetes-native chaos engineering operator built using the [Kubebuilder](https://kubebuilder.io/) framework. It follows Kubernetes controller patterns and leverages Custom Resource Definitions (CRDs) to provide declarative chaos experiments.

### Design Philosophy

- **Safety First**: Multiple layers of validation and protection
- **Kubernetes-Native**: Follows K8s patterns and conventions
- **Lightweight**: Minimal dependencies, efficient resource usage
- **Observable**: Comprehensive metrics and audit logging
- **Extensible**: Easy to add new chaos actions

### Technology Stack

- **Go 1.24.5+**: Primary programming language
- **Kubebuilder v4**: Scaffolding and controller framework
- **controller-runtime**: Kubernetes controller library
- **client-go**: Kubernetes API client
- **Prometheus**: Metrics collection
- **OpenAPI v3**: CRD validation schema

## System Architecture

### High-Level Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                        │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                  k8s-chaos Operator                   │   │
│  │                                                        │   │
│  │  ┌──────────────┐    ┌──────────────┐               │   │
│  │  │              │    │              │               │   │
│  │  │  Controller  │◄───┤  Admission   │               │   │
│  │  │   Manager    │    │   Webhook    │               │   │
│  │  │              │    │              │               │   │
│  │  └──────┬───────┘    └──────────────┘               │   │
│  │         │                                            │   │
│  │         │ Reconcile Loop                             │   │
│  │         ▼                                            │   │
│  │  ┌──────────────────────────────────┐               │   │
│  │  │    Chaos Action Executors        │               │   │
│  │  │  • PodKill  • PodDelay           │               │   │
│  │  │  • PodCPU   • PodMemory          │               │   │
│  │  │  • PodFail  • NodeDrain          │               │   │
│  │  │  • NetLoss  • DiskFill           │               │   │
│  │  └──────────────┬───────────────────┘               │   │
│  │                 │                                    │   │
│  │                 ▼                                    │   │
│  │  ┌──────────────────────────────────┐               │   │
│  │  │      Metrics Exporter            │               │   │
│  │  │      (Prometheus)                │               │   │
│  │  └──────────────────────────────────┘               │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐   │
│  │           Custom Resources (CRDs)                     │   │
│  │                                                        │   │
│  │  ┌──────────────────┐    ┌──────────────────┐       │   │
│  │  │ ChaosExperiment  │    │ ExperimentHistory│       │   │
│  │  │                  │    │                  │       │   │
│  │  │ • Spec           │    │ • Audit Trail    │       │   │
│  │  │ • Status         │    │ • Execution Data │       │   │
│  │  └──────────────────┘    └──────────────────┘       │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Target Resources                         │   │
│  │                                                        │   │
│  │  ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐            │   │
│  │  │ Pod  │  │ Pod  │  │ Pod  │  │ Node │            │   │
│  │  └──────┘  └──────┘  └──────┘  └──────┘            │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘

         │                      │                    │
         ▼                      ▼                    ▼
   ┌──────────┐          ┌──────────┐         ┌──────────┐
   │Prometheus│          │ Grafana  │         │ kubectl  │
   │ (Metrics)│          │(Dashboard)│        │  (CLI)   │
   └──────────┘          └──────────┘         └──────────┘
```

## Core Components

### 1. Controller Manager

**Location**: `cmd/main.go`, `internal/controller/chaosexperiment_controller.go`

The controller manager is the heart of k8s-chaos. It:

- Watches ChaosExperiment resources
- Implements the reconciliation loop
- Coordinates chaos action execution
- Updates experiment status
- Manages retries and scheduling
- Emits metrics and events

**Key Responsibilities**:
- Resource watching and caching
- Reconciliation logic
- Error handling and retry
- Status updates
- Leader election (HA)

### 2. Admission Webhook

**Location**: `api/v1alpha1/chaosexperiment_webhook.go`

The validating webhook provides pre-creation/update validation:

- **Namespace Validation**: Ensures target namespace exists
- **Selector Validation**: Verifies selector matches pods
- **Cross-Field Validation**: Checks required field combinations
- **Safety Validation**: Enforces production protection, percentage limits
- **Schedule Validation**: Validates cron expressions

**Webhook Flow**:
```
kubectl apply → API Server → Webhook → Validation → Accept/Reject
```

### 3. Custom Resource Definitions (CRDs)

**Location**: `api/v1alpha1/`

#### ChaosExperiment CRD

Primary resource for defining chaos experiments.

```go
type ChaosExperimentSpec struct {
    Action            string            // Chaos action type
    Namespace         string            // Target namespace
    Selector          map[string]string // Pod label selector
    Count             *int32            // Number of resources to affect
    Duration          *string           // Action duration
    Schedule          *string           // Cron schedule
    ExperimentDuration *string          // Total experiment runtime
    // Safety features
    DryRun           bool              // Preview mode
    MaxPercentage    *int32            // Percentage limit
    AllowProduction  bool              // Production approval
    // Retry configuration
    MaxRetries       *int32            // Max retry attempts
    RetryBackoff     *string           // Backoff strategy
    RetryDelay       *string           // Initial retry delay
    // Action-specific
    CPULoad          *int32            // CPU stress percentage
    CPUWorkers       *int32            // CPU stress workers
    MemorySize       *string           // Memory stress size
    // ... other fields
}

type ChaosExperimentStatus struct {
    Phase              string    // Current phase
    Message            string    // Human-readable status
    LastRunTime        Time      // Last execution
    LastScheduledTime  Time      // Last schedule trigger
    NextScheduledTime  Time      // Next scheduled run
    NextRetryTime      Time      // Next retry attempt
    RetryCount         int32     // Current retry count
    LastError          string    // Last error message
    AffectedResources  []string  // Impacted resources
}
```

#### ChaosExperimentHistory CRD

**Location**: `api/v1alpha1/chaosexperimenthistory_types.go`

Records experiment execution history for audit and compliance.

```go
type ChaosExperimentHistory struct {
    Spec   HistorySpec   // Experiment snapshot
    Status HistoryStatus // Execution results
}
```

See [docs/HISTORY.md](HISTORY.md) for details.

### 4. Chaos Action Executors

**Location**: `internal/controller/chaosexperiment_controller.go`

Each chaos action has dedicated execution logic:

| Action | Function | Description |
|--------|----------|-------------|
| pod-kill | `executePodKillAction()` | Delete pods |
| pod-delay | `executePodDelayAction()` | Network latency injection |
| pod-cpu-stress | `executePodCPUStressAction()` | CPU resource stress |
| pod-memory-stress | `executePodMemoryStressAction()` | Memory stress |
| pod-failure | `executePodFailureAction()` | Kill main process |
| node-drain | `executeNodeDrainAction()` | Drain nodes |
| pod-network-loss | `executePodNetworkLossAction()` | Packet loss injection |
| pod-disk-fill | `executePodDiskFillAction()` | Fill disk space |

See [Chaos Actions](#chaos-actions) section for details.

### 5. Metrics Exporter

**Location**: `internal/metrics/metrics.go`

Prometheus metrics for observability:

- `chaos_experiments_total` - Total experiments executed
- `chaos_experiments_duration_seconds` - Execution duration
- `chaos_experiments_resources_affected_total` - Resources affected
- `chaos_experiments_errors_total` - Error counts
- `chaos_experiments_active` - Currently active experiments
- `chaos_experiments_history_*` - History-related metrics

See [docs/METRICS.md](METRICS.md) for complete list.

## Data Flow

### Experiment Lifecycle

```
┌─────────────────────────────────────────────────────────────┐
│                   Experiment Creation                        │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│  1. User creates ChaosExperiment resource (kubectl/GitOps)  │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│  2. Admission Webhook validates resource                     │
│     • Check namespace exists                                 │
│     • Validate selector matches pods                         │
│     • Enforce safety constraints                             │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ├─ Rejected ──► Error returned to user
                      │
                      ├─ Accepted
                      ▼
┌─────────────────────────────────────────────────────────────┐
│  3. Resource stored in etcd                                  │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│  4. Controller receives watch event                          │
│     • Resource added to work queue                           │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│  5. Reconcile() function invoked                             │
│     • Fetch ChaosExperiment from cache                       │
│     • Determine if execution needed                          │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│  6. Pre-execution checks                                     │
│     • Check schedule (if configured)                         │
│     • Check experiment duration                              │
│     • Apply safety filters                                   │
│     • Select eligible resources                              │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│  7. Execute chaos action                                     │
│     • DryRun mode: List affected resources                   │
│     • Normal mode: Apply chaos to resources                  │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ├─ Success ──► Update status, create history
                      │
                      ├─ Failure ──► Check retry configuration
                      ▼
┌─────────────────────────────────────────────────────────────┐
│  8. Update status                                            │
│     • Set phase (Running/Completed/Failed)                   │
│     • Update timestamps                                      │
│     • Record affected resources                              │
│     • Emit metrics                                           │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│  9. Determine next action                                    │
│     • Schedule-based: Requeue for next cron time             │
│     • Retry-needed: Requeue with backoff                     │
│     • Duration-based: Requeue for cleanup check              │
│     • Default: Requeue after 1 minute                        │
└─────────────────────────────────────────────────────────────┘
```

### Reconciliation Loop

```go
func (r *ChaosExperimentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Fetch experiment
    exp := &v1alpha1.ChaosExperiment{}
    err := r.Get(ctx, req.NamespacedName, exp)

    // 2. Check if deletion requested
    if !exp.DeletionTimestamp.IsZero() {
        return r.handleFinalization(ctx, exp)
    }

    // 3. Check schedule
    if shouldExecute, requeue := r.checkSchedule(exp); !shouldExecute {
        return requeue, nil
    }

    // 4. Check experiment duration
    if r.isExperimentExpired(exp) {
        return r.markExperimentCompleted(ctx, exp)
    }

    // 5. Select eligible resources
    resources, err := r.selectEligibleResources(ctx, exp)

    // 6. Execute chaos action
    switch exp.Spec.Action {
    case "pod-kill":
        err = r.executePodKillAction(ctx, exp, resources)
    case "pod-delay":
        err = r.executePodDelayAction(ctx, exp, resources)
    // ... other actions
    }

    // 7. Handle result
    if err != nil {
        return r.handleExecutionError(ctx, exp, err)
    }

    // 8. Create history record
    r.createHistoryRecord(ctx, exp, resources)

    // 9. Update status and requeue
    exp.Status.Phase = "Running"
    r.Status().Update(ctx, exp)

    return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
}
```

## Custom Resource Definitions

### API Group

- **Group**: `chaos.gushchin.dev`
- **Version**: `v1alpha1`
- **Kind**: `ChaosExperiment`, `ChaosExperimentHistory`

### Validation Layers

k8s-chaos implements multi-layer validation (see [ADR-0001](adr/0001-crd-validation-strategy.md)):

**Layer 1: OpenAPI Schema Validation**
- Type checking (string, int, etc.)
- Enum validation for actions
- Range validation (e.g., count: 1-100)
- Pattern validation (e.g., duration regex)
- Required fields

**Layer 2: Admission Webhook**
- Cross-field validation
- Dynamic validation (namespace exists, pods match selector)
- Safety constraint enforcement
- Business logic validation

## Controller Pattern

### Standard Kubernetes Controller

k8s-chaos follows the standard Kubernetes controller pattern:

```
Watch ──► Cache ──► WorkQueue ──► Reconcile ──► Update ──► Requeue
  │                                   │
  │                                   ▼
  │                              Execute Action
  │                                   │
  └─────────────────────────────────┘
```

### Reconciliation Strategy

- **Level-driven**: Not event-driven; relies on desired state
- **Idempotent**: Safe to run multiple times
- **Requeue**: Continuously reconciles (default: 1 minute)
- **Error handling**: Exponential backoff on errors

### Leader Election

For HA deployments:
- Only one controller active at a time
- Automatic failover if leader crashes
- Leader lease in ConfigMap

## Chaos Actions

### Implementation Pattern

Each chaos action follows a consistent pattern:

```go
func (r *ChaosExperimentReconciler) executeXxxAction(
    ctx context.Context,
    exp *v1alpha1.ChaosExperiment,
    resources []resource,
) error {
    // 1. Dry-run check
    if exp.Spec.DryRun {
        return r.recordDryRunResults(exp, resources)
    }

    // 2. Apply safety filters
    filtered := r.applySafetyFilters(resources, exp)

    // 3. Limit by count/percentage
    selected := r.limitResources(filtered, exp)

    // 4. Execute chaos
    for _, res := range selected {
        if err := r.applyChaosToresource(res); err != nil {
            return err
        }
    }

    // 5. Track affected resources
    exp.Status.AffectedResources = getResourceNames(selected)

    return nil
}
```

### Action Details

See individual ADRs for implementation details:
- [ADR-0002](adr/0002-safety-features-implementation.md) - Safety features
- [ADR-0003](adr/0003-pod-memory-stress-implementation.md) - Pod memory stress
- [ADR-0004](adr/0004-pod-failure-implementation.md) - Pod failure
- [ADR-0005](adr/0005-pod-cpu-stress-implementation.md) - Pod CPU stress
- [ADR-0007](adr/0007-pod-network-loss-implementation.md) - Pod network loss
- [ADR-0008](adr/0008-pod-disk-fill-implementation.md) - Pod disk fill

## Safety Architecture

### Multi-Layer Safety

```
┌─────────────────────────────────────────────────────────┐
│ Layer 1: OpenAPI Validation (CRD Schema)                │
│  • Type checking                                         │
│  • Range limits (count: 1-100, cpuLoad: 1-100)          │
│  • Required fields                                       │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│ Layer 2: Admission Webhook                              │
│  • Namespace existence                                   │
│  • Selector effectiveness                                │
│  • Production protection                                 │
│  • Percentage limits                                     │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│ Layer 3: Controller Runtime Checks                      │
│  • Exclusion labels                                      │
│  • Dry-run mode                                          │
│  • Maximum percentage enforcement                        │
│  • Resource availability                                 │
└─────────────────────────────────────────────────────────┘
```

### Safety Features

See [ADR-0002](adr/0002-safety-features-implementation.md) for detailed design:

- **Dry-Run Mode**: Preview impact without execution
- **Max Percentage**: Limit resources affected (e.g., max 30%)
- **Production Protection**: Require explicit approval for prod namespaces
- **Exclusion Labels**: Protect critical resources

## Observability Architecture

### Metrics Pipeline

```
Controller ──► Prometheus Metrics ──► Grafana Dashboards
                      │
                      ├─► experiments_total
                      ├─► experiments_duration_seconds
                      ├─► resources_affected_total
                      ├─► experiments_errors_total
                      ├─► experiments_active
                      └─► safety metrics
```

### History Architecture

See [ADR-0006](adr/0006-experiment-history-and-audit-logging.md):

```
Experiment Execution ──► Create History Record
                              │
                              ├─► Store in ChaosExperimentHistory CRD
                              ├─► Label for querying
                              └─► Auto-cleanup (retention limit)
```

### Dashboard Hierarchy

1. **Overview Dashboard**: Executive summary
2. **Detailed Dashboard**: Deep-dive analysis
3. **Safety Dashboard**: Error and impact monitoring

## Security Architecture

### RBAC Model

```
┌─────────────────────────────────────────────────────────┐
│ Controller ServiceAccount                                │
│  k8s-chaos-controller-manager                            │
└─────────────────┬───────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────┐
│ ClusterRole: k8s-chaos-manager-role                      │
│  • get/list/watch/update ChaosExperiment resources       │
│  • get/list/watch/update ChaosExperimentHistory          │
│  • get/list/delete/patch Pods                            │
│  • create pods/exec, pods/eviction                       │
│  • get/list/watch/update/patch Nodes                     │
│  • get/list Namespaces                                   │
│  • update pods/ephemeralcontainers                       │
└─────────────────────────────────────────────────────────┘
```

### Pod Security

- **runAsNonRoot**: true
- **runAsUser**: 65532 (non-root)
- **fsGroup**: 65532
- **allowPrivilegeEscalation**: false
- **readOnlyRootFilesystem**: true
- **capabilities**: DROP ALL

### Network Policies

Recommended network policies:

```yaml
# Restrict controller egress
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: k8s-chaos-controller
spec:
  podSelector:
    matchLabels:
      control-plane: controller-manager
  policyTypes:
  - Egress
  egress:
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 443  # K8s API
```

## Deployment Architecture

### Standard Deployment

```
┌─────────────────────────────────────────────────────────┐
│ Namespace: k8s-chaos-system                              │
│                                                           │
│  ┌────────────────────────────────────────────┐         │
│  │ Deployment: controller-manager             │         │
│  │  replicas: 1                                │         │
│  │  ┌──────────────────────────────────────┐  │         │
│  │  │ Container: manager                   │  │         │
│  │  │  image: ghcr.io/neogan74/k8s-chaos  │  │         │
│  │  │  resources:                          │  │         │
│  │  │    requests: 100m CPU, 128Mi mem     │  │         │
│  │  │    limits: 500m CPU, 512Mi mem       │  │         │
│  │  └──────────────────────────────────────┘  │         │
│  └────────────────────────────────────────────┘         │
│                                                           │
│  ┌────────────────────────────────────────────┐         │
│  │ Service: metrics-service                   │         │
│  │  port: 8443 (HTTPS)                        │         │
│  └────────────────────────────────────────────┘         │
│                                                           │
│  ┌────────────────────────────────────────────┐         │
│  │ ValidatingWebhookConfiguration             │         │
│  │  webhooks:                                 │         │
│  │  - chaosexperiments.kb.io                  │         │
│  └────────────────────────────────────────────┘         │
└─────────────────────────────────────────────────────────┘
```

### GitOps Integration

Supports:
- **ArgoCD**: Application + Kustomization
- **Flux**: HelmRelease + HelmRepository
- **Kustomize**: Base + overlays

See [deploy/](../deploy/) directory.

## Architecture Decision Records

Key architectural decisions documented in ADRs:

| ADR | Decision | Impact |
|-----|----------|--------|
| [0001](adr/0001-crd-validation-strategy.md) | Multi-layer validation | Robust input validation |
| [0002](adr/0002-safety-features-implementation.md) | Comprehensive safety features | Production-ready safety |
| [0003](adr/0003-pod-memory-stress-implementation.md) | Memory stress via ephemeral containers | Flexible stress testing |
| [0004](adr/0004-pod-failure-implementation.md) | Process killing for failures | Realistic crash testing |
| [0005](adr/0005-pod-cpu-stress-implementation.md) | CPU stress with stress-ng | Effective CPU testing |
| [0006](adr/0006-experiment-history-and-audit-logging.md) | CRD-based history | Audit compliance |
| [0007](adr/0007-pod-network-loss-implementation.md) | tc-based packet loss | Network chaos |
| [0008](adr/0008-pod-disk-fill-implementation.md) | Ephemeral storage filling | Disk chaos |

See [docs/adr/README.md](adr/README.md) for complete list.

## Design Principles

### 1. Kubernetes-Native

- Use CRDs for declarative API
- Follow controller pattern
- Leverage built-in RBAC
- Support standard tools (kubectl, Helm, Kustomize)

### 2. Safety-Focused

- Multiple validation layers
- Dry-run mode for testing
- Production protections
- Exclusion mechanisms
- Percentage limits

### 3. Observable

- Comprehensive metrics
- Structured logging
- Audit history
- Status tracking
- Event emission

### 4. Extensible

- Pluggable action architecture
- Easy to add new chaos types
- Configurable via CRD spec
- Support for custom logic

### 5. Lightweight

- Single binary deployment
- Minimal dependencies
- Efficient resource usage
- No external databases required

### 6. Developer-Friendly

- Clear error messages
- Comprehensive documentation
- Example configurations
- Local development support
- Testing utilities

## Future Architecture Considerations

### Scalability

- **Multi-tenancy**: Namespace-scoped operators
- **Rate limiting**: Limit experiment execution rate
- **Batching**: Execute multiple experiments efficiently

### Advanced Features

- **Experiment orchestration**: Chain experiments
- **Time windows**: Maintenance windows for experiments
- **Service mesh integration**: Istio/Linkerd fault injection
- **Event-driven**: Trigger experiments on events

### Extensibility

- **Plugin system**: External chaos providers
- **Custom actions**: User-defined chaos types
- **Webhooks**: Notifications to external systems

## Related Documentation

- [Installation Guide](INSTALLATION.md) - Deployment instructions
- [Getting Started](GETTING-STARTED.md) - First experiment tutorial
- [API Reference](API.md) - CRD specification
- [Development Guide](DEVELOPMENT.md) - Contributor setup
- [Metrics Guide](METRICS.md) - Observability details
- [ADR Directory](adr/README.md) - Design decisions

## Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for architectural contribution guidelines.

When proposing architectural changes:
1. Create an ADR documenting the decision
2. Update this architecture document
3. Ensure changes align with design principles
4. Consider backward compatibility