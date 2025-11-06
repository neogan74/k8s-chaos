# k8s-chaos CLI Tool

A command-line interface for managing and monitoring k8s-chaos experiments in Kubernetes clusters.

## Installation

### Build from Source

```bash
# Build the CLI
make build-cli

# Install to /usr/local/bin
make install-cli

# Or manually copy the binary
cp bin/k8s-chaos /usr/local/bin/
```

### Prerequisites

- Go 1.24.5 or later
- Access to a Kubernetes cluster
- kubectl configured with appropriate permissions

## Usage

The CLI connects to your Kubernetes cluster using your kubeconfig file (default: `~/.kube/config`).

```bash
# Global flags
k8s-chaos [command] --kubeconfig=/path/to/config --namespace=<namespace>
```

## Commands

### `list` - List Experiments

List all chaos experiments in the cluster or a specific namespace.

```bash
# List all experiments across all namespaces
k8s-chaos list

# List experiments in a specific namespace
k8s-chaos list -n chaos-testing

# List with wide output showing more details
k8s-chaos list --wide
```

**Output (normal):**
```
NAMESPACE       NAME                  ACTION      TARGET-NS    PHASE      AGE
chaos-testing   nginx-chaos-demo      pod-kill    default      Running    2h
chaos-testing   api-delay-test        pod-delay   staging      Completed  1d
```

**Output (wide):**
```
NAMESPACE       NAME               ACTION      TARGET-NS  SELECTOR      COUNT  PHASE      RETRIES  DURATION  AGE
chaos-testing   nginx-chaos-demo   pod-kill    default    app=nginx     2      Running    0        10m       2h
chaos-testing   api-delay-test     pod-delay   staging    app=api       1      Completed  1        ∞         1d
```

### `describe` - Show Experiment Details

Display detailed information about a specific experiment.

```bash
# Describe an experiment
k8s-chaos describe nginx-chaos-demo -n chaos-testing
```

**Output:**
```
Name:         nginx-chaos-demo
Namespace:    chaos-testing
Created:      2025-10-27 14:30:00 (Age: 2h)

Spec:
  Action:              pod-kill
  Target Namespace:    default
  Selector:            app=nginx
  Count:               2
  Experiment Duration: 10m

Retry Configuration:
  Max Retries:         3
  Retry Backoff:       exponential
  Retry Delay:         30s

Status:
  Phase:               Running
  Message:             Successfully killed 2 pod(s)
  Start Time:          2025-10-27 14:30:05
  Last Run Time:       2025-10-27 16:25:00
```

### `delete` - Delete an Experiment

Remove a chaos experiment from the cluster.

```bash
# Delete an experiment (will prompt for confirmation)
k8s-chaos delete nginx-chaos-demo -n chaos-testing

# Delete without confirmation
k8s-chaos delete nginx-chaos-demo -n chaos-testing --force
```

### `stats` - View Statistics

Display aggregate statistics about chaos experiments.

```bash
# Show stats for all experiments
k8s-chaos stats

# Show stats for a specific namespace
k8s-chaos stats -n chaos-testing
```

**Output:**
```
=== Chaos Experiment Statistics ===
Namespace: All namespaces

Overall:
  Total Experiments:   15
  Running:             5
  Completed:           8
  Failed:              2
  Pending:             0

Success Rate:
  Successful:          53.3%
  Failed:              13.3%

By Action:
  ACTION        COUNT   PERCENTAGE
  pod-kill      8       53.3%
  pod-delay     4       26.7%
  node-drain    3       20.0%

Configuration:
  With Retry Logic:    12 (80.0%)
  Time-Limited:        10 (66.7%)
  Indefinite:          5 (33.3%)
```

### `top` - Show Top Experiments

Display experiments ranked by various metrics.

```bash
# Show top experiments by retry count
k8s-chaos top

# Show top 5 experiments
k8s-chaos top --limit 5

# Show top experiments in a specific namespace
k8s-chaos top -n chaos-testing
```

