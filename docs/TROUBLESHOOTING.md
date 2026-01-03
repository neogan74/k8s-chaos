# Troubleshooting Guide

This guide helps you diagnose and resolve common issues with k8s-chaos.

## Table of Contents

- [Quick Diagnostics](#quick-diagnostics)
- [Installation Issues](#installation-issues)
- [Operator Issues](#operator-issues)
- [Experiment Issues](#experiment-issues)
- [Webhook Issues](#webhook-issues)
- [Permission Issues](#permission-issues)
- [Action-Specific Issues](#action-specific-issues)
- [Performance Issues](#performance-issues)
- [Getting Help](#getting-help)

---

## Quick Diagnostics

Run these commands first to gather information:

```bash
# Check operator status
kubectl get pods -n k8s-chaos-system
kubectl logs -n k8s-chaos-system deployment/k8s-chaos-controller-manager --tail=50

# Check CRDs
kubectl get crds | grep chaos

# Check experiments
kubectl get chaosexperiments --all-namespaces

# Check webhook
kubectl get validatingwebhookconfigurations | grep chaos
```

---

## Installation Issues

### Issue: CRDs Not Installing

**Symptoms:**
```bash
$ kubectl get crds | grep chaos
# No output
```

**Cause:** Installation command failed or insufficient permissions

**Solution:**
```bash
# Check if you have cluster-admin rights
kubectl auth can-i create customresourcedefinitions

# If yes, reinstall CRDs
make uninstall
make install

# Verify
kubectl get crds | grep chaos.gushchin.dev
```

**Expected output:**
```
chaosexperimenthistories.chaos.gushchin.dev
chaosexperiments.chaos.gushchin.dev
```

### Issue: Operator Pod Not Starting

**Symptoms:**
```bash
$ kubectl get pods -n k8s-chaos-system
NAME                                               READY   STATUS    RESTARTS   AGE
k8s-chaos-controller-manager-xxx                   0/2     Pending   0          5m
```

**Causes & Solutions:**

**1. Image Pull Error**
```bash
# Check pod events
kubectl describe pod -n k8s-chaos-system <pod-name>

# If "ImagePullBackOff" or "ErrImagePull":
# For Kind clusters:
kind load docker-image k8s-chaos-controller:latest --name <cluster-name>

# For other clusters, push to registry:
docker tag k8s-chaos-controller:latest your-registry/k8s-chaos:v1.0
docker push your-registry/k8s-chaos:v1.0
make deploy IMG=your-registry/k8s-chaos:v1.0
```

**2. Insufficient Resources**
```bash
# Check node resources
kubectl top nodes

# If nodes are full, scale down other workloads or add nodes
```

**3. Webhook Certificate Issues**
```bash
# Check webhook pod logs
kubectl logs -n k8s-chaos-system deployment/k8s-chaos-controller-manager

# If certificate errors, reinstall:
make undeploy
make deploy IMG=<your-image>
```

### Issue: Multiple Controller Versions

**Symptoms:**
- Conflicting behavior
- Experiments not executing correctly

**Solution:**
```bash
# List all deployments in k8s-chaos-system
kubectl get deployments -n k8s-chaos-system

# Delete old deployments
kubectl delete deployment <old-deployment-name> -n k8s-chaos-system

# Redeploy correct version
make deploy IMG=<correct-image>
```

---

## Operator Issues

### Issue: Operator Crashes or Restarts Frequently

**Symptoms:**
```bash
$ kubectl get pods -n k8s-chaos-system
NAME                                     READY   STATUS             RESTARTS   AGE
k8s-chaos-controller-manager-xxx         1/2     CrashLoopBackOff   5          10m
```

**Diagnosis:**
```bash
# Check logs
kubectl logs -n k8s-chaos-system deployment/k8s-chaos-controller-manager --previous

# Common errors to look for:
# - "OOM killed" → Memory limit too low
# - "panic" → Bug in code
# - "connection refused" → Can't reach API server
```

**Solutions:**

**1. Increase Resource Limits**
```bash
# Edit deployment
kubectl edit deployment -n k8s-chaos-system k8s-chaos-controller-manager

# Increase limits:
resources:
  limits:
    cpu: 500m      # from 200m
    memory: 512Mi  # from 128Mi
```

**2. Check API Server Connectivity**
```bash
# From operator pod
kubectl exec -n k8s-chaos-system <pod-name> -c manager -- \
  curl -k https://kubernetes.default.svc
```

### Issue: Operator Not Reconciling Experiments

**Symptoms:**
- Experiments stuck in `Pending` phase
- No logs showing reconciliation

**Diagnosis:**
```bash
# Check if controller is running
kubectl logs -n k8s-chaos-system deployment/k8s-chaos-controller-manager -f

# Look for reconciliation logs:
# Should see: "Reconciling ChaosExperiment"
```

**Solutions:**

**1. Verify RBAC Permissions**
```bash
# Check if controller has necessary permissions
kubectl auth can-i list pods --as=system:serviceaccount:k8s-chaos-system:k8s-chaos-controller-manager -n default

# If "no", reinstall RBAC:
make manifests
kubectl apply -f config/rbac/
```

**2. Restart Operator**
```bash
kubectl rollout restart deployment -n k8s-chaos-system k8s-chaos-controller-manager
```

---

## Experiment Issues

### Issue: Experiment Rejected by Webhook

**Symptoms:**
```bash
$ kubectl apply -f experiment.yaml
Error from server: admission webhook "vchaosexperiment.kb.io" denied the request: ...
```

**Common Rejection Reasons:**

**1. Production Namespace Without Approval**
```
Error: chaos experiments in production namespace "production" require explicit approval: set allowProduction: true
```

**Solution:**
```yaml
spec:
  allowProduction: true  # Add this line
```

**2. Count Exceeds maxPercentage**
```
Error: count (5) would affect 50.0% of pods, exceeding maxPercentage limit of 30%: reduce count to 3 or lower
```

**Solution:**
```yaml
spec:
  count: 3  # Reduce to suggested value
  # or increase maxPercentage
```

**3. Namespace Doesn't Exist**
```
Error: target namespace "my-app" does not exist
```

**Solution:**
```bash
kubectl create namespace my-app
```

**4. Selector Matches No Pods**
```
Error: selector does not match any pods in namespace "default"
```

**Solution:**
```bash
# Check existing pod labels
kubectl get pods -n default --show-labels

# Update selector to match
spec:
  selector:
    app: correct-label  # Use actual label
```

**5. Missing Required Fields**
```
Error: duration is required for pod-delay action
```

**Solution:**
```yaml
spec:
  action: pod-delay
  duration: "100ms"  # Add required field
```

### Issue: Experiment Stuck in Pending

**Symptoms:**
```bash
$ kubectl get chaosexperiments
NAME        PHASE     AGE
my-test     Pending   10m
```

**Diagnosis:**
```bash
# Check experiment status
kubectl describe chaosexperiment my-test

# Check operator logs
kubectl logs -n k8s-chaos-system deployment/k8s-chaos-controller-manager --tail=100 | grep my-test
```

**Common Causes:**

**1. Scheduled Experiment Not Due Yet**
```yaml
spec:
  schedule: "0 10 * * *"  # Runs at 10:00 AM only
```

**Solution:** Wait for scheduled time or remove schedule for immediate execution

**2. All Pods Excluded**
```yaml
# All matching pods have exclusion label
Error: all 5 matching pods are excluded via chaos.gushchin.dev/exclude label
```

**Solution:** Remove exclusion labels or adjust selector

**3. Retry Delay**
```bash
# Experiment failed and waiting for retry
Status:
  retryCount: 1
  nextRetryTime: 2025-12-02T10:30:00Z
```

**Solution:** Wait for retry or delete and recreate experiment

### Issue: Experiment Completes Too Quickly

**Symptoms:**
- Experiment runs once and stops
- Expected continuous chaos

**Cause:** Missing `experimentDuration` or schedule

**Solution:**
```yaml
spec:
  # Option 1: Run for specific duration
  experimentDuration: "30m"

  # Option 2: Run on schedule
  schedule: "*/5 * * * *"  # Every 5 minutes
```

### Issue: Dry-Run Not Showing Expected Pods

**Symptoms:**
```bash
$ kubectl get chaosexperiment test -o jsonpath='{.status.message}'
DRY RUN: Would delete 1 pod(s): [pod-abc]
# Expected 3 pods, only shows 1
```

**Cause:** `count` vs available pods mismatch

**Diagnosis:**
```bash
# Check how many pods match selector
kubectl get pods -n <namespace> -l <selector> --no-headers | wc -l
```

**Solution:**
- If fewer pods than expected: Fix selector or deploy more replicas
- If more pods exist: Increase `count` in experiment

---

## Webhook Issues

### Issue: Webhook Not Responding

**Symptoms:**
```bash
$ kubectl apply -f experiment.yaml
Error: Internal error occurred: failed calling webhook "vchaosexperiment.kb.io": Post "https://k8s-chaos-webhook-service.k8s-chaos-system.svc:443/validate-chaos-gushchin-dev-v1alpha1-chaosexperiment?timeout=10s": context deadline exceeded
```

**Diagnosis:**
```bash
# Check webhook service
kubectl get svc -n k8s-chaos-system k8s-chaos-webhook-service

# Check webhook configuration
kubectl get validatingwebhookconfigurations | grep chaos

# Check operator pod (runs webhook)
kubectl get pods -n k8s-chaos-system
kubectl logs -n k8s-chaos-system <pod-name> -c manager
```

**Solutions:**

**1. Webhook Service Not Found**
```bash
# Recreate webhook service
make undeploy
make deploy IMG=<your-image>
```

**2. Certificate Issues**
```bash
# Delete and recreate webhook configuration
kubectl delete validatingwebhookconfigurations chaosexperiment-validating-webhook-configuration
make deploy IMG=<your-image>
```

**3. Temporary Bypass (NOT for production)**
```bash
# Delete webhook to bypass temporarily
kubectl delete validatingwebhookconfigurations chaosexperiment-validating-webhook-configuration

# Fix underlying issue, then reinstall
make deploy IMG=<your-image>
```

---

## Permission Issues

### Understanding Permission Error Messages

The operator provides detailed permission error messages with remediation steps. Example:

```
Permission denied: cannot list pods in namespace default. Missing permission: pods/list.
Troubleshooting: https://github.com/neogan74/k8s-chaos/blob/main/docs/TROUBLESHOOTING.md#permission-issues
Check with: kubectl auth can-i list pods --as=system:serviceaccount:k8s-chaos-system:k8s-chaos-controller-manager -n default
Fix: make manifests && kubectl apply -f config/rbac/
```

Each error message includes:
- **What failed**: The specific operation that was denied
- **Missing permission**: The exact resource/verb combination needed
- **Troubleshooting link**: Direct link to this documentation
- **Check command**: kubectl command to verify permissions
- **Fix suggestion**: How to remediate the issue

### Common Permission Scenarios

#### pod-kill Action
Required permissions:
- `pods/list` - To find target pods
- `pods/delete` - To kill pods

**Verification:**
```bash
kubectl auth can-i list pods \
  --as=system:serviceaccount:k8s-chaos-system:k8s-chaos-controller-manager \
  -n <namespace>
kubectl auth can-i delete pods \
  --as=system:serviceaccount:k8s-chaos-system:k8s-chaos-controller-manager \
  -n <namespace>
```

#### pod-cpu-stress, pod-memory-stress, pod-disk-fill Actions
Required permissions:
- `pods/list` - To find target pods
- `pods/ephemeralcontainers/update` - To inject ephemeral containers

**Verification:**
```bash
kubectl auth can-i update pods/ephemeralcontainers \
  --as=system:serviceaccount:k8s-chaos-system:k8s-chaos-controller-manager \
  -n <namespace>
```

#### pod-failure, pod-restart Actions
Required permissions:
- `pods/list` - To find target pods
- `pods/exec/create` - To execute commands in pods

**Verification:**
```bash
kubectl auth can-i create pods/exec \
  --as=system:serviceaccount:k8s-chaos-system:k8s-chaos-controller-manager \
  -n <namespace>
```

#### node-drain Action
Required permissions:
- `nodes/list` - To find target nodes
- `nodes/update` - To cordon nodes
- `nodes/patch` - To update node status
- `pods/eviction/create` - To evict pods from nodes

**Verification:**
```bash
kubectl auth can-i list nodes \
  --as=system:serviceaccount:k8s-chaos-system:k8s-chaos-controller-manager
kubectl auth can-i update nodes \
  --as=system:serviceaccount:k8s-chaos-system:k8s-chaos-controller-manager
kubectl auth can-i create pods/eviction \
  --as=system:serviceaccount:k8s-chaos-system:k8s-chaos-controller-manager \
  -n <namespace>
```

### Debugging Permission Issues

**Step 1: Check experiment status**
```bash
kubectl describe chaosexperiment <experiment-name>
# Look at Status.Message for detailed error
```

**Step 2: Verify RBAC is installed**
```bash
kubectl get clusterrole manager-role -o yaml
# Should contain all required permissions listed above
```

**Step 3: Run suggested kubectl command**
The error message contains a `kubectl auth can-i` command tailored to your specific issue. Run it to verify the permission is missing.

**Step 4: Reinstall RBAC**
```bash
make manifests
kubectl apply -f config/rbac/
```

**Step 5: Verify fix**
```bash
# Re-run the kubectl auth can-i command from step 3
# Should now return "yes"
```

### Retry Behavior for Permission Errors

Permission errors are automatically retried **once** with a 30-second delay, then marked as failed. This is different from execution errors which use exponential backoff with multiple retries.

**Rationale**: Permission errors rarely self-resolve and require manual RBAC fixes. Quick failure helps identify issues faster.

**Example:**
```bash
# First attempt: Permission denied
# Retry 1/1 in 30s
# (30 seconds later)
# Second attempt: Permission denied
# Failed after 1 retries
```

After fixing RBAC, you can manually re-run the experiment or wait for the next scheduled execution (if using cron).

---

## Action-Specific Issues

### pod-delay: Network Delay Not Applied

**Symptoms:**
- Experiment shows success
- No latency observed

**Causes:**

**1. tc Command Not Available in Container**
```bash
# Check if container has tc
kubectl exec -n <namespace> <pod-name> -- which tc
# If "not found": Container image doesn't include iproute2
```

**Solution:** Use container images that include `tc` command (most standard images do)

**2. Insufficient Permissions**
```bash
# Check pod logs
kubectl logs -n <namespace> <pod-name>
# Look for: "Operation not permitted"
```

**Solution:** Container needs `NET_ADMIN` capability (automatically granted by operator)

### pod-cpu-stress: Ephemeral Container Not Injecting

**Symptoms:**
- Experiment shows success
- No CPU usage increase

**Diagnosis:**
```bash
# Check if ephemeral container was added
kubectl get pod <pod-name> -o json | jq '.spec.ephemeralContainers'

# Check ephemeral container status
kubectl get pod <pod-name> -o json | jq '.status.ephemeralContainerStatuses'
```

**Common Issues:**

**1. Feature Gate Not Enabled (Kubernetes <1.23)**
```bash
# Check if feature is enabled
kubectl get --raw /metrics | grep ephemeral_containers
```

**Solution:** Upgrade to Kubernetes 1.23+ or enable feature gate

**2. Container Image Pull Failed**
```bash
kubectl describe pod <pod-name> | grep -A 5 "Ephemeral Containers"
```

**Solution:** Ensure `alexeiled/stress-ng:latest-alpine` image is accessible

### node-drain: Nodes Not Draining

**Symptoms:**
- Experiment stuck
- Nodes remain schedulable

**Diagnosis:**
```bash
# Check node status
kubectl get nodes

# Check if node is cordoned
kubectl describe node <node-name> | grep Unschedulable

# Check operator logs
kubectl logs -n k8s-chaos-system deployment/k8s-chaos-controller-manager | grep drain
```

**Common Issues:**

**1. PodDisruptionBudget Blocking Eviction**
```bash
# Check PDBs
kubectl get pdb --all-namespaces

# Pods can't be evicted if PDB not satisfied
```

**Solution:** Temporarily adjust PDB or increase replica count

**2. Pods with Local Storage**
```bash
# Pods with emptyDir or hostPath can't be evicted
kubectl get pods -o json | jq '.items[] | select(.spec.volumes[]?.emptyDir != null)'
```

**Solution:** Delete pods manually or skip these nodes

---

## Performance Issues

### Issue: High Memory Usage by Operator

**Symptoms:**
```bash
$ kubectl top pod -n k8s-chaos-system
NAME                                        CPU   MEMORY
k8s-chaos-controller-manager-xxx            50m   500Mi  # High!
```

**Causes:**
- Too many experiments running simultaneously
- Large number of history records
- Memory leak

**Solutions:**

**1. Limit Concurrent Experiments**
```bash
# Don't run too many experiments at once
# Aim for <10 concurrent experiments
```

**2. Configure History Retention**
```bash
# Edit operator deployment
kubectl edit deployment -n k8s-chaos-system k8s-chaos-controller-manager

# Add flag:
- --history-retention-limit=50  # Reduce from default 100
```

**3. Increase Memory Limit**
```bash
kubectl edit deployment -n k8s-chaos-system k8s-chaos-controller-manager

resources:
  limits:
    memory: 1Gi  # Increase from 512Mi
```

### Issue: Slow Experiment Execution

**Symptoms:**
- Experiments take long time to start
- Delays between reconciliations

**Diagnosis:**
```bash
# Check operator CPU
kubectl top pod -n k8s-chaos-system

# Check API server latency
kubectl get --raw /metrics | grep apiserver_request_duration
```

**Solutions:**
- Increase operator CPU limits
- Reduce number of concurrent experiments
- Check cluster overall health

---

## Getting Help

If you can't resolve the issue:

### 1. Gather Debug Information

```bash
#!/bin/bash
# Save this as debug-info.sh

echo "=== CRDs ===" > debug.txt
kubectl get crds | grep chaos >> debug.txt

echo "\n=== Operator Status ===" >> debug.txt
kubectl get pods -n k8s-chaos-system >> debug.txt

echo "\n=== Operator Logs ===" >> debug.txt
kubectl logs -n k8s-chaos-system deployment/k8s-chaos-controller-manager --tail=100 >> debug.txt

echo "\n=== Experiments ===" >> debug.txt
kubectl get chaosexperiments --all-namespaces -o wide >> debug.txt

echo "\n=== Webhook ===" >> debug.txt
kubectl get validatingwebhookconfigurations | grep chaos >> debug.txt

echo "Debug info saved to debug.txt"
```

### 2. Check Existing Issues

Search GitHub issues: https://github.com/neogan74/k8s-chaos/issues

### 3. Create New Issue

Include:
- k8s-chaos version
- Kubernetes version
- Cloud provider / distribution
- Steps to reproduce
- Expected vs actual behavior
- Debug information from above script
- Relevant logs

### 4. Community Support

- **GitHub Discussions**: https://github.com/neogan74/k8s-chaos/discussions
- **Documentation**: https://github.com/neogan74/k8s-chaos/tree/main/docs

---

## Additional Resources

- **Getting Started**: [GETTING-STARTED.md](GETTING-STARTED.md)
- **Best Practices**: [BEST-PRACTICES.md](BEST-PRACTICES.md)
- **API Documentation**: [API.md](API.md)
- **Metrics Guide**: [METRICS.md](METRICS.md)

---

Remember: Most issues are configuration-related. Check your YAML carefully! 🔍