.PHONY: setup teardown help status run-full-scenario clean-scenarios

help: ## Display this help
	@echo "Lab 08: Advanced Scenarios"
	@echo "=========================="
	@echo ""
	@echo "Available commands:"
	@echo "  make setup            - Deploy microservices environment"
	@echo "  make teardown         - Clean up all resources"
	@echo "  make status           - Show current state"
	@echo "  make run-full-scenario - Run full resilience test"
	@echo "  make clean-scenarios  - Delete all scenario experiments"
	@echo ""
	@echo "Scenarios:"
	@echo "  scenarios/01-cascading-failure/"
	@echo "  scenarios/02-rolling-zone/"
	@echo "  scenarios/03-game-day/"
	@echo "  scenarios/04-circuit-breaker/"
	@echo "  scenarios/05-blast-radius/"
	@echo ""
	@echo "Follow README.md for full lab instructions"

setup: ## Deploy the microservices environment
	@echo "Setting up Lab 08 - Advanced Scenarios..."
	@echo ""
	kubectl create namespace chaos-lab --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -f setup/
	@echo ""
	@echo "Waiting for all pods to be ready..."
	kubectl wait --for=condition=ready pod -l app=frontend -n chaos-lab --timeout=120s
	kubectl wait --for=condition=ready pod -l app=api -n chaos-lab --timeout=120s
	kubectl wait --for=condition=ready pod -l app=database -n chaos-lab --timeout=120s
	kubectl wait --for=condition=ready pod -l app=cache -n chaos-lab --timeout=120s
	@echo ""
	@echo "Lab 08 environment ready!"
	@echo ""
	@echo "Components deployed:"
	@echo "  Frontend: $$(kubectl get pods -n chaos-lab -l app=frontend --no-headers | wc -l | tr -d ' ') pods"
	@echo "  API:      $$(kubectl get pods -n chaos-lab -l app=api --no-headers | wc -l | tr -d ' ') pods"
	@echo "  Database: $$(kubectl get pods -n chaos-lab -l app=database --no-headers | wc -l | tr -d ' ') pods"
	@echo "  Cache:    $$(kubectl get pods -n chaos-lab -l app=cache --no-headers | wc -l | tr -d ' ') pods"

teardown: ## Clean up all lab resources
	@echo "Cleaning up Lab 08..."
	kubectl delete chaosexperiments --all -n chaos-lab --ignore-not-found=true
	kubectl delete -f setup/ --ignore-not-found=true
	kubectl delete namespace chaos-lab --ignore-not-found=true
	@echo "Cleanup complete!"

status: ## Show current lab status
	@echo "Lab 08 Status"
	@echo "============="
	@echo ""
	@echo "Pods by Component:"
	@echo ""
	@echo "Frontend:"
	@kubectl get pods -n chaos-lab -l app=frontend 2>/dev/null || echo "  Not deployed"
	@echo ""
	@echo "API:"
	@kubectl get pods -n chaos-lab -l app=api 2>/dev/null || echo "  Not deployed"
	@echo ""
	@echo "Database:"
	@kubectl get pods -n chaos-lab -l app=database 2>/dev/null || echo "  Not deployed"
	@echo ""
	@echo "Cache:"
	@kubectl get pods -n chaos-lab -l app=cache 2>/dev/null || echo "  Not deployed"
	@echo ""
	@echo "Active Experiments:"
	@kubectl get chaosexperiments -n chaos-lab 2>/dev/null || echo "  No experiments"

clean-scenarios: ## Delete all scenario experiments
	@echo "Cleaning up all scenario experiments..."
	kubectl delete chaosexperiments -n chaos-lab -l scenario --ignore-not-found=true
	@echo "Done!"

run-full-scenario: ## Run full resilience scenario
	@echo "Running Full Stack Resilience Test..."
	@echo "This will take approximately 10 minutes."
	@echo ""
	@echo "Phase 1: Frontend pod failures..."
	kubectl apply -f scenarios/06-full-resilience/phase1-frontend.yaml
	sleep 30
	@echo ""
	@echo "Phase 2: API network latency..."
	kubectl apply -f scenarios/06-full-resilience/phase2-api.yaml
	sleep 30
	@echo ""
	@echo "Phase 3: Database CPU stress..."
	kubectl apply -f scenarios/06-full-resilience/phase3-database.yaml
	sleep 30
	@echo ""
	@echo "Phase 4: Cache pod failures..."
	kubectl apply -f scenarios/06-full-resilience/phase4-cache.yaml
	@echo ""
	@echo "All phases deployed. Experiments will auto-stop after their duration."
	@echo "Monitor with: kubectl get chaosexperiments -n chaos-lab -w"