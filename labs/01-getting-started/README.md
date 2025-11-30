# Lab 01: Getting Started with k8s-chaos

## Objectives
After completing this lab, you will be able to:
- [ ] Install the k8s-chaos operator
- [ ] Create your first ChaosExperiment
- [ ] Monitor experiment execution
- [ ] Understand basic CRD structure
- [ ] Clean up resources

## Prerequisites
- Kubernetes cluster (use `make cluster-single-node` if you don't have one)
- kubectl configured
- Basic understanding of Kubernetes pods

## Lab Duration
Estimated time: 15-20 minutes

---

## Step 1: Verify Your Environment

Check that you have a running Kubernetes cluster:

```bash
kubectl cluster-info
kubectl get nodes
```

You should see your cluster information and at least one node in `Ready` state.

## Step 2: Install k8s-chaos Operator

From the project root directory, install the CRDs and deploy the operator:

```bash
# Install CRDs
make install

# Deploy the operator
make deploy IMG=controller:latest

# Verify the installation
kubectl get pods -n k8s-chaos-system
```

Wait for the operator pod to be in `Running` state.

## Step 3: Deploy a Demo Application

Create a namespace and deploy a simple nginx application:

```bash
kubectl create namespace chaos-demo

kubectl apply -f labs/01-getting-started/setup/demo-app.yaml

# Wait for pods to be ready
kubectl wait --for=condition=ready pod -l app=nginx -n chaos-demo --timeout=60s

# Verify pods are running
kubectl get pods -n chaos-demo
```

You should see 5 nginx pods running.

## Step 4: Create Your First Chaos Experiment

Let's create a simple pod-kill experiment:

```bash
kubectl apply -f labs/01-getting-started/experiments/01-simple-pod-kill.yaml
```

View the experiment:

```bash
kubectl get chaosexperiments -n chaos-demo

kubectl describe chaosexperiment simple-pod-kill -n chaos-demo
```

## Step 5: Monitor the Chaos

Watch the pods being killed and recreated:

```bash
# In one terminal, watch the pods
kubectl get pods -n chaos-demo -w

# In another terminal, watch the experiment
watch -n 2 'kubectl get chaosexperiments -n chaos-demo -o wide'
```

You should see:
1. Pods being terminated
2. Deployment recreating the terminated pods
3. Experiment status updating

## Step 6: Explore Experiment Status

Check the experiment status in detail:

```bash
kubectl get chaosexperiment simple-pod-kill -n chaos-demo -o yaml | grep -A 10 status:
```

Notice these status fields:
- `phase`: Current state (Running, Completed, Failed)
- `lastRunTime`: When the chaos action last executed
- `message`: Human-readable status message

## Step 7: Stop the Experiment

Delete the experiment to stop the chaos:

```bash
kubectl delete chaosexperiment simple-pod-kill -n chaos-demo
```

The pods will no longer be killed randomly.

## Step 8: Try a Dry-Run Experiment

Before executing real chaos, you can preview the impact:

```bash
kubectl apply -f labs/01-getting-started/experiments/02-dry-run-example.yaml

# Check the status message
kubectl get chaosexperiment dry-run-test -n chaos-demo -o jsonpath='{.status.message}'
```

The dry-run shows which pods would be affected without actually affecting them!

## Step 9: Cleanup

Remove all lab resources:

```bash
# Delete experiments
kubectl delete chaosexperiments --all -n chaos-demo

# Delete demo app
kubectl delete -f labs/01-getting-started/setup/demo-app.yaml

# Delete namespace
kubectl delete namespace chaos-demo
```

---

## What You Learned

✅ How to install k8s-chaos operator
✅ Basic ChaosExperiment CRD structure
✅ How to create and monitor experiments
✅ Understanding experiment status fields
✅ Using dry-run mode for safety

## Next Steps

- **Lab 02**: Explore different pod chaos actions (delay, CPU stress, failure)
- **Lab 03**: Learn about safety features (maxPercentage, exclusions, production protection)
- **Lab 04**: Experiment with node chaos

## Troubleshooting

**Operator pod not starting?**
```bash
kubectl logs -n k8s-chaos-system deployment/k8s-chaos-controller-manager
```

**Experiment not executing?**
Check the operator logs and experiment status:
```bash
kubectl describe chaosexperiment <name> -n chaos-demo
```

**Pods not being recreated?**
Ensure your deployment has replicas configured properly.

---

## Additional Resources

- Main documentation: `/docs`
- API Reference: `/docs/API.md`
- Metrics Guide: `/docs/METRICS.md`