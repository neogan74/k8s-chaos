.PHONY: setup teardown help status

help: ## Display this help
	@echo "Lab 05: Scheduling & Duration"
	@echo "============================="
	@echo ""
	@echo "Available commands:"
	@echo "  make setup     - Deploy demo application"
	@echo "  make teardown  - Clean up all resources"
	@echo "  make status    - Show current state and schedules"
	@echo ""
	@echo "Follow README.md for full lab instructions"

setup: ## Deploy the demo application
	@echo "Setting up Lab 05..."
	kubectl create namespace chaos-lab --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -f setup/demo-app.yaml
	@echo "Waiting for pods to be ready..."
	kubectl wait --for=condition=ready pod -l app=nginx -n chaos-lab --timeout=120s
	@echo ""
	@echo "Lab 05 environment ready!"
	@echo "Current time: $$(date)"
	@echo "Pods: $$(kubectl get pods -n chaos-lab --no-headers | wc -l | tr -d ' ')"

teardown: ## Clean up all lab resources
	@echo "Cleaning up Lab 05..."
	kubectl delete chaosexperiments --all -n chaos-lab --ignore-not-found=true
	kubectl delete -f setup/demo-app.yaml --ignore-not-found=true
	kubectl delete namespace chaos-lab --ignore-not-found=true
	@echo "Cleanup complete!"

status: ## Show current lab status
	@echo "Lab 05 Status"
	@echo "============="
	@echo ""
	@echo "Current time: $$(date)"
	@echo ""
	@echo "Pods:"
	@kubectl get pods -n chaos-lab 2>/dev/null || echo "  No pods found"
	@echo ""
	@echo "Scheduled Experiments:"
	@kubectl get chaosexperiments -n chaos-lab -o custom-columns=NAME:.metadata.name,SCHEDULE:.spec.schedule,DURATION:.spec.experimentDuration,NEXT:.status.nextScheduledTime,LAST:.status.lastScheduledTime 2>/dev/null || echo "  No experiments"