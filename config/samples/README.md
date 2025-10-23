# ChaosExperiment Samples

This directory contains sample ChaosExperiment resources to help you get started with chaos engineering.

## Available Samples

### 1. Basic Pod Kill (`chaos_v1alpha1_chaosexperiment.yaml`)
- Kills 1 pod with label `app=nginx` in the `default` namespace
- Good for getting started and basic testing

### 2. Multiple Pod Kill (`chaos_v1alpha1_chaosexperiment_multiple.yaml`)
- Kills up to 3 pods with labels `app=web-server` AND `tier=frontend`
- Demonstrates multiple label selectors
- Targets `production` namespace

### 3. StatefulSet Testing (`chaos_v1alpha1_chaosexperiment_stateful.yaml`)
- Targets a specific StatefulSet pod by name
- Good for testing database failover scenarios
- Only affects 1 replica to avoid data loss

### 4. Demo Environment (`chaos_v1alpha1_chaosexperiment_demo.yaml`)
- Works with the provided `demo-deployment.yaml`
- Kills 2 out of 5 nginx replicas
- Safe for learning and experimentation

### 5. Network Delay Testing (`chaos_v1alpha1_chaosexperiment_delay.yaml`)
- Adds network latency to API service pods
- Demonstrates the `pod-delay` action
- Targets staging environment for safe testing

### 6. Node Drain Testing (`chaos_v1alpha1_chaosexperiment_node_drain.yaml`)
- Cordons and drains worker nodes to test node failure scenarios
- Demonstrates the `node-drain` action
- Uses node labels to target specific nodes
- **CAUTION**: Can cause significant disruption - test carefully!

## Demo Deployment

The `demo-deployment.yaml` file creates:
- A `chaos-demo` namespace
- An nginx deployment with 5 replicas
- A service to expose the deployment
- Proper health checks and resource limits

## Usage

### 1. Deploy the demo environment:
```bash
kubectl apply -f demo-deployment.yaml
```

### 2. Install the ChaosExperiment CRD (from project root):
```bash
make install
```

### 3. Run a chaos experiment:
```bash
kubectl apply -f chaos_v1alpha1_chaosexperiment_demo.yaml
```

### 4. Watch the chaos in action:
```bash
# Watch pods being terminated and recreated
kubectl get pods -n chaos-demo -w

# Check the experiment status
kubectl get chaosexperiment -n chaos-demo
kubectl describe chaosexperiment nginx-chaos-demo -n chaos-demo
```

### 5. Clean up:
```bash
kubectl delete -f chaos_v1alpha1_chaosexperiment_demo.yaml
kubectl delete -f demo-deployment.yaml
```

## Customizing Samples

Each sample includes comments explaining the fields:

- **action**: Supports `"pod-kill"`, `"pod-delay"`, `"node-drain"`
- **namespace**: Target namespace for pod-level chaos (not used for node-drain)
- **selector**: Label selector to identify target pods/nodes
- **count**: Number of pods/nodes to affect
- **duration**: Duration for time-based actions (required for pod-delay)

## Safety Tips

1. **Start small**: Begin with 1 pod/node and low-traffic environments
2. **Use selectors carefully**: Make sure your selector only targets intended resources
3. **Monitor impact**: Watch system metrics during experiments
4. **Have rollback ready**: Know how to quickly restore service if needed
5. **Avoid production**: Test in staging/dev environments first
6. **Node-drain caution**: Always test node selectors with `kubectl get nodes -l <selector>` first
7. **Uncordon nodes**: Remember to manually uncordon nodes after testing: `kubectl uncordon <node-name>`

## Troubleshooting

If pods aren't being killed:
1. Verify the selector matches your pods: `kubectl get pods -l app=nginx`
2. Check RBAC permissions for the controller
3. Look at controller logs: `kubectl logs -n k8s-chaos-system deployment/k8s-chaos-controller-manager`
4. Verify the CRD is installed: `kubectl get crd chaosexperiments.chaos.gushchin.dev`