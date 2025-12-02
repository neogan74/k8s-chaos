# Real-World Chaos Engineering Scenarios

This guide provides practical, ready-to-use chaos engineering scenarios for common use cases.

## Table of Contents

- [Web Application Scenarios](#web-application-scenarios)
- [Microservices Scenarios](#microservices-scenarios)
- [Database Scenarios](#database-scenarios)
- [Infrastructure Scenarios](#infrastructure-scenarios)
- [CI/CD & Deployment Scenarios](#cicd--deployment-scenarios)
- [Advanced Scenarios](#advanced-scenarios)

---

## Web Application Scenarios

### Scenario 1: Test Deployment Rollout Resilience

**Goal:** Verify your application can handle pod restarts during deployment

**Hypothesis:** When pods are killed during deployment, the rollout completes successfully with zero downtime

**Setup:**
```yaml
# Deploy a web app with rolling update strategy
apiVersion: apps/v1
kind: Deployment
metadata:
  name: webapp
  namespace: production
spec:
  replicas: 10
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 2
      maxSurge: 2
  selector:
    matchLabels:
      app: webapp
  template:
    metadata:
      labels:
        app: webapp
    spec:
      containers:
      - name: webapp
        image: nginx:1.25
        readinessProbe:
          httpGet:
            path: /health
            port: 80
          initialDelaySeconds: 5
          periodSeconds: 5
---
# Service
apiVersion: v1
kind: Service
metadata:
  name: webapp
  namespace: production
spec:
  selector:
    app: webapp
  ports:
  - port: 80
    targetPort: 80
```

**Chaos Experiment:**
```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: rollout-resilience-test
  namespace: production
spec:
  action: pod-kill
  namespace: production
  selector:
    app: webapp
  count: 2                    # Kill 2 pods at a time
  maxPercentage: 30           # Never more than 30%
  experimentDuration: "10m"   # Run during deployment window
  allowProduction: true
```

**Validation:**
```bash
# Start chaos
kubectl apply -f rollout-experiment.yaml

# Trigger rollout in another terminal
kubectl set image deployment/webapp -n production webapp=nginx:1.26

# Monitor
watch -n 1 'kubectl get pods -n production -l app=webapp'
watch -n 1 'kubectl rollout status deployment/webapp -n production'

# Check for zero downtime
curl -s https://webapp.example.com/health
```

**Success Criteria:**
- ✅ Rollout completes successfully
- ✅ No 5xx errors during rollout
- ✅ Latency stays under SLO (e.g., p95 < 500ms)

---

### Scenario 2: Simulate Network Latency Spike

**Goal:** Test how your application handles slow backend responses

**Hypothesis:** Application degrades gracefully with timeouts and retries when network latency increases to 200ms

**Chaos Experiment:**
```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: network-latency-test
  namespace: production
spec:
  action: pod-delay
  namespace: production
  selector:
    app: api-backend
  count: 3
  duration: "200ms"            # Add 200ms latency
  maxPercentage: 50            # Affect up to 50% of pods
  experimentDuration: "5m"
  allowProduction: true
```

**Monitor:**
```promql
# Request duration
histogram_quantile(0.95,
  rate(http_request_duration_seconds_bucket{service="frontend"}[1m])
)

# Timeout errors
rate(http_requests_total{status="504"}[1m])

# Retry attempts
rate(http_retries_total[1m])
```

**Expected Behavior:**
- Frontend shows increased latency
- Timeouts trigger retry logic
- Circuit breaker prevents cascading failures
- User sees graceful degradation, not errors

---

### Scenario 3: Test Autoscaling Under Stress

**Goal:** Verify HPA scales up when pods are under CPU stress

**Setup:**
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: webapp-hpa
  namespace: production
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: webapp
  minReplicas: 5
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

**Chaos Experiment:**
```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: autoscaling-test
  namespace: production
spec:
  action: pod-cpu-stress
  namespace: production
  selector:
    app: webapp
  count: 3
  cpuLoad: 90                  # 90% CPU load
  cpuWorkers: 2                # 2 CPU workers
  duration: "10m"              # Stress for 10 minutes
  experimentDuration: "15m"    # Run experiment for 15 min
  maxPercentage: 50
  allowProduction: true
```

**Validation:**
```bash
# Watch HPA
watch -n 5 'kubectl get hpa webapp-hpa -n production'

# Watch pod count
watch -n 5 'kubectl get pods -n production -l app=webapp --no-headers | wc -l'

# Check CPU usage
kubectl top pods -n production -l app=webapp
```

**Success Criteria:**
- ✅ HPA scales from 5 → 15+ replicas
- ✅ Scaling happens within 2-3 minutes
- ✅ Application maintains SLO during scaling

---

## Microservices Scenarios

### Scenario 4: Test Service Mesh Resilience (Istio)

**Goal:** Verify Istio circuit breakers work when a microservice fails

**Setup:**
```yaml
# Istio DestinationRule with circuit breaker
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: payment-service
  namespace: production
spec:
  host: payment-service
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 100
      http:
        http1MaxPendingRequests: 10
        maxRequestsPerConnection: 2
    outlierDetection:
      consecutiveErrors: 3
      interval: 30s
      baseEjectionTime: 30s
```

**Chaos Experiment:**
```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: circuit-breaker-test
  namespace: production
spec:
  action: pod-failure           # Kill main process
  namespace: production
  selector:
    app: payment-service
  count: 2
  experimentDuration: "5m"
  maxPercentage: 40
  allowProduction: true
```

**Validation:**
```bash
# Check Istio circuit breaker metrics
kubectl exec -n istio-system <istio-pod> -- \
  curl -s localhost:15000/stats | grep payment-service | grep outlier_detection

# Monitor traffic
istioctl dashboard kiali
```

**Expected Behavior:**
- Failed pods ejected from load balancer pool
- Circuit breaker trips after 3 consecutive errors
- Traffic routed only to healthy pods
- Fallback responses returned to users

---

### Scenario 5: Test Distributed Tracing During Failures

**Goal:** Verify tracing works correctly when services fail

**Chaos Experiment:**
```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: tracing-test
  namespace: production
spec:
  action: pod-delay
  namespace: production
  selector:
    tier: backend
  count: 5
  duration: "500ms"            # Significant delay
  experimentDuration: "10m"
  maxPercentage: 30
  allowProduction: true
```

**Validation:**
```bash
# Open Jaeger UI
kubectl port-forward -n observability svc/jaeger-query 16686:16686

# Check traces
# Look for:
# - Correct span timing
# - Error spans marked
# - Full trace coverage
```

**Success Criteria:**
- ✅ All spans captured during chaos
- ✅ Slow spans clearly visible
- ✅ Error propagation tracked correctly

---

## Database Scenarios

### Scenario 6: Test Database Connection Pool Exhaustion

**Goal:** Verify application handles database connection failures gracefully

**Setup:**
```yaml
# Application with connection pooling
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
  namespace: production
data:
  database.properties: |
    db.pool.size=20
    db.pool.timeout=5s
    db.retry.attempts=3
    db.retry.backoff=exponential
```

**Chaos Experiment:**
```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: db-connection-test
  namespace: production
spec:
  action: pod-kill
  namespace: production
  selector:
    app: postgres
    role: replica              # Target read replicas only!
  count: 2
  experimentDuration: "5m"
  maxPercentage: 50
  allowProduction: true
```

**Validation:**
```bash
# Monitor connection pool
kubectl exec -n production <app-pod> -- \
  curl localhost:8080/metrics | grep db_connections

# Check error logs
kubectl logs -n production -l app=api --tail=100 | grep -i database
```

**Expected Behavior:**
- Application retries failed queries
- Connection pool rebalances to healthy replicas
- Circuit breaker prevents overwhelming database
- Users see slight latency increase, not errors

---

### Scenario 7: Test Read Replica Failure Handling

**Goal:** Verify application fails over from read replicas to primary

**Chaos Experiment:**
```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: replica-failover-test
  namespace: production
spec:
  action: pod-kill
  namespace: production
  selector:
    app: postgres
    role: replica
  count: 3                     # Kill all replicas
  experimentDuration: "10m"
  allowProduction: true

---
# Protect primary database
apiVersion: v1
kind: Pod
metadata:
  name: postgres-primary-0
  namespace: production
  labels:
    app: postgres
    role: primary
    chaos.gushchin.dev/exclude: "true"  # ← Protected!
spec:
  containers:
  - name: postgres
    image: postgres:15
```

**Success Criteria:**
- ✅ Read traffic fails over to primary
- ✅ Write traffic unaffected
- ✅ No data loss
- ✅ Replicas rejoin when recovered

---

## Infrastructure Scenarios

### Scenario 8: Node Failure Simulation

**Goal:** Test workload migration when nodes fail

**Setup:**
```yaml
# Ensure PodDisruptionBudget exists
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: webapp-pdb
  namespace: production
spec:
  minAvailable: 3
  selector:
    matchLabels:
      app: webapp
```

**Chaos Experiment:**
```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: node-failure-test
  namespace: production
spec:
  action: node-drain
  namespace: production          # Not used for node-drain
  selector:
    kubernetes.io/role: worker
    environment: lab             # Target lab nodes only!
  count: 1
  experimentDuration: "15m"      # Auto-uncordon after 15 min
```

**Validation:**
```bash
# Watch pod migration
watch -n 2 'kubectl get pods -n production -o wide'

# Check node status
watch -n 2 'kubectl get nodes'

# Monitor PDB
kubectl get pdb -n production
```

**Success Criteria:**
- ✅ Pods evicted gracefully
- ✅ Pods rescheduled on healthy nodes
- ✅ PDB respected (min 3 pods always available)
- ✅ Node auto-uncordoned after 15 minutes

---

### Scenario 9: Multi-Zone Failure

**Goal:** Test application survives entire zone failure

**Chaos Experiment:**
```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: zone-failure-test
  namespace: production
spec:
  action: node-drain
  namespace: production
  selector:
    topology.kubernetes.io/zone: us-east-1a  # Drain one zone
  count: 10                                   # All nodes in zone
  experimentDuration: "30m"
```

**Prerequisites:**
- Pods spread across multiple zones
- Anti-affinity rules configured
- Cross-zone load balancing

**Success Criteria:**
- ✅ Traffic shifts to other zones
- ✅ No service disruption
- ✅ Autoscaling compensates in healthy zones

---

## CI/CD & Deployment Scenarios

### Scenario 10: Test Blue-Green Deployment Switchover

**Goal:** Verify blue-green deployment handles chaos during switchover

**Setup:**
```yaml
# Blue deployment (current)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: webapp-blue
  namespace: production
spec:
  replicas: 10
  selector:
    matchLabels:
      app: webapp
      version: blue
  template:
    metadata:
      labels:
        app: webapp
        version: blue
    spec:
      containers:
      - name: webapp
        image: webapp:v1.0

---
# Green deployment (new)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: webapp-green
  namespace: production
spec:
  replicas: 10
  selector:
    matchLabels:
      app: webapp
      version: green
  template:
    metadata:
      labels:
        app: webapp
        version: green
    spec:
      containers:
      - name: webapp
        image: webapp:v2.0
```

**Chaos Experiment:**
```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: blue-green-chaos
  namespace: production
spec:
  action: pod-kill
  namespace: production
  selector:
    app: webapp
    # No version label - affects both blue and green!
  count: 3
  experimentDuration: "5m"
  maxPercentage: 20
  allowProduction: true
```

**During Switchover:**
```bash
# Run chaos
kubectl apply -f blue-green-chaos.yaml

# Switch traffic from blue to green
kubectl patch service webapp -n production \
  -p '{"spec":{"selector":{"version":"green"}}}'

# Monitor
watch -n 1 'kubectl get pods -n production -l app=webapp'
```

**Success Criteria:**
- ✅ Switchover completes successfully
- ✅ Zero downtime during transition
- ✅ Rollback works if issues detected

---

## Advanced Scenarios

### Scenario 11: Scheduled Maintenance Window Testing

**Goal:** Simulate regular maintenance activities

**Chaos Experiment:**
```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: maintenance-simulation
  namespace: production
spec:
  action: node-drain
  namespace: production
  selector:
    maintenance-window: "true"
  count: 2
  schedule: "0 2 * * 0"        # Every Sunday at 2 AM
  experimentDuration: "2h"     # 2-hour maintenance window
```

**Use Case:**
- Test weekly maintenance procedures
- Verify application handles planned downtime
- Practice runbooks automatically

---

### Scenario 12: Cascading Failure Test

**Goal:** Test if one service failure cascades to others

**Setup:**
```yaml
# Frontend → API → Database architecture
# We'll fail the API tier
```

**Chaos Experiments:**
```yaml
# Start with backend
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: cascade-test-phase1
  namespace: production
spec:
  action: pod-failure
  namespace: production
  selector:
    tier: api
  count: 5
  maxPercentage: 60            # Significant failure
  experimentDuration: "10m"
  allowProduction: true

---
# Monitor if it cascades to frontend
# (No experiment - just observe)
```

**Monitor:**
```bash
# API tier
kubectl top pods -n production -l tier=api

# Frontend tier (should NOT be affected)
kubectl top pods -n production -l tier=frontend

# Check circuit breakers
kubectl logs -n production -l tier=frontend | grep "circuit.*open"
```

**Success Criteria:**
- ✅ Frontend circuit breakers open
- ✅ Frontend shows fallback responses
- ✅ Frontend does NOT crash or restart
- ✅ Database tier unaffected

---

### Scenario 13: Game Day - Complete System Test

**Goal:** Comprehensive resilience testing

**Timeline:**
```
09:00 - Team assembles, reviews plan
09:15 - Start monitoring baseline
09:30 - Phase 1: Kill 20% of API pods
10:00 - Phase 2: Add 200ms network latency
10:30 - Phase 3: Drain one node
11:00 - Phase 4: CPU stress on database replicas
11:30 - Review results, document findings
12:00 - Debrief and retrospective
```

**Experiment Manifests:**
```yaml
# Phase 1
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: gameday-phase1-pod-kill
  namespace: production
spec:
  action: pod-kill
  namespace: production
  selector:
    tier: api
  count: 3
  maxPercentage: 20
  experimentDuration: "30m"
  allowProduction: true

---
# Phase 2
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: gameday-phase2-latency
  namespace: production
spec:
  action: pod-delay
  namespace: production
  selector:
    tier: api
  count: 5
  duration: "200ms"
  maxPercentage: 30
  experimentDuration: "30m"
  allowProduction: true

---
# Phase 3
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: gameday-phase3-node-drain
  namespace: production
spec:
  action: node-drain
  namespace: production
  selector:
    kubernetes.io/role: worker
    zone: us-east-1c
  count: 1
  experimentDuration: "30m"

---
# Phase 4
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: gameday-phase4-cpu-stress
  namespace: production
spec:
  action: pod-cpu-stress
  namespace: production
  selector:
    app: postgres
    role: replica
  count: 2
  cpuLoad: 80
  duration: "30m"
  experimentDuration: "30m"
  allowProduction: true
```

**Runbook:**
1. Monitor dashboards continuously
2. Document every observation
3. Stop immediately if SLOs violated
4. Capture metrics before/during/after each phase
5. Hold team retrospective

---

## Experiment Templates

### Quick Start Template

```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: my-experiment
  namespace: my-namespace
spec:
  # REQUIRED
  action: pod-kill                    # or pod-delay, pod-cpu-stress, etc.
  namespace: target-namespace
  selector:
    app: my-app

  # SAFETY (highly recommended)
  dryRun: false                       # Start with true!
  maxPercentage: 20                   # Limit blast radius
  experimentDuration: "10m"           # Auto-stop
  count: 1                            # Start small

  # OPTIONAL
  schedule: ""                        # Cron schedule
  allowProduction: false              # Required for prod

  # RETRY
  maxRetries: 3
  retryDelay: "30s"
  retryBackoff: "exponential"
```

---

## Additional Resources

- **Getting Started**: [GETTING-STARTED.md](GETTING-STARTED.md)
- **Best Practices**: [BEST-PRACTICES.md](BEST-PRACTICES.md)
- **Troubleshooting**: [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
- **API Documentation**: [API.md](API.md)
- **Hands-on Labs**: [../labs/README.md](../labs/README.md)

Happy testing! Remember: chaos engineering is about building confidence, not just breaking things. 🚀