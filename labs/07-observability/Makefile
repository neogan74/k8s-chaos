.PHONY: setup teardown help status metrics grafana prometheus

help: ## Display this help
	@echo "Lab 07: Observability"
	@echo "====================="
	@echo ""
	@echo "Available commands:"
	@echo "  make setup       - Deploy demo app and monitoring stack"
	@echo "  make teardown    - Clean up all resources"
	@echo "  make status      - Show current state"
	@echo "  make metrics     - Port-forward to metrics endpoint"
	@echo "  make grafana     - Port-forward to Grafana"
	@echo "  make prometheus  - Port-forward to Prometheus"
	@echo ""
	@echo "Follow README.md for full lab instructions"

setup: ## Deploy the demo application and monitoring
	@echo "Setting up Lab 07..."
	@echo ""
	@echo "Creating namespace and demo app..."
	kubectl create namespace chaos-lab --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -f setup/demo-app.yaml
	@echo ""
	@echo "Deploying Prometheus (if not exists)..."
	kubectl apply -f setup/prometheus.yaml || true
	@echo ""
	@echo "Deploying Grafana (if not exists)..."
	kubectl apply -f setup/grafana.yaml || true
	@echo ""
	@echo "Waiting for pods to be ready..."
	kubectl wait --for=condition=ready pod -l app=nginx -n chaos-lab --timeout=120s
	@echo ""
	@echo "Lab 07 environment ready!"
	@echo ""
	@echo "Access monitoring:"
	@echo "  make metrics    - Access operator metrics"
	@echo "  make grafana    - Access Grafana (admin/admin)"
	@echo "  make prometheus - Access Prometheus"

teardown: ## Clean up all lab resources
	@echo "Cleaning up Lab 07..."
	kubectl delete chaosexperiments --all -n chaos-lab --ignore-not-found=true
	kubectl delete -f setup/demo-app.yaml --ignore-not-found=true
	kubectl delete namespace chaos-lab --ignore-not-found=true
	@echo ""
	@echo "Note: Prometheus and Grafana are preserved."
	@echo "To remove monitoring: kubectl delete -f setup/prometheus.yaml setup/grafana.yaml"
	@echo ""
	@echo "Cleanup complete!"

status: ## Show current lab status
	@echo "Lab 07 Status"
	@echo "============="
	@echo ""
	@echo "Demo Pods:"
	@kubectl get pods -n chaos-lab 2>/dev/null || echo "  No pods found"
	@echo ""
	@echo "Monitoring Stack:"
	@kubectl get pods -n monitoring 2>/dev/null || echo "  Not deployed"
	@echo ""
	@echo "Experiments:"
	@kubectl get chaosexperiments -n chaos-lab 2>/dev/null || echo "  No experiments"
	@echo ""
	@echo "History Records:"
	@kubectl get chaosexperimenthistory -n k8s-chaos-system --no-headers 2>/dev/null | wc -l | xargs echo "  Total:"

metrics: ## Port-forward to operator metrics
	@echo "Starting port-forward to metrics endpoint..."
	@echo "Access at: http://localhost:8080/metrics"
	@echo "Press Ctrl+C to stop"
	kubectl port-forward -n k8s-chaos-system deployment/k8s-chaos-controller-manager 8080:8080

grafana: ## Port-forward to Grafana
	@echo "Starting port-forward to Grafana..."
	@echo "Access at: http://localhost:3000"
	@echo "Default credentials: admin / admin"
	@echo "Press Ctrl+C to stop"
	kubectl port-forward -n monitoring svc/grafana 3000:3000

prometheus: ## Port-forward to Prometheus
	@echo "Starting port-forward to Prometheus..."
	@echo "Access at: http://localhost:9090"
	@echo "Press Ctrl+C to stop"
	kubectl port-forward -n monitoring svc/prometheus 9090:9090