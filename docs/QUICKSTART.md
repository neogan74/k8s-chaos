# Quick Start Guide

Get k8s-chaos running in 5 minutes! 🚀

## One-Command Setup

```bash
make dev-setup
```

This creates everything you need:
- ✅ Kind cluster with 3 nodes
- ✅ CRDs installed
- ✅ Demo nginx deployment (5 replicas)
- ✅ Development tools

## Run Your First Chaos Experiment

### 1. Start the Controller
```bash
# In terminal 1
make dev-run
```

### 2. Run Chaos Experiment
```bash
# In terminal 2
make demo-run
```

### 3. Watch the Chaos
```bash
make demo-watch
```

You'll see pods being terminated and recreated! 💥

## What Just Happened?

1. **ChaosExperiment** resource was created
2. **Controller** found nginx pods with `app=nginx` label
3. **2 random pods** were selected and deleted
4. **Kubernetes** automatically recreated them
5. **Experiment continues** every minute

## Next Steps

### View Status
```bash
make demo-status
```

### Stop Chaos
```bash
make demo-stop
```

### Try Different Experiments
```bash
# Kill multiple pods
kubectl apply -f config/samples/chaos_v1alpha1_chaosexperiment_multiple.yaml

# Target specific pods
kubectl apply -f config/samples/chaos_v1alpha1_chaosexperiment_stateful.yaml
```

### Clean Up
```bash
make dev-clean
```

## Commands Cheat Sheet

| What You Want | Command |
|---------------|---------|
| 🚀 Set up everything | `make dev-setup` |
| 🏃 Run controller | `make dev-run` |
| 💥 Start chaos | `make demo-run` |
| 👀 Watch chaos | `make demo-watch` |
| 📊 Check status | `make demo-status` |
| 🛑 Stop chaos | `make demo-stop` |
| 🧹 Clean up | `make dev-clean` |

## Understanding the Experiment

The demo creates this ChaosExperiment:
```yaml
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: nginx-chaos-demo
  namespace: chaos-demo
spec:
  action: "pod-kill"      # What to do
  namespace: "chaos-demo" # Where to do it
  selector:               # Which pods
    app: nginx
    environment: demo
  count: 2               # How many pods
```

## Safety First! 🛡️

- Experiments run in isolated `chaos-demo` namespace
- Only affects demo nginx pods
- Kubernetes automatically recovers
- Easy to stop with `make demo-stop`

## Troubleshooting

### Controller won't start?
```bash
make dev-status  # Check what's missing
```

### No chaos happening?
```bash
# Check if pods match selector
kubectl get pods -l app=nginx -n chaos-demo

# Check experiment status
kubectl describe chaosexperiment nginx-chaos-demo -n chaos-demo
```

### Need help?
- 📖 Read [DEVELOPMENT.md](DEVELOPMENT.md) for detailed guide
- 📁 Check `config/samples/` for more examples
- 🔍 Look at controller logs when running `make dev-run`

## Video Demos & Tutorials

### Creating Your Own Demo Video

Want to create a demo video? Here's what to show:

1. **Environment Setup** (1-2 minutes)
   ```bash
   make dev-setup
   kubectl get nodes  # Show cluster is ready
   kubectl get pods -n chaos-demo  # Show demo pods
   ```

2. **Controller Start** (30 seconds)
   ```bash
   make dev-run  # Show controller starting
   ```

3. **Chaos in Action** (2-3 minutes)
   ```bash
   make demo-run      # Start chaos
   make demo-watch    # Show pods being killed and recreated
   make demo-status   # Show experiment status
   ```

4. **Different Experiments** (2 minutes)
   ```bash
   # Try different chaos actions
   kubectl apply -f config/samples/chaos_v1alpha1_chaosexperiment_pod_delay.yaml
   kubectl apply -f config/samples/chaos_v1alpha1_chaosexperiment_cpu_stress.yaml
   ```

5. **Safety Features** (1-2 minutes)
   ```bash
   # Demonstrate dry-run mode
   kubectl apply -f config/samples/chaos_v1alpha1_chaosexperiment_safety_demo.yaml
   ```

6. **Metrics & Dashboards** (1 minute)
   - Show Prometheus metrics at `http://localhost:8080/metrics`
   - Display Grafana dashboards (if installed)

### Screen Recording Tips

- Use **asciinema** for terminal recording: `asciinema rec demo.cast`
- Or use **OBS Studio** for full-screen recording
- Keep videos **under 10 minutes**
- Add **narration or captions** explaining what's happening
- Include **timestamps** in descriptions

### Demo Scenarios

**Scenario 1: Basic Pod Kill** (Great for beginners)
```bash
make dev-setup && make dev-run
# In another terminal
make demo-run && make demo-watch
```

**Scenario 2: Network Latency** (Shows real-world chaos)
```bash
kubectl apply -f config/samples/chaos_v1alpha1_chaosexperiment_pod_delay.yaml
kubectl exec -it <pod-name> -n chaos-demo -- ping google.com
# Show increased latency
```

**Scenario 3: CPU Stress Testing** (Resource chaos)
```bash
kubectl apply -f config/samples/chaos_v1alpha1_chaosexperiment_cpu_stress.yaml
kubectl top pods -n chaos-demo
# Show CPU usage spike
```

**Scenario 4: Safety in Production** (Best practices)
```bash
# Show dry-run mode
kubectl apply -f config/samples/chaos_v1alpha1_chaosexperiment_safety_demo.yaml
kubectl describe chaosexperiment safety-demo -n chaos-testing
# Show it previews without executing
```

## Next Steps

Ready to cause some controlled chaos? 😈

```bash
make dev-setup && make dev-run
```

### Learn More

- **[Installation Guide](INSTALLATION.md)**: Production deployment
- **[Getting Started Tutorial](GETTING-STARTED.md)**: Complete walkthrough
- **[Architecture Overview](ARCHITECTURE.md)**: System design
- **[Contributing Guide](../CONTRIBUTING.md)**: Join the project
- **[Best Practices](BEST-PRACTICES.md)**: Safety-first principles
- **[Real-World Scenarios](SCENARIOS.md)**: 13 ready-to-use examples