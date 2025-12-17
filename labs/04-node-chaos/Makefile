.PHONY: setup teardown help status uncordon

help: ## Display this help
	@echo "Lab 04: Node Chaos"
	@echo "=================="
	@echo ""
	@echo "IMPORTANT: This lab requires a multi-node cluster!"
	@echo "  cd .. && make cluster-multi-node"
	@echo ""
	@echo "Available commands:"
	@echo "  make setup     - Deploy demo application"
	@echo "  make teardown  - Clean up all resources"
	@echo "  make status    - Show current state"
	@echo "  make uncordon  - Uncordon all lab nodes"
	@echo ""
	@echo "Follow README.md for full lab instructions"

setup: ## Deploy the demo application
	@echo "Setting up Lab 04..."
	@echo ""
	@echo "Checking cluster nodes..."
	@NODE_COUNT=$$(kubectl get nodes --no-headers | wc -l | tr -d ' '); \
	if [ "$$NODE_COUNT" -lt "2" ]; then \
		echo "WARNING: Only $$NODE_COUNT node(s) found!"; \
		echo "Node chaos requires multiple nodes."; \
		echo "Run: cd .. && make cluster-multi-node"; \
		echo ""; \
	fi
	kubectl get nodes
	@echo ""
	kubectl create namespace chaos-lab --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -f setup/demo-app.yaml
	@echo "Waiting for pods to be ready..."
	kubectl wait --for=condition=ready pod -l app=nginx -n chaos-lab --timeout=120s
	@echo ""
	@echo "Lab 04 environment ready!"
	@echo ""
	@echo "Pod distribution across nodes:"
	@kubectl get pods -n chaos-lab -o wide

teardown: ## Clean up all lab resources
	@echo "Cleaning up Lab 04..."
	kubectl delete chaosexperiments --all -n chaos-lab --ignore-not-found=true
	@echo "Uncordoning any cordoned nodes..."
	kubectl uncordon -l chaos-test=enabled 2>/dev/null || true
	kubectl delete -f setup/demo-app.yaml --ignore-not-found=true
	kubectl delete namespace chaos-lab --ignore-not-found=true
	@echo "Cleanup complete!"

status: ## Show current lab status
	@echo "Lab 04 Status"
	@echo "============="
	@echo ""
	@echo "Cluster Nodes:"
	@kubectl get nodes
	@echo ""
	@echo "Pods (note which node each is on):"
	@kubectl get pods -n chaos-lab -o wide 2>/dev/null || echo "  No pods found"
	@echo ""
	@echo "Experiments:"
	@kubectl get chaosexperiments -n chaos-lab 2>/dev/null || echo "  No experiments running"

uncordon: ## Uncordon all lab worker nodes
	@echo "Uncordoning nodes with label chaos-test=enabled..."
	kubectl uncordon -l chaos-test=enabled 2>/dev/null || echo "No nodes to uncordon"
	@echo ""
	kubectl get nodes