**Output:**
```
=== Top Experiments by Retry Count ===
NAMESPACE       NAME               ACTION      RETRIES  PHASE    AGE
chaos-testing   flaky-test         pod-kill    5        Failed   3d
production      api-stress-test    pod-delay   3        Running  1d

=== Top Experiments by Age ===
NAMESPACE       NAME               ACTION      PHASE      AGE
production      long-running       pod-kill    Running    7d
staging         continuous-test    pod-delay   Running    5d

=== Failed Experiments ===
NAMESPACE       NAME               ACTION      RETRIES  AGE
chaos-testing   flaky-test         pod-kill    5        3d
staging         node-test-fail     node-drain  3        2d
```

## Common Workflows

### Quick Experiment Overview

```bash
# Get a quick overview of all experiments
k8s-chaos list --wide

# See detailed stats
k8s-chaos stats

# Identify problematic experiments
k8s-chaos top
```

### Investigating a Specific Experiment

```bash
# Get detailed information
k8s-chaos describe my-experiment -n chaos-testing

# Check if it's in the failed list
k8s-chaos top | grep my-experiment
```

### Cleaning Up Experiments

```bash
# List all experiments
k8s-chaos list -n chaos-testing

# Delete completed experiments
k8s-chaos delete old-experiment -n chaos-testing --force
```

## Configuration

### Kubeconfig

By default, the CLI uses your default kubeconfig file (`~/.kube/config`). You can specify a different config:

```bash
k8s-chaos list --kubeconfig=/path/to/custom/config
```

Or set the `KUBECONFIG` environment variable:

```bash
export KUBECONFIG=/path/to/custom/config
k8s-chaos list
```

### Namespace

Specify the namespace for operations:

```bash
# Short flag
k8s-chaos list -n chaos-testing

# Long flag
k8s-chaos list --namespace=chaos-testing

# All namespaces (default for list and stats)
k8s-chaos list
```

## Troubleshooting

### "No chaos experiments found"

- Verify the CRD is installed: `kubectl get crd chaosexperiments.chaos.gushchin.dev`
- Check you're looking in the right namespace: `k8s-chaos list` (without -n flag)
- Verify RBAC permissions for listing ChaosExperiment resources

### "Failed to get Kubernetes client"

- Check your kubeconfig is valid: `kubectl cluster-info`
- Verify the kubeconfig path is correct
- Ensure you have network connectivity to the cluster

### "Failed to get experiment"

- Verify the experiment name is correct: `k8s-chaos list -n <namespace>`
- Ensure you specified the correct namespace with `-n` flag
- Check RBAC permissions for the namespace

## Examples

### Monitor a Long-Running Experiment

```bash
# Start the experiment
kubectl apply -f my-experiment.yaml

# Watch its status
watch -n 5 'k8s-chaos describe my-experiment -n chaos-testing'

# Check overall stats periodically
k8s-chaos stats -n chaos-testing
```

### Clean Up Failed Experiments

```bash
# Find failed experiments
k8s-chaos top | grep Failed

# Delete them
k8s-chaos delete failed-exp-1 -n chaos-testing --force
k8s-chaos delete failed-exp-2 -n chaos-testing --force
```

### Compare Experiment Statistics Over Time

```bash
# Take a snapshot
k8s-chaos stats > stats-$(date +%Y%m%d).txt

# Compare later
k8s-chaos stats > stats-$(date +%Y%m%d).txt
diff stats-20251027.txt stats-20251028.txt
```

## Future Enhancements

Planned features for future releases:

- **Interactive Wizard**: `k8s-chaos create --interactive`
- **Validation**: `k8s-chaos validate experiment.yaml`
- **Health Check**: `k8s-chaos check` - verify cluster readiness
- **Logs/History**: View experiment execution history
- **Watch Mode**: Real-time updates with `--watch` flag
- **Export**: Export stats to JSON/CSV format
- **Dashboard**: Web-based UI integration

## Integration with kubectl

You can use the CLI alongside kubectl:

```bash
# Use kubectl for detailed CRD inspection
kubectl describe chaosexperiment my-experiment -n chaos-testing

# Use k8s-chaos CLI for better formatting
k8s-chaos describe my-experiment -n chaos-testing

# Combine both
kubectl get chaosexperiment -n chaos-testing
k8s-chaos stats -n chaos-testing
```

## See Also

- [Metrics Documentation](METRICS.md) - Prometheus metrics integration
- [ChaosExperiment Samples](../config/samples/README.md) - Example experiments
- [Main README](../Readme.md) - Project overview and controller setup
