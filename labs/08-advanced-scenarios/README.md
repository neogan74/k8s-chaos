# Lab 08: Advanced Scenarios

## Objectives
After completing this lab, you will be able to:
- [ ] Design multi-experiment chaos scenarios
- [ ] Simulate realistic failure modes
- [ ] Implement game day exercises
- [ ] Create chaos experiments for microservices
- [ ] Use advanced selectors for targeted chaos

## Prerequisites
- Completed Labs 01-07
- k8s-chaos operator installed and running
- Familiarity with all chaos actions and safety features

## Lab Duration
Estimated time: 40-45 minutes

---

## Overview: Advanced Chaos Scenarios

Real-world chaos engineering goes beyond single experiments:

| Scenario Type | Description | Example |
|---------------|-------------|---------|
| **Cascading Failures** | Multiple simultaneous issues | API + database pressure |
| **Game Day** | Planned chaos with teams | Monthly resilience testing |
| **Rolling Chaos** | Gradual, phased attacks | Zone-by-zone failures |
| **Microservices** | Service mesh chaos | Circuit breaker testing |

---

## Step 1: Setup Lab Environment

```bash
cd labs/08-advanced-scenarios
make setup
```

This creates a realistic microservices environment:
- Frontend (nginx) - 3 replicas
- API (nginx simulating API) - 5 replicas
- Database (nginx simulating DB) - 3 replicas
- Cache (nginx simulating Redis) - 2 replicas

Verify:
```bash
kubectl get pods -n chaos-lab -o wide
kubectl get svc -n chaos-lab
```

---

## Scenario 1: Cascading Failure Simulation

Simulate multiple component failures simultaneously.

### The Scenario
Your production system experiences:
1. Network latency to database
2. High CPU on API servers
3. Random cache pod failures

### Execute

```bash
# Apply all cascading experiments at once
kubectl apply -f scenarios/01-cascading-failure/

# Watch the chaos unfold
watch kubectl get pods -n chaos-lab -o wide

# Monitor experiments
kubectl get chaosexperiments -n chaos-lab
```

### What to Observe
- How does the frontend handle slow API responses?
- Do retries cascade into more load?
- Are circuit breakers triggered?

### Clean Up
```bash
kubectl delete chaosexperiments -n chaos-lab -l scenario=cascading
```

---

## Scenario 2: Rolling Zone Failure

Simulate availability zone failures progressively.

### The Scenario
Cloud provider has issues in multiple zones:
1. First, Zone A has network problems
2. Then, Zone B loses some instances
3. Finally, Zone C experiences CPU pressure

### Execute

```bash
# Phase 1: Zone A issues
kubectl apply -f scenarios/02-rolling-zone/phase1-zone-a.yaml
sleep 30

# Phase 2: Zone B issues
kubectl apply -f scenarios/02-rolling-zone/phase2-zone-b.yaml
sleep 30

# Phase 3: Zone C issues
kubectl apply -f scenarios/02-rolling-zone/phase3-zone-c.yaml
```

### Monitor
```bash
# Watch pod distribution and health
kubectl get pods -n chaos-lab -L zone -w

# Check experiment progression
kubectl get chaosexperiments -n chaos-lab -o wide
```

### Clean Up
```bash
kubectl delete chaosexperiments -n chaos-lab -l scenario=zone-failure
```

---

## Scenario 3: Game Day Exercise

Structured chaos engineering session with your team.

### Preparation (Before Game Day)

1. **Define Hypothesis**
   - "Our system should maintain 99% availability when 30% of API pods fail"

2. **Set Boundaries**
   ```yaml
   maxPercentage: 30
   experimentDuration: "15m"
   dryRun: true  # Start with preview
   ```

3. **Notify Teams**
   - SRE team on standby
   - Application owners informed
   - Incident channel ready

### Execute Game Day

```bash
# Step 1: Dry run to preview impact
kubectl apply -f scenarios/03-game-day/01-preview.yaml
kubectl get chaosexperiment game-day-preview -n chaos-lab -o yaml | grep -A5 status:

# Step 2: Execute actual chaos (after team approval)
kubectl apply -f scenarios/03-game-day/02-execute.yaml

# Step 3: Monitor for 15 minutes
watch -n 10 'kubectl get pods -n chaos-lab && echo "---" && kubectl get chaosexperiments -n chaos-lab'

# Step 4: Experiment auto-stops after experimentDuration
```

### Post-Game Day
- Review metrics in Grafana
- Check experiment history
- Document findings
- Create action items

