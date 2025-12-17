# Lab 02: Pod Chaos Basics

## Objectives
After completing this lab, you will be able to:
- [ ] Execute different pod chaos actions
- [ ] Inject network latency with pod-delay
- [ ] Generate CPU stress with pod-cpu-stress
- [ ] Simulate container crashes with pod-failure
- [ ] Understand when to use each chaos type

## Prerequisites
- Completed Lab 01 (Getting Started)
- k8s-chaos operator installed and running
- kubectl configured

## Lab Duration
Estimated time: 25-30 minutes

---

## Step 1: Setup Lab Environment

Deploy the demo application with multiple replicas:

```bash
cd labs/02-pod-chaos-basics
make setup
```

Verify the deployment:
```bash
kubectl get pods -n chaos-lab -o wide
kubectl get svc -n chaos-lab
```

You should see 5 nginx pods running across the cluster.

---

## Step 2: Pod Kill (Review from Lab 01)

Let's quickly review pod-kill before exploring other actions:

```bash
# Apply the pod-kill experiment
kubectl apply -f experiments/01-pod-kill.yaml

# Watch pods being killed and recreated
kubectl get pods -n chaos-lab -w
```

Notice how pods are terminated and the Deployment recreates them. Delete the experiment:

```bash
kubectl delete chaosexperiment pod-kill-demo -n chaos-lab
```

---

## Step 3: Network Delay (pod-delay)

Network latency is one of the most common production issues. Let's inject artificial delay:

```bash
# Review the experiment
cat experiments/02-pod-delay.yaml

# Apply the network delay experiment
kubectl apply -f experiments/02-pod-delay.yaml
```

**What's happening:**
- The operator uses `tc` (traffic control) to add network latency
- Delay is applied to the pod's network interface
- This affects all network traffic to/from the pod

Monitor the experiment:
```bash
kubectl get chaosexperiment pod-delay-demo -n chaos-lab -o wide
kubectl describe chaosexperiment pod-delay-demo -n chaos-lab
```

**Test the latency:**
```bash
# Get a pod name
POD=$(kubectl get pods -n chaos-lab -l app=nginx -o jsonpath='{.items[0].metadata.name}')

# Exec into the pod and test latency (you should see ~500ms added)
kubectl exec -n chaos-lab $POD -- sh -c "time wget -q -O /dev/null http://localhost"
```

Clean up:
```bash
kubectl delete chaosexperiment pod-delay-demo -n chaos-lab
```

---

## Step 4: CPU Stress (pod-cpu-stress)

CPU stress testing validates how your application handles resource contention:

```bash
# Review the experiment
cat experiments/03-pod-cpu-stress.yaml

# Apply CPU stress experiment
kubectl apply -f experiments/03-pod-cpu-stress.yaml
```

**What's happening:**
- The operator injects an ephemeral container running `stress-ng`
- `cpuLoad: 80` means 80% CPU utilization per worker
- `cpuWorkers: 2` means 2 parallel stress processes
- Duration controls how long the stress runs

Monitor CPU usage:
```bash
# Watch pod resource usage
kubectl top pods -n chaos-lab --containers

# Check experiment status
kubectl get chaosexperiment cpu-stress-demo -n chaos-lab -o yaml | grep -A 20 status:
```

Clean up:
```bash
kubectl delete chaosexperiment cpu-stress-demo -n chaos-lab
```

---

## Step 5: Pod Failure (pod-failure)

Pod failure simulates container crashes to test recovery behavior:

```bash
# Review the experiment
cat experiments/04-pod-failure.yaml

# Apply pod failure experiment
kubectl apply -f experiments/04-pod-failure.yaml
```

**What's happening:**
- The operator executes `kill -9 1` inside the container
- This kills PID 1 (the main process), causing container crash
- Kubernetes restarts the container based on restartPolicy

Watch the pods crash and recover:
```bash
# Watch for container restarts
kubectl get pods -n chaos-lab -w

# Check restart counts
kubectl get pods -n chaos-lab -o custom-columns=NAME:.metadata.name,RESTARTS:.status.containerStatuses[0].restartCount
```

Clean up:
```bash
kubectl delete chaosexperiment pod-failure-demo -n chaos-lab
```

---

## Step 6: Compare Chaos Actions

Let's compare all pod chaos types:

| Action | Effect | Use Case | Recovery |
|--------|--------|----------|----------|
| `pod-kill` | Deletes pod completely | Test pod rescheduling, PDB compliance | New pod scheduled |
| `pod-delay` | Adds network latency | Test timeout handling, circuit breakers | Remove tc rules |
| `pod-cpu-stress` | Consumes CPU resources | Test throttling, autoscaling | Ephemeral container exits |
| `pod-failure` | Crashes container | Test restart behavior, init containers | Container restarts |

**When to use each:**

- **pod-kill**: Test how your system handles complete pod loss
- **pod-delay**: Test resilience to network slowness (more realistic than kill)
- **pod-cpu-stress**: Test behavior under CPU pressure, validate limits
- **pod-failure**: Test container restart behavior, application recovery

---

## Step 7: Combined Experiment

Try running multiple experiments on different pod sets:

```bash
# Apply a combined experiment
kubectl apply -f experiments/05-combined.yaml

# Watch the chaos unfold
kubectl get chaosexperiments -n chaos-lab
kubectl get pods -n chaos-lab -w
```

Clean up:
```bash
kubectl delete chaosexperiments --all -n chaos-lab
```

---

## Step 8: Cleanup

```bash
make teardown
```

---

## What You Learned

- How each pod chaos action works internally
- pod-delay uses tc for network latency injection
- pod-cpu-stress uses ephemeral containers with stress-ng
- pod-failure kills PID 1 to crash containers
- When to use each chaos type for different scenarios

## Next Steps

- **Lab 03**: Learn safety features (dry-run, maxPercentage, exclusions)
- **Lab 04**: Explore node-level chaos
- **Lab 05**: Schedule recurring chaos experiments

## Troubleshooting

**pod-delay not working?**
- Check if pod has NET_ADMIN capability
- Review operator logs: `kubectl logs -n k8s-chaos-system deployment/k8s-chaos-controller-manager`

**pod-cpu-stress ephemeral container not starting?**
- Ensure Kubernetes version 1.25+ (ephemeral containers GA)
- Check if stress-ng image is accessible

**pod-failure not crashing containers?**
- Some containers run non-PID-1 processes
- Check container's main process: `kubectl exec <pod> -- ps aux`