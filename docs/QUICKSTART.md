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

Ready to cause some controlled chaos? 😈

```bash
make dev-setup && make dev-run
```