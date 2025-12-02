# Best Practices for Chaos Engineering

This guide provides recommendations for safely and effectively practicing chaos engineering with k8s-chaos.

## Table of Contents

- [Safety First](#safety-first)
- [Progressive Adoption](#progressive-adoption)
- [Production Readiness](#production-readiness)
- [Experiment Design](#experiment-design)
- [Monitoring & Observability](#monitoring--observability)
- [Team Collaboration](#team-collaboration)
- [Common Pitfalls](#common-pitfalls)

---

## Safety First

Chaos engineering is about learning, not breaking things. Always prioritize safety.

### 1. Always Start with Dry-Run

**✅ DO:**
```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: test-resilience
spec:
  action: pod-kill
  namespace: production
  selector:
    app: api-service
  count: 2
  dryRun: true  # ← Start here!
```

**Why?** Dry-run shows exactly which resources will be affected without any actual impact.

**Workflow:**
```bash
# 1. Apply with dry-run
kubectl apply -f experiment.yaml

# 2. Check the preview
kubectl get chaosexperiment test-resilience -o jsonpath='{.status.message}'
# Output: "DRY RUN: Would delete 2 pod(s): [api-service-abc, api-service-def]"

# 3. If happy, remove dryRun and apply again
```

### 2. Use maxPercentage to Limit Blast Radius

**✅ DO:**
```yaml
spec:
  action: pod-kill
  count: 5
  maxPercentage: 30  # ← Never affect more than 30%
```

**Why?** Prevents accidentally affecting too many resources, especially when pod counts scale.

**Example:**
- 10 pods, maxPercentage: 30, count: 5 → **Rejected** (50% > 30%)
- 20 pods, maxPercentage: 30, count: 5 → **Allowed** (25% ≤ 30%)

### 3. Protect Critical Resources with Exclusions

**At the Pod Level:**
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: critical-db
  labels:
    chaos.gushchin.dev/exclude: "true"  # ← Never touch this pod
```

**At the Namespace Level:**
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: kube-system
  annotations:
    chaos.gushchin.dev/exclude: "true"  # ← Exclude entire namespace
```

**Examples of what to exclude:**
- Database primaries
- Control plane components
- Authentication services
- Payment processing systems

### 4. Require Explicit Production Approval

Production namespaces automatically require `allowProduction: true`:

**✅ DO:**
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: production
  labels:
    environment: production  # ← Marks as production
---
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: prod-test
  namespace: production
spec:
  action: pod-kill
  namespace: production
  selector:
    app: api
  allowProduction: true  # ← Explicit approval required!
```

**❌ DON'T:**
```yaml
# This will be REJECTED by the webhook
spec:
  action: pod-kill
  namespace: production
  # Missing: allowProduction: true
```

### 5. Use experimentDuration for Auto-Stop

**✅ DO:**
```yaml
spec:
  action: pod-kill
  experimentDuration: "10m"  # ← Auto-stop after 10 minutes
```

**Why?** Prevents experiments from running indefinitely if forgotten.

**Benefits:**
- Automatic cleanup (e.g., nodes auto-uncordon after drain)
- Predictable test windows
- Reduces risk of prolonged impact

---

## Progressive Adoption

Don't jump straight to production chaos. Follow a progressive path.

### Phase 1: Development Environment (Week 1-2)

**Goal:** Learn the tools and build confidence

**Actions:**
1. Install k8s-chaos in dev cluster
2. Use dry-run mode extensively
3. Test all chaos actions
4. Understand metrics and monitoring

**Experiments to try:**
- Simple pod-kill with 1 pod
- Network delay (50ms)
- CPU stress (30% load)
- Node drain with 1 worker

**Success Criteria:**
- Team comfortable with CRD syntax
- Can read metrics in Prometheus
- Understand experiment lifecycle

### Phase 2: Staging Environment (Week 3-4)

**Goal:** Test real application behavior

**Actions:**
1. Deploy identical setup to production
2. Run experiments during work hours
3. Validate monitoring alerts fire correctly
4. Practice incident response procedures

**Experiments to try:**
- Pod kill with maxPercentage: 20
- Network delay (100-200ms)
- CPU stress (60-80% load)
- Multiple simultaneous experiments

**Success Criteria:**
- Application handles failures gracefully
- Monitoring catches all issues
- Team can respond to alerts quickly

### Phase 3: Production (Week 5+)

**Goal:** Validate production resilience

**Actions:**
1. Start with lowest-risk experiments
2. Run during business hours (NOT at 3 AM!)
3. Have full team on standby
4. Document every experiment

**Experiments to start with:**
- Pod kill: count=1, maxPercentage=10
- Schedule during low-traffic periods
- Use experimentDuration: "5m"

**Success Criteria:**
- Zero customer impact
- All SLOs maintained
- Team confident in system resilience

### Game Days

Once comfortable, run coordinated "game days":

```yaml
# Game Day: Black Friday Simulation
# Goal: Test if we can handle increased load + failures

# Experiment 1: Kill pods during load test
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: gameday-pod-kill
spec:
  action: pod-kill
  namespace: production
  selector:
    app: api
  count: 2
  maxPercentage: 15
  experimentDuration: "30m"
  allowProduction: true

---
# Experiment 2: Add network latency
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: gameday-latency
spec:
  action: pod-delay
  namespace: production
  selector:
    app: api
  count: 3
  duration: "100ms"
  maxPercentage: 20
  experimentDuration: "30m"
  allowProduction: true
```

---

## Production Readiness

Before running chaos in production, ensure these requirements are met:

### ✅ Infrastructure Requirements

- [ ] **High Availability**: Multiple replicas for all services
- [ ] **Load Balancing**: Traffic distributed across instances
- [ ] **Auto-scaling**: HPA configured appropriately
- [ ] **Health Checks**: Liveness and readiness probes working
- [ ] **Resource Limits**: CPU/memory limits set correctly
- [ ] **Pod Disruption Budgets**: PDBs configured for critical services

Example PDB:
```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: api-pdb
spec:
  minAvailable: 2  # Always keep at least 2 pods running
  selector:
    matchLabels:
      app: api
```

### ✅ Monitoring Requirements

- [ ] **Prometheus**: Metrics collection working
- [ ] **Grafana**: Dashboards showing key metrics
- [ ] **Alerting**: Alerts configured for SLO violations
- [ ] **Logging**: Centralized log aggregation
- [ ] **Tracing**: Distributed tracing for debugging

### ✅ Process Requirements

- [ ] **Runbook**: Documented response procedures
- [ ] **On-Call**: Team available during experiments
- [ ] **Communication**: Stakeholders informed
- [ ] **Rollback Plan**: Quick way to stop chaos
- [ ] **Post-Mortem Template**: Ready to document findings

### ✅ Application Requirements

- [ ] **Graceful Degradation**: App handles partial failures
- [ ] **Retry Logic**: Transient failures retried automatically
- [ ] **Circuit Breakers**: Prevent cascade failures
- [ ] **Timeouts**: Reasonable timeouts configured
- [ ] **Idempotency**: Operations can be safely retried

---

## Experiment Design

Design experiments to answer specific questions.

### 1. Define a Hypothesis

**❌ BAD:** "Let's kill some pods and see what happens"

**✅ GOOD:**
```
Hypothesis: When we kill up to 30% of API pods, the service
remains available with latency under 500ms and zero errors.

Expected Outcome:
- P95 latency: < 500ms
- Error rate: 0%
- New pods: Ready in < 30s
```

### 2. Establish Steady State

Before chaos, know what "normal" looks like:

```bash
# Measure baseline
kubectl top pods -n production
curl https://api.example.com/health
```

Record:
- Request rate
- Latency (P50, P95, P99)
- Error rate
- Resource usage

### 3. Introduce Chaos

Apply the experiment:

```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: api-resilience-test
spec:
  action: pod-kill
  namespace: production
  selector:
    app: api
  count: 3
  maxPercentage: 30
  experimentDuration: "10m"
  allowProduction: true
```

### 4. Observe System Behavior

Monitor during experiment:
- Does traffic shift to healthy pods?
- Do new pods start quickly?
- Are there any errors?
- Do alerts fire appropriately?

### 5. Learn and Improve

Document findings:
```markdown
## Experiment Results

**Hypothesis:** ✅ CONFIRMED / ❌ REJECTED

**Observations:**
- Latency increased from 50ms → 200ms
- No errors observed
- New pods ready in 25s
- Alerts fired correctly

**Improvements Needed:**
1. Increase HPA maxReplicas from 10 → 15
2. Reduce container startup time
3. Add more aggressive readiness probe

**Next Experiment:**
Test with 50% pods killed
```

---

## Monitoring & Observability

You can't practice chaos engineering without proper monitoring.

### Essential Metrics to Track

**Application Metrics:**
```promql
# Request rate
rate(http_requests_total[5m])

# Latency
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Error rate
rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])
```

**Chaos Metrics:**
```promql
# Experiments running
chaosexperiment_active_experiments

# Success rate
rate(chaosexperiment_experiments_total{status="success"}[1h])
/ rate(chaosexperiment_experiments_total[1h])

# Resources affected
chaosexperiment_resources_affected
```

**Infrastructure Metrics:**
```promql
# Pod availability
kube_deployment_status_replicas_available
/ kube_deployment_spec_replicas

# Node status
kube_node_status_condition{condition="Ready",status="true"}
```

### Grafana Dashboard Best Practices

1. **Split screen view**: App metrics vs Chaos metrics
2. **Annotations**: Mark when experiments start/stop
3. **Variables**: Filter by namespace, action, experiment
4. **Alerts**: Visual indicators when SLOs violated

See [GRAFANA.md](GRAFANA.md) for pre-built dashboards.

---

## Team Collaboration

Chaos engineering is a team sport.

### 1. Define Roles

**Chaos Lead:**
- Designs experiments
- Runs the chaos
- Monitors dashboards

**Subject Matter Expert:**
- Understands the application
- Interprets results
- Suggests improvements

**Incident Commander:**
- Coordinates response
- Makes go/no-go decisions
- Communicates with stakeholders

### 2. Communication Protocol

**Before Experiment:**
```
To: #engineering-team
Subject: Chaos Experiment Starting - API Resilience Test

What: Testing API pod failures
When: Today 2:00 PM - 2:15 PM (15 min)
Impact: None expected (max 30% pods)
Monitor: https://grafana.company.com/chaos-dashboard
Contact: @chaos-lead in #incidents
```

**During Experiment:**
- Real-time updates in Slack/Teams
- Share dashboard links
- Document observations

**After Experiment:**
- Share results in team meeting
- Update runbooks with learnings
- Plan follow-up experiments

### 3. Blameless Culture

**✅ DO:**
- Celebrate finding weaknesses
- Focus on system improvements
- Share learnings openly

**❌ DON'T:**
- Blame developers for bugs found
- Hide failures
- Skip experiments due to fear

---

## Common Pitfalls

### 1. Running Chaos "Just Because"

**❌ Problem:** Random chaos without purpose

**✅ Solution:** Every experiment should answer a specific question

### 2. Only Testing in Production

**❌ Problem:** First chaos in prod → surprises

**✅ Solution:** Dev → Staging → Production progression

### 3. Setting and Forgetting

**❌ Problem:** Experiments run forever

**✅ Solution:** Always use `experimentDuration`

### 4. Ignoring Exclusions

**❌ Problem:** Critical pods get affected

**✅ Solution:** Label critical resources with exclusion labels

### 5. No Monitoring

**❌ Problem:** Can't tell if chaos causes issues

**✅ Solution:** Set up metrics before running experiments

### 6. Insufficient Retries/Timeouts

**❌ Problem:** App doesn't handle transient failures

**✅ Solution:** Implement proper retry logic and timeouts in your application

### 7. Running During Off-Hours

**❌ Problem:** Issues discovered at 3 AM

**✅ Solution:** Run chaos during business hours when team is available

---

## Quick Checklist

Before each experiment, verify:

### Pre-Experiment
- [ ] Hypothesis defined
- [ ] Baseline metrics recorded
- [ ] Team informed
- [ ] Dashboards ready
- [ ] Rollback plan prepared
- [ ] Dry-run executed
- [ ] maxPercentage configured
- [ ] experimentDuration set
- [ ] Critical resources excluded
- [ ] Production approval (if needed)

### During Experiment
- [ ] Monitoring dashboards
- [ ] Documenting observations
- [ ] Ready to stop if needed
- [ ] Communicating status

### Post-Experiment
- [ ] Hypothesis confirmed/rejected
- [ ] Results documented
- [ ] Improvements identified
- [ ] Team debriefed
- [ ] Next experiment planned

---

## Additional Resources

- **Getting Started**: [GETTING-STARTED.md](GETTING-STARTED.md)
- **Troubleshooting**: [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
- **Example Scenarios**: [SCENARIOS.md](SCENARIOS.md)
- **Metrics Guide**: [METRICS.md](METRICS.md)
- **API Documentation**: [API.md](API.md)

Remember: The goal is not to break things, but to build confidence in your system's resilience! 🛡️