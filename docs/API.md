# ChaosExperiment API Reference

Complete API documentation for the `ChaosExperiment` Custom Resource Definition (CRD).

## Table of Contents

- [Overview](#overview)
- [API Version](#api-version)
- [Resource Structure](#resource-structure)
- [Spec Fields](#spec-fields)
- [Status Fields](#status-fields)
- [Validation Rules](#validation-rules)
- [Examples](#examples)
- [Best Practices](#best-practices)

## Overview

The `ChaosExperiment` CRD is the primary interface for defining and executing chaos engineering experiments in Kubernetes. It allows you to specify controlled failure scenarios to test system resilience.

**API Group:** `chaos.gushchin.dev`
**API Version:** `v1alpha1`
**Kind:** `ChaosExperiment`
**Plural:** `chaosexperiments`
**Singular:** `chaosexperiment`
**Short Names:** None

## API Version

**Current Version:** `v1alpha1`

The `v1alpha1` version indicates this API is in alpha stage:
- Schema may change without backward compatibility guarantees
- Suitable for testing and development
- Not recommended for production workloads without careful consideration
- Breaking changes may occur in future versions

## Resource Structure

```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: string              # Required: Unique identifier
  namespace: string         # Required: Kubernetes namespace
  labels: map[string]string # Optional: Key-value labels
spec:
  action: string            # Required: Type of chaos action
  namespace: string         # Required: Target namespace
  selector: map[string]string # Required: Pod label selector
  count: int                # Optional: Number of pods to affect (default: 1)
  duration: string          # Optional: Duration for time-based actions
status:
  lastRunTime: timestamp    # Auto-populated: Last execution time
  message: string           # Auto-populated: Human-readable status
  phase: string             # Auto-populated: Current execution phase
```

## Spec Fields

The `spec` section defines the desired state of the chaos experiment.

### action

**Type:** `string`
**Required:** Yes
**Validation:** Must be one of: `pod-kill`, `pod-delay`, `node-drain`, `pod-cpu-stress`, `pod-memory-stress`, `pod-failure`, `pod-network-loss`, `pod-disk-fill`

Specifies the type of chaos action to perform.

#### Supported Actions

| Action | Description | Required Fields |
|--------|-------------|----------------|
| `pod-kill` | Terminates selected pods | action, namespace, selector |
| `pod-delay` | Adds network latency to pods | action, namespace, selector, duration |
| `node-drain` | Drains and cordons nodes | action, namespace, selector |
| `pod-cpu-stress` | Injects CPU stress via ephemeral containers | action, namespace, selector, duration, cpuLoad |
| `pod-memory-stress` | Injects memory stress via ephemeral containers | action, namespace, selector, duration, memorySize |
| `pod-failure` | Kills main process (PID 1) to cause container crash | action, namespace, selector |
| `pod-network-loss` | Injects packet loss using tc netem | action, namespace, selector, duration, lossPercentage |
| `pod-disk-fill` | Fills disk space using an ephemeral container | action, namespace, selector, duration, fillPercentage |

#### Examples

```yaml
# Pod termination
spec:
  action: "pod-kill"
```

```yaml
# Network delay (requires duration)
spec:
  action: "pod-delay"
  duration: "30s"
```

```yaml
# Node drain
spec:
  action: "node-drain"
```

```yaml
# CPU stress (requires duration and cpuLoad)
spec:
  action: "pod-cpu-stress"
  duration: "5m"
  cpuLoad: 80
  cpuWorkers: 2
```

```yaml
# Memory stress (requires duration and memorySize)
spec:
  action: "pod-memory-stress"
  duration: "5m"
  memorySize: "512M"
  memoryWorkers: 2
```

```yaml
# Pod failure (kills main process)
spec:
  action: "pod-failure"
```

```yaml
# Network packet loss (requires duration and lossPercentage)
spec:
  action: "pod-network-loss"
  duration: "2m"
  lossPercentage: 10
  lossCorrelation: 25
```

```yaml
# Disk fill (requires duration and fillPercentage)
spec:
  action: "pod-disk-fill"
  duration: "2m"
  fillPercentage: 80
  targetPath: "/tmp"
```

#### Notes
- Action names are case-sensitive
- Actions using ephemeral containers (cpu-stress, memory-stress, network-loss, disk-fill) require Kubernetes 1.25+
- Network chaos actions require NET_ADMIN capability in the cluster

---

### namespace

**Type:** `string`
**Required:** Yes
**Validation:** Minimum length of 1 character

Specifies the Kubernetes namespace where target resources are located.

#### Examples

```yaml
spec:
  namespace: "default"
```

```yaml
spec:
  namespace: "production"
```

```yaml
spec:
  namespace: "my-app-staging"
```

#### Important Notes
- The namespace must exist before the experiment runs
- Controller must have RBAC permissions in the target namespace
- Cross-namespace targeting is not supported (one experiment = one namespace)
- The experiment resource itself can be in a different namespace than the target

#### Common Patterns

```yaml
# Development testing
spec:
  namespace: "dev"

# Production chaos (use with caution)
spec:
  namespace: "production"

# Isolated testing
spec:
  namespace: "chaos-testing"
```

---

### selector

**Type:** `map[string]string`
**Required:** Yes
**Validation:** Must contain at least one key-value pair

Label selector used to identify target pods. All labels specified must match (AND logic).

#### Examples

**Single label:**
```yaml
spec:
  selector:
    app: nginx
```

**Multiple labels (AND condition):**
```yaml
spec:
  selector:
    app: web-server
    tier: frontend
    version: v2
```

**StatefulSet pods:**
```yaml
spec:
  selector:
    app: postgresql
    statefulset.kubernetes.io/pod-name: postgresql-0
```

**Complex selectors:**
```yaml
spec:
  selector:
    app: api-service
    environment: staging
    team: platform
```

#### Selector Matching Behavior

The selector uses **exact match** for all labels:
- All specified labels must be present on the pod
- All label values must match exactly
- If a pod has additional labels not in the selector, it still matches

**Example:**

```yaml
# Selector
selector:
  app: nginx
  env: prod

# This pod WILL match
labels:
  app: nginx
  env: prod
  version: v1.2.3  # Extra label is OK

# This pod will NOT match
labels:
  app: nginx  # Missing 'env' label

# This pod will NOT match
labels:
  app: nginx
  env: staging  # Wrong value for 'env'
```

#### Finding Matching Pods

Before creating an experiment, verify your selector:

```bash
# List pods matching your selector
kubectl get pods -n <namespace> -l app=nginx,tier=frontend

# Show detailed pod labels
kubectl get pods -n <namespace> --show-labels

# Count matching pods
kubectl get pods -n <namespace> -l app=nginx --no-headers | wc -l
```

#### Best Practices

1. **Be specific**: Use multiple labels to avoid unintended targeting
2. **Test first**: Verify selector matches before running experiment
3. **Use standard labels**: Follow [Kubernetes recommended labels](https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/)
4. **Avoid broad selectors**: Don't use overly generic labels like `app: backend`

---

### count

**Type:** `integer`
**Required:** No
**Default:** `1`
**Validation:**
- Minimum: `1`
- Maximum: `100`

Number of pods to affect with the chaos action.

#### Examples

```yaml
# Kill one pod (default)
spec:
  count: 1
```

```yaml
# Kill multiple pods
spec:
  count: 3
```

```yaml
# Test large-scale failure
spec:
  count: 10
```

```yaml
# Can omit for default behavior
spec:
  # count defaults to 1
  action: "pod-kill"
```

#### Behavior

- If `count` exceeds the number of matching pods, **all** matching pods are affected
- Pods are selected **randomly** from the matching set
- Selection happens at each reconciliation loop (approximately every minute)
- The same pods may be selected multiple times in consecutive runs

**Example:**
```yaml
spec:
  selector:
    app: nginx
  count: 5

# If only 3 pods match:
# - All 3 pods will be killed
# - No error is raised
```

#### Safe Limits

To prevent accidental large-scale chaos:
- Maximum value is enforced at `100`
- Consider using percentage-based limits (future feature)
- Start with small values and increase gradually

**Recommended progression:**
```yaml
# Phase 1: Single pod
count: 1

# Phase 2: Small subset
count: 2-3

# Phase 3: Larger scale
count: 5-10

# Phase 4: Production testing
count: <30% of total pods>
```

#### Edge Cases

```yaml
# No matching pods
# Result: Experiment runs but affects nothing

# count = 0 (invalid)
# Result: Validation error, resource rejected

# count > 100 (invalid)
# Result: Validation error, resource rejected

# count = matching pods
# Result: All pods affected exactly once per cycle
```

---

### duration

**Type:** `string`
**Required:** No (required for `pod-delay` action)
**Validation:** Must match pattern `^([0-9]+(s|m|h))+$`
**Default:** None

Specifies how long the chaos effect should last. Currently used only for `pod-delay` action.

#### Format

Duration string with units:
- `s` - seconds
- `m` - minutes
- `h` - hours

Can combine multiple units: `1h30m` = 1 hour 30 minutes

#### Examples

```yaml
# 30 seconds
spec:
  action: "pod-delay"
  duration: "30s"
```

```yaml
# 5 minutes
spec:
  action: "pod-delay"
  duration: "5m"
```

```yaml
# 1 hour
spec:
  action: "pod-delay"
  duration: "1h"
```

```yaml
# Complex duration
spec:
  action: "pod-delay"
  duration: "2h30m"
```

#### Valid Formats

```yaml
duration: "10s"      # ✅ 10 seconds
duration: "5m"       # ✅ 5 minutes
duration: "1h"       # ✅ 1 hour
duration: "90s"      # ✅ 90 seconds (1.5 minutes)
duration: "1h30m45s" # ✅ Complex duration
```

#### Invalid Formats

```yaml
duration: "10"       # ❌ Missing unit
duration: "10 s"     # ❌ Space not allowed
duration: "10sec"    # ❌ Invalid unit
duration: "1.5h"     # ❌ Decimal not allowed
duration: "-5m"      # ❌ Negative not allowed
```

#### Action-Specific Requirements

| Action | Duration Required? | Behavior |
|--------|-------------------|----------|
| `pod-kill` | No | Ignored if specified |
| `pod-delay` | Yes | Network delay lasts for specified duration |
| `node-drain` | No | Ignored if specified |
| `pod-cpu-stress` | Yes | CPU stress lasts for specified duration |
| `pod-memory-stress` | Yes | Memory stress lasts for specified duration |
| `pod-failure` | No | Ignored (immediate process kill) |
| `pod-network-loss` | Yes | Packet loss lasts for specified duration |

#### Notes
- For `pod-kill` and `pod-failure`, duration is ignored (immediate action)
- Zero duration is not allowed

---

### cpuLoad

**Type:** `integer`
**Required:** Yes (for `pod-cpu-stress` action)
**Default:** None
**Validation:** 1-100

Percentage of CPU to consume during stress testing.

#### Example

```yaml
spec:
  action: "pod-cpu-stress"
  duration: "5m"
  cpuLoad: 80      # 80% CPU load
  cpuWorkers: 2    # 2 CPU workers
```

---

### cpuWorkers

**Type:** `integer`
**Required:** No
**Default:** `1`
**Validation:** 1-32

Number of CPU worker processes for stress testing.

---

### memorySize

**Type:** `string`
**Required:** Yes (for `pod-memory-stress` action)
**Default:** None
**Validation:** Pattern `^[0-9]+[MG]$`

Amount of memory to allocate per worker. Format: number followed by M (megabytes) or G (gigabytes).

#### Examples

```yaml
memorySize: "256M"   # 256 megabytes
memorySize: "1G"     # 1 gigabyte
memorySize: "512M"   # 512 megabytes
```

---

### memoryWorkers

**Type:** `integer`
**Required:** No
**Default:** `1`
**Validation:** 1-8

Number of memory worker processes. Total memory = memorySize × memoryWorkers.

---

### lossPercentage

**Type:** `integer`
**Required:** Yes (for `pod-network-loss` action)
**Default:** `5`
**Validation:** 1-40

Percentage of network packets to drop. Limited to 40% for safety.

#### Example

```yaml
spec:
  action: "pod-network-loss"
  duration: "2m"
  lossPercentage: 10    # Drop 10% of packets
  lossCorrelation: 25   # 25% correlation
```

---

### lossCorrelation

**Type:** `integer`
**Required:** No
**Default:** `0`
**Validation:** 0-100

Correlation percentage for packet loss. Higher values make losses cluster together (burst losses).

- `0` = Independent random losses
- `25` = Some correlation (realistic network issues)
- `50+` = High correlation (simulates network congestion bursts)

#### Example

```yaml
# Simulate bursty packet loss (more realistic)
spec:
  action: "pod-network-loss"
  duration: "5m"
  lossPercentage: 15
  lossCorrelation: 50
```

---

### fillPercentage

**Type:** `integer`
**Required:** Yes (for `pod-disk-fill` action)
**Default:** `80`
**Validation:** 50-95

Percentage of disk space to fill on the target filesystem.

#### Example

```yaml
spec:
  action: "pod-disk-fill"
  duration: "2m"
  fillPercentage: 85
  targetPath: "/tmp"
```

---

### targetPath

**Type:** `string`
**Required:** No
**Default:** `/tmp`

Path inside the pod filesystem to fill. Ignored when `volumeName` is set.

#### Example

```yaml
spec:
  action: "pod-disk-fill"
  duration: "2m"
  fillPercentage: 80
  targetPath: "/var/log"
```

---

### volumeName

**Type:** `string`
**Required:** No

Optional volume name to target. The controller resolves the first matching mount path and uses it for filling disk.

#### Example

```yaml
spec:
  action: "pod-disk-fill"
  duration: "2m"
  fillPercentage: 80
  volumeName: "data"
```

---

## Status Fields

The `status` section is populated automatically by the controller. **Do not set these fields manually.**

### lastRunTime

**Type:** `metav1.Time` (RFC3339 timestamp)
**Set by:** Controller
**Optional:** Yes

Timestamp of when the experiment was last executed.

#### Example

```yaml
status:
  lastRunTime: "2025-10-10T14:30:00Z"
```

#### Usage

```bash
# View last run time
kubectl get chaosexperiment my-experiment -o jsonpath='{.status.lastRunTime}'

# Sort experiments by last run
kubectl get chaosexperiment -o custom-columns=NAME:.metadata.name,LAST_RUN:.status.lastRunTime
```

#### Notes
- Updated on each reconciliation cycle (approximately every minute)
- Uses UTC timezone
- May be empty for newly created experiments that haven't run yet

---

### message

**Type:** `string`
**Set by:** Controller
**Optional:** Yes

Human-readable status message describing the result of the last execution.

#### Example Values

```yaml
status:
  message: "Successfully killed 2 pods"
```

```yaml
status:
  message: "No pods matched selector app=nginx"
```

```yaml
status:
  message: "Error: insufficient permissions to delete pods"
```

#### Common Messages

| Message | Meaning |
|---------|---------|
| `Successfully killed N pods` | Chaos action completed successfully |
| `No pods matched selector` | Selector didn't match any pods |
| `Error: action 'X' not supported` | Invalid action specified |
| `Error: insufficient permissions` | RBAC permissions missing |
| `Namespace not found` | Target namespace doesn't exist |

#### Usage

```bash
# View status message
kubectl get chaosexperiment my-experiment -o jsonpath='{.status.message}'

# Watch status changes
kubectl get chaosexperiment my-experiment -w
```

---

### phase

**Type:** `string`
**Set by:** Controller
**Optional:** Yes
**Validation:** Must be one of: `Pending`, `Running`, `Completed`, `Failed`

Current execution phase of the experiment.

#### Phase Values

| Phase | Description | Transitions To |
|-------|-------------|----------------|
| `Pending` | Experiment created, not yet executed | `Running` |
| `Running` | Currently executing chaos action | `Completed`, `Failed` |
| `Completed` | Successfully executed | `Running` (on next cycle) |
| `Failed` | Execution failed with error | `Running` (on retry) |

#### Examples

```yaml
status:
  phase: "Completed"
  message: "Successfully killed 2 pods"
```

```yaml
status:
  phase: "Failed"
  message: "Error: namespace 'prod' not found"
```

#### Lifecycle

```
Pending → Running → Completed
                  ↘ Failed
                     ↓
                  (retry) → Running
```

#### Usage

```bash
# Filter by phase
kubectl get chaosexperiment -A -o custom-columns=NAME:.metadata.name,PHASE:.status.phase

# Watch phase changes
kubectl get chaosexperiment my-experiment -o jsonpath='{.status.phase}' -w

# Count by phase
kubectl get chaosexperiment -A -o json | jq '[.items[].status.phase] | group_by(.) | map({phase: .[0], count: length})'
```

---

## Validation Rules

All validation is enforced at the API level using OpenAPI schema validation.

### Spec Validation

| Field | Rules | Error if Violated |
|-------|-------|-------------------|
| `action` | Required, must be `pod-kill\|pod-delay\|node-drain` | `Invalid value: "X": spec.action in body should be one of [pod-kill pod-delay node-drain]` |
| `namespace` | Required, min length 1 | `Invalid value: "": spec.namespace in body should be at least 1 chars long` |
| `selector` | Required, min 1 property | `Invalid value: {}: spec.selector in body should have at least 1 properties` |
| `count` | Optional, 1-100 | `Invalid value: 101: spec.count in body should be less than or equal to 100` |
| `duration` | Optional, matches `^([0-9]+(s\|m\|h))+$` | `Invalid value: "10": spec.duration in body should match '^([0-9]+(s\|m\|h))+$'` |

### Testing Validation

```bash
# Valid resource
cat <<EOF | kubectl apply -f -
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: valid-experiment
spec:
  action: "pod-kill"
  namespace: "default"
  selector:
    app: nginx
  count: 5
EOF

# Invalid action
cat <<EOF | kubectl apply -f -
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: invalid-action
spec:
  action: "pod-explode"  # ❌ Not in enum
  namespace: "default"
  selector:
    app: nginx
EOF
# Error: spec.action in body should be one of [pod-kill pod-delay node-drain]
```

---

## Examples

### Basic Pod Kill

Kill a single nginx pod in the default namespace:

```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: nginx-pod-kill
  namespace: chaos-testing
spec:
  action: "pod-kill"
  namespace: "default"
  selector:
    app: nginx
  count: 1
```

### Multiple Pod Failure

Test resilience by killing 3 frontend pods:

```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: frontend-resilience-test
  namespace: production
spec:
  action: "pod-kill"
  namespace: "production"
  selector:
    app: web-server
    tier: frontend
  count: 3
```

### StatefulSet Failover

Test database failover by killing a specific replica:

```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: postgres-failover-test
  namespace: database
spec:
  action: "pod-kill"
  namespace: "database"
  selector:
    app: postgresql
    statefulset.kubernetes.io/pod-name: postgresql-1
  count: 1
```

### Network Delay (Future)

Add 100ms latency to API pods for 5 minutes:

```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: api-latency-test
  namespace: staging
spec:
  action: "pod-delay"
  namespace: "staging"
  selector:
    app: api-service
  count: 2
  duration: "5m"
```

---

## Best Practices

### Naming Conventions

Use descriptive names that indicate:
- What is being tested
- The type of chaos
- The environment

```yaml
# Good names
metadata:
  name: frontend-pod-kill-test
  name: api-latency-resilience
  name: postgres-failover-validation

# Poor names
metadata:
  name: experiment-1
  name: test
  name: chaos
```

### Label Organization

Use standard Kubernetes labels:

```yaml
metadata:
  labels:
    app.kubernetes.io/name: k8s-chaos
    app.kubernetes.io/component: experiment
    app.kubernetes.io/part-of: chaos-testing
    chaos.gushchin.dev/severity: low
    chaos.gushchin.dev/team: platform
```

### Selector Safety

Always be specific with selectors:

```yaml
# ❌ Too broad - might affect unintended pods
selector:
  tier: backend

# ✅ Specific - clear intent
selector:
  app: user-service
  version: v2.1.0
  environment: staging
```

### Progressive Chaos

Start small and increase gradually:

```yaml
# Week 1: Development
spec:
  namespace: "dev"
  count: 1

# Week 2: Staging, small scale
spec:
  namespace: "staging"
  count: 2

# Week 3: Staging, larger scale
spec:
  namespace: "staging"
  count: 5

# Week 4: Production (if confident)
spec:
  namespace: "production"
  count: 1  # Start small again
```

### Monitoring

Always monitor experiments:

```bash
# Watch pod status
kubectl get pods -n <namespace> -w

# Watch experiment status
kubectl get chaosexperiment -w

# View controller logs
kubectl logs -n k8s-chaos-system deployment/k8s-chaos-controller-manager -f
```

### Cleanup

Remove experiments when done:

```bash
# Delete specific experiment
kubectl delete chaosexperiment my-experiment

# Delete all experiments in namespace
kubectl delete chaosexperiment --all -n chaos-testing

# Delete with confirmation
kubectl delete chaosexperiment --all --dry-run=client
```

---

## kubectl explain

Get field documentation directly from kubectl:

```bash
# Explain ChaosExperiment
kubectl explain chaosexperiment

# Explain spec
kubectl explain chaosexperiment.spec

# Explain specific field
kubectl explain chaosexperiment.spec.action

# Explain status
kubectl explain chaosexperiment.status
```

---

## Related Documentation

- [Sample CRDs](../config/samples/README.md) - Ready-to-use examples
- [ADR 0001: CRD Validation Strategy](adr/0001-crd-validation-strategy.md) - Design decisions
- [Quick Start Guide](QUICKSTART.md) - Getting started
- [Development Guide](DEVELOPMENT.md) - Contributing and development setup
