# Lab 03: Safety Features

## Objectives
After completing this lab, you will be able to:
- [ ] Preview chaos impact with dry-run mode
- [ ] Limit blast radius with maxPercentage
- [ ] Protect production namespaces
- [ ] Exclude critical pods and namespaces
- [ ] Combine safety features for maximum protection

## Prerequisites
- Completed Labs 01-02
- k8s-chaos operator installed and running
- kubectl configured

## Lab Duration
Estimated time: 25-30 minutes

---

## Overview: Safety Features

k8s-chaos provides four layers of safety:

| Feature | Purpose | Use Case |
|---------|---------|----------|
| `dryRun` | Preview impact | See affected resources before execution |
| `maxPercentage` | Limit blast radius | Ensure count doesn't exceed N% of pods |
| `allowProduction` | Production gate | Require explicit approval for production |
| Exclusion labels | Protect resources | Never affect critical pods/namespaces |

---

## Step 1: Setup Lab Environment

```bash
cd labs/03-safety-features
make setup
```

This creates:
- `chaos-lab` namespace with 10 nginx pods
- `production` namespace (marked as production) with 5 pods
- `critical-ns` namespace (marked excluded) with 3 pods

Verify:
```bash
kubectl get pods -n chaos-lab
kubectl get pods -n production
kubectl get pods -n critical-ns
kubectl get ns production -o yaml | grep -A5 annotations
kubectl get ns critical-ns -o yaml | grep -A5 annotations
```

---

## Step 2: Dry-Run Mode

Dry-run lets you preview which resources would be affected without executing chaos:

```bash
# Apply dry-run experiment
kubectl apply -f experiments/01-dry-run.yaml

# Check the status - shows what WOULD happen
kubectl get chaosexperiment dry-run-preview -n chaos-lab -o yaml | grep -A5 status:
```

**Expected output:**
```
status:
  message: "DRY RUN: Would delete 3 pod(s): [nginx-xxx, nginx-yyy, nginx-zzz]"
  phase: Completed
```

The pods are listed but NOT deleted. This is perfect for:
- Validating selectors match expected pods
- Reviewing impact before production chaos
- Demonstrating experiments to stakeholders

Clean up:
```bash
kubectl delete chaosexperiment dry-run-preview -n chaos-lab
```

---

## Step 3: Maximum Percentage Limit

`maxPercentage` prevents affecting too many resources at once:

```bash
# Try to apply an experiment that would affect 50% of pods
kubectl apply -f experiments/02-max-percentage-fail.yaml
```

**This will FAIL** because:
- 10 pods exist in chaos-lab
- count=5 would affect 50%
- maxPercentage=30 only allows 30%

**Expected error:**
```
Error: admission webhook denied: count 5 exceeds maxPercentage 30% (max allowed: 3 pods out of 10)
```

Now apply a valid experiment:
```bash
# This works - count=2 is 20% (under 30% limit)
kubectl apply -f experiments/03-max-percentage-ok.yaml

# Verify it runs
kubectl get chaosexperiment percentage-limited -n chaos-lab
```

Clean up:
```bash
kubectl delete chaosexperiment percentage-limited -n chaos-lab
```

---

## Step 4: Production Namespace Protection

Production namespaces require explicit `allowProduction: true`:

```bash
# Try to create chaos in production WITHOUT allowProduction
kubectl apply -f experiments/04-production-denied.yaml
```

**This will FAIL:**
```
Error: admission webhook denied: namespace "production" is marked as production; set allowProduction: true to proceed
```

Now create with explicit approval:
```bash
# Apply with allowProduction: true
kubectl apply -f experiments/05-production-allowed.yaml

# Verify - this works because we explicitly approved
kubectl get chaosexperiment production-approved -n chaos-lab
kubectl describe chaosexperiment production-approved -n chaos-lab
```

Clean up:
```bash
kubectl delete chaosexperiment production-approved -n chaos-lab
```

### How Production is Detected

