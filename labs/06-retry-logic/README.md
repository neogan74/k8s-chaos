# Lab 06: Retry Logic

## Objectives
After completing this lab, you will be able to:
- [ ] Configure retry attempts for experiments
- [ ] Choose between exponential and fixed backoff
- [ ] Set appropriate retry delays
- [ ] Monitor retry status in experiment status
- [ ] Handle permanent failures gracefully

## Prerequisites
- Completed Labs 01-05
- k8s-chaos operator installed and running
- kubectl configured

## Lab Duration
Estimated time: 15-20 minutes

---

## Overview: Retry Logic

When chaos experiments encounter transient failures, retry logic ensures they eventually succeed:

| Setting | Description | Default |
|---------|-------------|---------|
| `maxRetries` | Maximum retry attempts (0-10) | 3 |
| `retryBackoff` | Strategy: "exponential" or "fixed" | exponential |
| `retryDelay` | Initial delay between retries | 30s |

### Backoff Strategies

**Exponential** (recommended):
- Delay doubles with each retry
- Example: 30s → 1m → 2m → 4m → 8m (max 10m)
- Best for transient infrastructure issues

**Fixed**:
- Constant delay between all retries
- Example: 30s → 30s → 30s → 30s
- Best for predictable recovery times

---

## Step 1: Setup Lab Environment

```bash
cd labs/06-retry-logic
make setup
```

Verify:
```bash
kubectl get pods -n chaos-lab
```

---

## Step 2: Basic Retry Configuration

Create an experiment with retry enabled:

```bash
# Apply experiment with retries
kubectl apply -f experiments/01-basic-retry.yaml

# Watch the experiment
kubectl get chaosexperiment retry-basic -n chaos-lab -w
```

Check retry status:
```bash
kubectl get chaosexperiment retry-basic -n chaos-lab -o yaml | grep -A15 status:
```

**Key status fields:**
- `retryCount`: Current retry attempt number
- `lastError`: Most recent error message
- `nextRetryTime`: When next retry will occur

Clean up:
```bash
kubectl delete chaosexperiment retry-basic -n chaos-lab
```

---

## Step 3: Exponential Backoff

Exponential backoff increases delay with each failure:

```bash
# Apply exponential backoff experiment
kubectl apply -f experiments/02-exponential-backoff.yaml

# Watch retry timing
watch -n 5 'kubectl get chaosexperiment exponential-retry -n chaos-lab -o yaml | grep -E "retryCount|nextRetryTime|phase"'
```

**Retry timeline with 30s initial delay:**
| Attempt | Delay | Cumulative |
|---------|-------|------------|
| 1 | 30s | 30s |
| 2 | 1m | 1m 30s |
| 3 | 2m | 3m 30s |
| 4 | 4m | 7m 30s |
| 5 | 8m | 15m 30s |

*Maximum delay capped at 10 minutes*

Clean up:
```bash
kubectl delete chaosexperiment exponential-retry -n chaos-lab
```

---

## Step 4: Fixed Backoff

Fixed backoff uses constant delay:

```bash
# Apply fixed backoff experiment
kubectl apply -f experiments/03-fixed-backoff.yaml

# Watch retry timing
watch -n 5 'kubectl get chaosexperiment fixed-retry -n chaos-lab -o yaml | grep -E "retryCount|nextRetryTime|phase"'
```

**Retry timeline with 1m fixed delay:**
| Attempt | Delay | Cumulative |
|---------|-------|------------|
| 1 | 1m | 1m |
| 2 | 1m | 2m |
| 3 | 1m | 3m |

Clean up:
```bash
kubectl delete chaosexperiment fixed-retry -n chaos-lab
```

---

## Step 5: Custom Retry Configuration

Tune retry settings for your needs:

```bash
# Apply custom configuration
kubectl apply -f experiments/04-custom-retry.yaml

# Check configuration
kubectl describe chaosexperiment custom-retry -n chaos-lab
```

**Guidelines for tuning:**

| Scenario | maxRetries | retryBackoff | retryDelay |
|----------|------------|--------------|------------|
| Fast recovery expected | 3 | fixed | 10s |
| Infrastructure issues | 5 | exponential | 30s |
| Critical operation | 10 | exponential | 1m |
| Quick test | 1 | fixed | 5s |

Clean up:
```bash
kubectl delete chaosexperiment custom-retry -n chaos-lab
```

---

## Step 6: Retry with Scheduling

Combine retries with scheduled experiments:

```bash
# Apply scheduled experiment with retries
kubectl apply -f experiments/05-scheduled-with-retry.yaml

# Check status
kubectl describe chaosexperiment scheduled-retry -n chaos-lab
```

**Behavior:**
1. Experiment triggers on schedule
2. If it fails, retries with configured backoff
3. If all retries fail, waits for next scheduled time
4. Retry count resets for each scheduled run

Clean up:
```bash
kubectl delete chaosexperiment scheduled-retry -n chaos-lab
```

---

## Step 7: Understanding Failure States

After maximum retries exhausted:

```yaml
status:
  phase: Failed
  retryCount: 5
  lastError: "failed to execute chaos: no matching pods found"
  message: "Maximum retries (5) exceeded"
```

**Failure recovery:**
- Fix the underlying issue (selector, permissions, etc.)
- Delete and recreate the experiment
- Or update the experiment spec to trigger reconciliation

---

## Step 8: Cleanup

```bash
make teardown
```

---

## What You Learned

- `maxRetries` controls how many times to retry (0-10)
- `retryBackoff: exponential` doubles delay each retry
- `retryBackoff: fixed` uses constant delay
- `retryDelay` sets initial delay between retries
- Status tracks retryCount, lastError, nextRetryTime
- Maximum retries exceeded marks experiment as Failed

## Best Practices

1. **Start with exponential backoff** - handles most transient issues
2. **Set reasonable maxRetries** - 3-5 for most cases
3. **Use fixed backoff for predictable failures** - known recovery time
4. **Monitor retry metrics** - track patterns in failures
5. **Don't set maxRetries too high** - can mask real issues

## Next Steps

- **Lab 07**: Monitor chaos with Prometheus and Grafana
- **Lab 08**: Advanced multi-experiment scenarios

## Troubleshooting

**Retries not happening?**
- Check if experiment reached "Failed" state
- Verify maxRetries > 0
- Check operator logs for errors

**Backoff times seem wrong?**
- Exponential max is 10 minutes
- Fixed uses exact retryDelay value
- Time is calculated from failure, not last attempt

**Experiment stuck in retry loop?**
- Check lastError for root cause
- Verify selector matches pods
- Ensure operator has required permissions