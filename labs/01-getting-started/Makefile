.PHONY: setup teardown help

help: ## Display this help
	@echo "Lab 01: Getting Started"
	@echo "======================="
	@echo ""
	@echo "Available commands:"
	@echo "  make setup     - Deploy demo application"
	@echo "  make teardown  - Clean up all resources"
	@echo "  make status    - Show current state"
	@echo ""
	@echo "Follow README.md for full lab instructions"

setup: ## Deploy the demo application
	@echo "📦 Setting up Lab 01..."
	kubectl create namespace chaos-demo --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -f setup/demo-app.yaml
	@echo "⏳ Waiting for pods to be ready..."
	kubectl wait --for=condition=ready pod -l app=nginx -n chaos-demo --timeout=60s
	@echo "✅ Demo application ready!"
	@echo "   Pods: $$(kubectl get pods -n chaos-demo --no-headers | wc -l)"

teardown: ## Clean up all lab resources
	@echo "🧹 Cleaning up Lab 01..."
	kubectl delete chaosexperiments --all -n chaos-demo --ignore-not-found=true
	kubectl delete -f setup/demo-app.yaml --ignore-not-found=true
	kubectl delete namespace chaos-demo --ignore-not-found=true
	@echo "✅ Cleanup complete!"

status: ## Show current lab status
	@echo "📊 Lab 01 Status"
	@echo "================"
	@echo ""
	@echo "Namespace:"
	@kubectl get namespace chaos-demo 2>/dev/null || echo "  ❌ Not created (run: make setup)"
	@echo ""
	@echo "Pods:"
	@kubectl get pods -n chaos-demo 2>/dev/null || echo "  No pods found"
	@echo ""
	@echo "Experiments:"
	@kubectl get chaosexperiments -n chaos-demo 2>/dev/null || echo "  No experiments running"