Namespaces are considered production if ANY of these match:
1. **Annotation**: `chaos.gushchin.dev/production: "true"`
2. **Label**: `environment: production` or `env: prod`
3. **Name patterns**: `production`, `prod-*`, `*-production`, `*-prod`

Check our setup:
```bash
kubectl get ns production -o yaml | grep -A10 metadata:
```

---

## Step 5: Exclusion Labels

Critical pods and namespaces can be permanently excluded:

### Pod-level Exclusion

```bash
# Check the critical pod in chaos-lab
kubectl get pod -n chaos-lab -l app=critical -o yaml | grep -A5 labels:
```

Notice `chaos.gushchin.dev/exclude: "true"` label.

```bash
# Try to affect all pods including critical
kubectl apply -f experiments/06-exclusion-pods.yaml

# Check status - critical pod is NOT in the list
kubectl describe chaosexperiment exclusion-test -n chaos-lab
```

The webhook warns but allows the experiment, automatically filtering excluded pods.

### Namespace-level Exclusion

```bash
# Check critical-ns namespace
kubectl get ns critical-ns -o yaml | grep -A5 annotations:
```

Notice `chaos.gushchin.dev/exclude: "true"` annotation.

```bash
# Try to target critical-ns
kubectl apply -f experiments/07-excluded-namespace.yaml
```

**This will FAIL** - the entire namespace is protected.

Clean up:
```bash
kubectl delete chaosexperiment exclusion-test -n chaos-lab --ignore-not-found
```

---

## Step 6: Combining Safety Features

Use all features together for maximum safety:

```bash
# Review the comprehensive experiment
cat experiments/08-all-safety.yaml

# Apply it
kubectl apply -f experiments/08-all-safety.yaml

# Check the result
kubectl describe chaosexperiment all-safety-demo -n chaos-lab
```

This experiment:
1. Uses `dryRun: true` to preview first
2. Limits to `maxPercentage: 25` of pods
3. Would require `allowProduction: true` if targeting production
4. Excludes any pods with exclusion labels

**Best practice workflow:**
1. Create experiment with `dryRun: true`
2. Review status to see affected resources
3. Adjust selector or count if needed
4. Change `dryRun: false` to execute

Clean up:
```bash
kubectl delete chaosexperiment all-safety-demo -n chaos-lab
```

---

## Step 7: Safety Checklist

Before running chaos in production, verify:

- [ ] **Dry-run first**: Always preview with `dryRun: true`
- [ ] **Limit blast radius**: Set appropriate `maxPercentage`
- [ ] **Explicit approval**: Use `allowProduction: true` consciously
- [ ] **Protect critical pods**: Add `chaos.gushchin.dev/exclude: "true"` label
- [ ] **Protect critical namespaces**: Add exclusion annotation
- [ ] **Start small**: Begin with `count: 1` and increase gradually
- [ ] **Use duration limits**: Set `experimentDuration` for auto-stop

---

## Step 8: Cleanup

```bash
make teardown
```

---

## What You Learned

- `dryRun: true` previews affected resources without executing
- `maxPercentage` limits chaos to N% of matching pods
- Production namespaces require explicit `allowProduction: true`
- Exclusion labels protect critical pods and namespaces
- Combining safety features provides defense in depth

## Next Steps

- **Lab 04**: Node-level chaos with node-drain
- **Lab 05**: Schedule recurring chaos experiments
- **Lab 06**: Configure retry logic for resilient experiments

## Troubleshooting

**Webhook rejecting all experiments?**
- Check operator logs: `kubectl logs -n k8s-chaos-system deployment/k8s-chaos-controller-manager`
- Verify webhook is configured: `kubectl get validatingwebhookconfigurations`

**maxPercentage calculation wrong?**
- Remember it calculates against ALL matching pods, not just available ones
- Excluded pods are filtered AFTER percentage calculation

**Production detection not working?**
- Check namespace labels/annotations exactly match expected patterns
- Name patterns are case-sensitive