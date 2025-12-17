# Lab 04: Node Chaos

## Objectives
After completing this lab, you will be able to:
- [ ] Understand node-drain action behavior
- [ ] Execute node chaos safely
- [ ] Handle workload migration during node drain
- [ ] Use dry-run to preview node impact
- [ ] Uncordon nodes after experiments

## Prerequisites
- Completed Labs 01-03
- **Multi-node cluster** (use `make cluster-multi-node` from labs root)
- k8s-chaos operator installed and running

## Lab Duration
Estimated time: 20-25 minutes

---

## Important: Multi-Node Cluster Required

Node chaos requires multiple nodes. Check your cluster:

```bash
kubectl get nodes
```

If you only have one node, create a multi-node cluster:

```bash
cd labs
make cluster-delete
make cluster-multi-node
make install deploy
```

---

## Step 1: Setup Lab Environment

```bash
cd labs/04-node-chaos
make setup
```

This creates a deployment with replicas spread across worker nodes.

Verify pod distribution:
```bash
kubectl get pods -n chaos-lab -o wide
```

You should see pods running on different nodes.

---

## Step 2: Understanding Node Drain

When a node is drained:
1. Node is **cordoned** (marked unschedulable)
2. Pods are **evicted** (gracefully terminated)
3. Pods are **rescheduled** on other nodes

This simulates:
- Node maintenance scenarios
- Cloud provider spot instance termination
- Hardware failures
- Cluster autoscaler scale-down

---

## Step 3: Preview Node Drain Impact

Always use dry-run first for node chaos:

```bash
# Apply dry-run experiment
kubectl apply -f experiments/01-node-drain-dryrun.yaml

# Check which node would be affected
kubectl describe chaosexperiment node-drain-preview -n chaos-lab
```

**Expected status:**
```
DRY RUN: Would drain 1 node(s): [k8s-chaos-lab-worker]
```

Clean up:
```bash
kubectl delete chaosexperiment node-drain-preview -n chaos-lab
```

---

## Step 4: Execute Node Drain

Now execute the actual drain:

```bash
# Apply the node drain experiment
kubectl apply -f experiments/02-node-drain.yaml
```

**Watch the chaos unfold:**

```bash
# Terminal 1: Watch node status
watch kubectl get nodes

# Terminal 2: Watch pod migration
kubectl get pods -n chaos-lab -o wide -w

# Terminal 3: Watch experiment status
watch kubectl get chaosexperiment -n chaos-lab
```

You should see:
1. One worker node becomes `SchedulingDisabled`
2. Pods on that node get evicted
3. New pods scheduled on remaining nodes

---

## Step 5: Verify Node State

After drain completes:

```bash
# Check node status - one should be cordoned
kubectl get nodes

# Check pods are all running on available nodes
kubectl get pods -n chaos-lab -o wide
```

**Note**: The node remains cordoned (unschedulable) after drain.

---

## Step 6: Uncordon the Node

To restore the node to service:

```bash
# Find the cordoned node
kubectl get nodes

# Uncordon it (replace with your node name)
kubectl uncordon k8s-chaos-lab-worker

# Verify
kubectl get nodes
```

All nodes should now show `Ready` without `SchedulingDisabled`.

---

## Step 7: Node Drain with Selector

Target specific nodes using labels:

```bash
# Check node labels
kubectl get nodes --show-labels

# Apply experiment targeting specific labels
kubectl apply -f experiments/03-node-drain-selector.yaml
```

This targets nodes with `chaos-test=enabled` label (set in our Kind config).

Clean up:
```bash
kubectl delete chaosexperiment node-drain-labeled -n chaos-lab
kubectl uncordon -l chaos-test=enabled
```

---

## Step 8: Node Chaos Best Practices

### Always Test with Dry-Run First
```yaml
spec:
  action: node-drain
  dryRun: true  # Preview which nodes will be affected
```

### Start with count: 1
```yaml
spec:
  action: node-drain
  count: 1  # Never drain multiple nodes at once initially
```

### Use Pod Disruption Budgets
```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: nginx-pdb
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: nginx
```

### Monitor During Drain
- Watch `kubectl get nodes` for cordon status
- Watch `kubectl get pods -o wide` for migration
- Check application health endpoints

---

## Step 9: Cleanup

```bash
# Delete experiments
kubectl delete chaosexperiments --all -n chaos-lab

# Uncordon any cordoned nodes
kubectl uncordon -l chaos-test=enabled 2>/dev/null || true

# Full cleanup
make teardown
```

---

## What You Learned

- node-drain cordons nodes and evicts pods
- Dry-run previews which nodes would be affected
- Pods are rescheduled on remaining nodes
- Nodes remain cordoned after drain (manual uncordon required)
- Pod Disruption Budgets protect workloads during drain

## Next Steps

- **Lab 05**: Schedule recurring chaos with cron expressions
- **Lab 06**: Configure retry logic for resilient experiments
- **Lab 07**: Monitor chaos with Prometheus and Grafana

## Troubleshooting

**Node not draining?**
- Check if PodDisruptionBudget is blocking eviction
- Review operator logs: `kubectl logs -n k8s-chaos-system deployment/k8s-chaos-controller-manager`
- Check for pods with `pod-eviction-timeout` annotation

**Pods not rescheduling?**
- Ensure other nodes have capacity
- Check for node affinity/anti-affinity rules
- Verify resource requests don't exceed available resources

**Single node cluster?**
- Node drain won't work properly with one node
- Use `make cluster-multi-node` to create 3-node cluster