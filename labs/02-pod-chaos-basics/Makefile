.PHONY: setup teardown help status

help: ## Display this help
	@echo "Lab 02: Pod Chaos Basics"
	@echo "========================"
	@echo ""
	@echo "Available commands:"
	@echo "  make setup     - Deploy demo application"
	@echo "  make teardown  - Clean up all resources"
	@echo "  make status    - Show current state"
	@echo ""
	@echo "Follow README.md for full lab instructions"

setup: ## Deploy the demo application
	@echo "Setting up Lab 02..."
	kubectl create namespace chaos-lab --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -f setup/demo-app.yaml
	@echo "Waiting for pods to be ready..."
	kubectl wait --for=condition=ready pod -l app=nginx -n chaos-lab --timeout=120s
	@echo ""
	@echo "Lab 02 environment ready!"
	@echo "Pods running: $$(kubectl get pods -n chaos-lab --no-headers | wc -l | tr -d ' ')"

teardown: ## Clean up all lab resources
	@echo "Cleaning up Lab 02..."
	kubectl delete chaosexperiments --all -n chaos-lab --ignore-not-found=true
	kubectl delete -f setup/demo-app.yaml --ignore-not-found=true
	kubectl delete namespace chaos-lab --ignore-not-found=true
	@echo "Cleanup complete!"

status: ## Show current lab status
	@echo "Lab 02 Status"
	@echo "============="
	@echo ""
	@echo "Namespace:"
	@kubectl get namespace chaos-lab 2>/dev/null || echo "  Not created (run: make setup)"
	@echo ""
	@echo "Pods:"
	@kubectl get pods -n chaos-lab -o wide 2>/dev/null || echo "  No pods found"
	@echo ""
	@echo "Experiments:"
	@kubectl get chaosexperiments -n chaos-lab 2>/dev/null || echo "  No experiments running"