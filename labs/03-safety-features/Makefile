.PHONY: setup teardown help status

help: ## Display this help
	@echo "Lab 03: Safety Features"
	@echo "======================="
	@echo ""
	@echo "Available commands:"
	@echo "  make setup     - Deploy demo applications"
	@echo "  make teardown  - Clean up all resources"
	@echo "  make status    - Show current state"
	@echo ""
	@echo "Follow README.md for full lab instructions"

setup: ## Deploy the demo applications
	@echo "Setting up Lab 03..."
	@echo ""
	@echo "Creating chaos-lab namespace with 10 pods..."
	kubectl apply -f setup/chaos-lab-ns.yaml
	@echo ""
	@echo "Creating production namespace (protected)..."
	kubectl apply -f setup/production-ns.yaml
	@echo ""
	@echo "Creating critical-ns namespace (excluded)..."
	kubectl apply -f setup/critical-ns.yaml
	@echo ""
	@echo "Waiting for all pods to be ready..."
	kubectl wait --for=condition=ready pod -l app=nginx -n chaos-lab --timeout=120s
	kubectl wait --for=condition=ready pod -l app=nginx -n production --timeout=120s
	kubectl wait --for=condition=ready pod -l app=nginx -n critical-ns --timeout=120s
	@echo ""
	@echo "Lab 03 environment ready!"
	@echo ""
	@echo "Namespaces:"
	@echo "  chaos-lab:   $$(kubectl get pods -n chaos-lab --no-headers | wc -l | tr -d ' ') pods (normal)"
	@echo "  production:  $$(kubectl get pods -n production --no-headers | wc -l | tr -d ' ') pods (protected)"
	@echo "  critical-ns: $$(kubectl get pods -n critical-ns --no-headers | wc -l | tr -d ' ') pods (excluded)"

teardown: ## Clean up all lab resources
	@echo "Cleaning up Lab 03..."
	kubectl delete chaosexperiments --all -n chaos-lab --ignore-not-found=true
	kubectl delete -f setup/ --ignore-not-found=true
	kubectl delete namespace chaos-lab --ignore-not-found=true
	kubectl delete namespace production --ignore-not-found=true
	kubectl delete namespace critical-ns --ignore-not-found=true
	@echo "Cleanup complete!"

status: ## Show current lab status
	@echo "Lab 03 Status"
	@echo "============="
	@echo ""
	@echo "Namespaces and Protection Status:"
	@echo ""
	@echo "chaos-lab (normal):"
	@kubectl get pods -n chaos-lab 2>/dev/null || echo "  Not created"
	@echo ""
	@echo "production (protected - requires allowProduction: true):"
	@kubectl get pods -n production 2>/dev/null || echo "  Not created"
	@echo ""
	@echo "critical-ns (excluded - chaos blocked):"
	@kubectl get pods -n critical-ns 2>/dev/null || echo "  Not created"
	@echo ""
	@echo "Experiments:"
	@kubectl get chaosexperiments -A 2>/dev/null || echo "  No experiments running"