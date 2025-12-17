# Lab 05: Scheduling & Duration

## Objectives
After completing this lab, you will be able to:
- [ ] Schedule recurring chaos with cron expressions
- [ ] Use predefined schedules (@hourly, @daily, etc.)
- [ ] Control experiment duration with auto-stop
- [ ] Combine scheduling with duration limits
- [ ] View schedule status (lastScheduledTime, nextScheduledTime)

## Prerequisites
- Completed Labs 01-04
- k8s-chaos operator installed and running
- kubectl configured

## Lab Duration
Estimated time: 20-25 minutes

---

## Overview: Scheduling & Duration

Two complementary features control when and how long chaos runs:

| Feature | Purpose | Format |
|---------|---------|--------|
| `schedule` | When to run | Cron expression or predefined |
| `experimentDuration` | How long to run | Duration string (e.g., "5m") |

---

## Step 1: Setup Lab Environment

```bash
cd labs/05-scheduling-duration
make setup
```

Verify:
```bash
kubectl get pods -n chaos-lab
```

---

## Step 2: Understanding Cron Schedules

### Cron Format
```
┌───────────── minute (0 - 59)
│ ┌───────────── hour (0 - 23)
│ │ ┌───────────── day of month (1 - 31)
│ │ │ ┌───────────── month (1 - 12)
│ │ │ │ ┌───────────── day of week (0 - 6) (Sunday = 0)
│ │ │ │ │
* * * * *
```

### Common Examples
| Expression | Meaning |
|------------|---------|
| `*/5 * * * *` | Every 5 minutes |
| `0 * * * *` | Every hour (at minute 0) |
| `0 2 * * *` | Daily at 2 AM |
| `0 9 * * 1` | Every Monday at 9 AM |
| `*/15 9-17 * * 1-5` | Every 15 min, 9-5 PM, Mon-Fri |

### Predefined Schedules
| Shorthand | Equivalent |
|-----------|------------|
| `@hourly` | `0 * * * *` |
| `@daily` | `0 0 * * *` |
| `@weekly` | `0 0 * * 0` |
| `@monthly` | `0 0 1 * *` |
| `@yearly` | `0 0 1 1 *` |

---

## Step 3: Create a Scheduled Experiment

Let's create an experiment that runs every 2 minutes:

```bash
# Apply the scheduled experiment
kubectl apply -f experiments/01-every-2-minutes.yaml

# Watch the experiment status
watch -n 5 'kubectl get chaosexperiment scheduled-every-2min -n chaos-lab -o yaml | grep -A10 status:'
```

**Key status fields:**
- `lastScheduledTime`: When it last ran
- `nextScheduledTime`: When it will run next
- `phase`: Current state

Watch for ~3-4 minutes to see it trigger twice.

Clean up:
```bash
kubectl delete chaosexperiment scheduled-every-2min -n chaos-lab
```

---

## Step 4: Using Predefined Schedules

Predefined schedules are easier to read:

```bash
# Review the experiment
cat experiments/02-hourly-schedule.yaml

# Apply (won't run immediately - waits for next hour)
kubectl apply -f experiments/02-hourly-schedule.yaml

# Check next scheduled time
kubectl get chaosexperiment scheduled-hourly -n chaos-lab -o jsonpath='{.status.nextScheduledTime}'
```

Clean up:
```bash
kubectl delete chaosexperiment scheduled-hourly -n chaos-lab
```

---

## Step 5: Experiment Duration Control

`experimentDuration` auto-stops the experiment after a period:

```bash
# Apply experiment with 2-minute duration
kubectl apply -f experiments/03-duration-limit.yaml

# Watch the experiment
watch kubectl get chaosexperiment duration-limited -n chaos-lab

# Watch pods being killed (for 2 minutes)
kubectl get pods -n chaos-lab -w
```

After 2 minutes, the experiment automatically:
1. Stops executing chaos
2. Sets phase to "Completed"
3. Updates status message

Clean up:
```bash
kubectl delete chaosexperiment duration-limited -n chaos-lab
```

---

## Step 6: Combining Schedule + Duration

Use both for recurring, time-boxed chaos:

```bash
# Review the combined experiment
cat experiments/04-scheduled-with-duration.yaml

# Apply it
kubectl apply -f experiments/04-scheduled-with-duration.yaml
```

**Behavior:**
1. Experiment triggers at scheduled time
2. Runs for `experimentDuration`
3. Auto-stops
4. Waits for next scheduled time
5. Repeats

Check status:
```bash
kubectl describe chaosexperiment scheduled-duration -n chaos-lab
```

Clean up:
```bash
kubectl delete chaosexperiment scheduled-duration -n chaos-lab
```

---

## Step 7: Business Hours Chaos

Real-world scenario: run chaos only during business hours:

```bash
# Review - runs every 15 min, 9 AM to 5 PM, Mon-Fri
cat experiments/05-business-hours.yaml

# Apply it
kubectl apply -f experiments/05-business-hours.yaml

# Check when it will next run
kubectl get chaosexperiment business-hours-chaos -n chaos-lab -o yaml | grep -E "nextScheduledTime|lastScheduledTime"
```

This is useful for:
- Running chaos when teams are available to respond
- Avoiding chaos during maintenance windows
- Aligning with SRE on-call schedules

Clean up:
```bash
kubectl delete chaosexperiment business-hours-chaos -n chaos-lab
```

---

## Step 8: View All Scheduled Experiments

```bash
# List all scheduled experiments with their next run time
kubectl get chaosexperiments -n chaos-lab -o custom-columns=\
NAME:.metadata.name,\
SCHEDULE:.spec.schedule,\
NEXT:.status.nextScheduledTime,\
LAST:.status.lastScheduledTime

# Or for all namespaces
kubectl get chaosexperiments -A -o wide
```

---

## Step 9: Cleanup

```bash
make teardown
```

---

## What You Learned

- Cron expressions define when chaos runs (e.g., `*/5 * * * *`)
- Predefined schedules simplify common patterns (@hourly, @daily)
- `experimentDuration` auto-stops experiments after a period
- Combining schedule + duration enables recurring, time-boxed chaos
- Status tracks lastScheduledTime and nextScheduledTime

## Next Steps

- **Lab 06**: Configure retry logic for resilient experiments
- **Lab 07**: Monitor chaos with Prometheus and Grafana
- **Lab 08**: Advanced multi-experiment scenarios

## Troubleshooting

**Experiment not triggering on schedule?**
- Check system time matches expected timezone
- Verify cron expression is correct
- Review operator logs for errors

**experimentDuration not stopping?**
- Duration must be valid format (e.g., "5m", "1h30m")
- Check experiment status for errors
- Ensure operator has correct permissions

**Missing nextScheduledTime?**
- Experiment might have run once already
- Check if schedule field is present
- Verify cron expression syntax