### Clean Up
```bash
kubectl delete chaosexperiments -n chaos-lab -l scenario=game-day
```

---

## Scenario 4: Microservices Circuit Breaker Testing

Test how services handle downstream failures.

### The Scenario
1. Database becomes slow (simulating high latency)
2. API should timeout and open circuit breaker
3. Frontend should show degraded but functional service

### Execute

```bash
# Apply database latency
kubectl apply -f scenarios/04-circuit-breaker/database-slow.yaml

# Watch API behavior (would normally check circuit breaker metrics)
kubectl get pods -n chaos-lab -l tier=api -w
```

### Expected Behavior
- Initial database requests timeout
- Circuit breaker opens after threshold
- API returns fallback responses
- System recovers when circuit closes

### Clean Up
```bash
kubectl delete chaosexperiments -n chaos-lab -l scenario=circuit-breaker
```

---

## Scenario 5: Blast Radius Containment

Demonstrate how safety features limit impact.

### The Scenario
Attempt aggressive chaos with safety limits:
- Try to kill 50% of pods
- maxPercentage limits to 20%
- Production protection blocks expansion

### Execute

```bash
# This should be rejected (50% > 20%)
kubectl apply -f scenarios/05-blast-radius/aggressive-limited.yaml 2>&1 || echo "Rejected as expected!"

# This succeeds (within limits)
kubectl apply -f scenarios/05-blast-radius/conservative.yaml

# Check what happened
kubectl get chaosexperiments -n chaos-lab -l scenario=blast-radius
```

### Clean Up
```bash
kubectl delete chaosexperiments -n chaos-lab -l scenario=blast-radius
```

---

## Scenario 6: Full Stack Resilience Test

Comprehensive test of entire application stack.

### Execute Full Scenario

```bash
# Run full resilience scenario (takes ~10 minutes)
make run-full-scenario
```

This executes:
1. Frontend pod failures
2. API network latency
3. Database CPU stress
4. Cache pod kills
5. All with safety limits and auto-stop

### Monitor

```bash
# In separate terminals:

# Terminal 1: Watch pods
watch kubectl get pods -n chaos-lab

# Terminal 2: Watch experiments
watch kubectl get chaosexperiments -n chaos-lab

# Terminal 3: Check metrics (if Prometheus deployed)
curl -s http://localhost:8080/metrics | grep chaos
```

---

## Step 2: Design Your Own Scenario

Create a custom chaos scenario for your needs:

### Template
```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: custom-scenario-step1
  namespace: chaos-lab
  labels:
    scenario: custom
    phase: "1"
spec:
  action: <choose-action>
  namespace: chaos-lab
  selector:
    <your-labels>
  count: 1

  # Safety first
  dryRun: true
  maxPercentage: 25

  # Timing
  experimentDuration: "5m"
```

### Steps to Design
1. Define your hypothesis
2. Identify target components
3. Choose appropriate actions
4. Set safety limits
5. Plan rollback
6. Document expected outcomes

---

## Step 3: Cleanup

```bash
make teardown
```

---

## What You Learned

- Multi-experiment scenarios simulate realistic failures
- Game day exercises require preparation and team coordination
- Safety features prevent uncontrolled blast radius
- Cascading failures test system resilience holistically
- Circuit breaker testing validates fault tolerance patterns

## Best Practices for Advanced Scenarios

1. **Always start with dry-run** - Preview impact before execution
2. **Set experiment duration** - Auto-stop prevents runaway chaos
3. **Use labels consistently** - Enable easy cleanup and tracking
4. **Document hypotheses** - Know what you're testing
5. **Have rollback plan** - Know how to recover
6. **Monitor actively** - Watch metrics and logs during chaos
7. **Review history** - Learn from past experiments

## Next Steps

- Create scenarios specific to your applications
- Integrate chaos into CI/CD pipelines
- Establish regular game day exercises
- Build runbooks from chaos learnings

## Troubleshooting

**Experiments interfering with each other?**
- Use different selectors or namespaces
- Stagger experiment start times
- Set lower counts for concurrent experiments

**System overwhelmed by chaos?**
- Reduce count values
- Lower maxPercentage limits
- Increase time between phases
- Use experimentDuration for auto-stop

**Can't clean up experiments?**
- Use label selectors: `kubectl delete chaosexperiments -l scenario=<name>`
- Delete by namespace: `kubectl delete chaosexperiments -n chaos-lab --all`