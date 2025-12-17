.PHONY: help cluster-single-node cluster-multi-node cluster-delete install deploy undeploy uninstall status all

# Default target
help: ## Show this help
	@echo "K8s Chaos Engineering Labs"
	@echo "=========================="
	@echo ""
	@echo "Cluster Management:"
	@echo "  make cluster-single-node  Create single-node Kind cluster"
	@echo "  make cluster-multi-node   Create 3-node Kind cluster (for node-drain labs)"
	@echo "  make cluster-delete       Delete Kind cluster"
	@echo ""
	@echo "Operator Management:"
	@echo "  make install              Install CRDs to cluster"
	@echo "  make deploy               Deploy operator to cluster"
	@echo "  make undeploy             Remove operator from cluster"
	@echo "  make uninstall            Remove CRDs from cluster"
	@echo ""
	@echo "Quick Start:"
	@echo "  make all                  Create cluster + install + deploy"
	@echo "  make status               Show cluster and operator status"
	@echo ""
	@echo "Labs:"
	@echo "  cd 01-getting-started && make help"
	@echo "  cd 02-pod-chaos-basics && make help"
	@echo "  cd 03-safety-features && make help"
	@echo "  cd 04-node-chaos && make help"
	@echo "  cd 05-scheduling-duration && make help"
	@echo "  cd 06-retry-logic && make help"
	@echo "  cd 07-observability && make help"
	@echo "  cd 08-advanced-scenarios && make help"

# Variables
KIND_CLUSTER_NAME := k8s-chaos-lab
PROJECT_ROOT := $(shell cd .. && pwd)
IMG ?= controller:latest

# =============================================================================
# Cluster Management
# =============================================================================

cluster-single-node: ## Create single-node Kind cluster
	@echo "Creating single-node Kind cluster..."
	@kind create cluster --config infra/kind-single-node.yaml --wait 60s
	@echo ""
	@echo "Cluster created successfully!"
	@kubectl cluster-info
	@echo ""
	@echo "Next steps:"
	@echo "  make install   # Install CRDs"
	@echo "  make deploy    # Deploy operator"

cluster-multi-node: ## Create 3-node Kind cluster (required for node-drain labs)
	@echo "Creating multi-node Kind cluster (1 control-plane + 2 workers)..."
	@kind create cluster --config infra/kind-multi-node.yaml --wait 120s
	@echo ""
	@echo "Cluster created successfully!"
	@kubectl cluster-info
	@kubectl get nodes
	@echo ""
	@echo "Next steps:"
	@echo "  make install   # Install CRDs"
	@echo "  make deploy    # Deploy operator"

cluster-delete: ## Delete Kind cluster
	@echo "Deleting Kind cluster '$(KIND_CLUSTER_NAME)'..."
	@kind delete cluster --name $(KIND_CLUSTER_NAME) || true
	@echo "Cluster deleted."

# =============================================================================
# Operator Management
# =============================================================================

install: ## Install CRDs
	@echo "Installing CRDs..."
	@cd $(PROJECT_ROOT) && make install
	@echo "CRDs installed."
	@kubectl get crds | grep chaos

deploy: ## Deploy operator
	@echo "Building and loading operator image..."
	@cd $(PROJECT_ROOT) && make docker-build IMG=$(IMG)
	@kind load docker-image $(IMG) --name $(KIND_CLUSTER_NAME)
	@echo "Deploying operator..."
	@cd $(PROJECT_ROOT) && make deploy IMG=$(IMG)
	@echo ""
	@echo "Waiting for operator to be ready..."
	@kubectl wait --for=condition=available deployment/k8s-chaos-controller-manager -n k8s-chaos-system --timeout=120s || true
	@echo ""
	@kubectl get pods -n k8s-chaos-system
	@echo ""
	@echo "Operator deployed! You can now run chaos experiments."

undeploy: ## Remove operator
	@echo "Removing operator..."
	@cd $(PROJECT_ROOT) && make undeploy || true
	@echo "Operator removed."

uninstall: ## Remove CRDs
	@echo "Removing CRDs..."
	@cd $(PROJECT_ROOT) && make uninstall || true
	@echo "CRDs removed."

# =============================================================================
# Quick Start
# =============================================================================

all: cluster-single-node install deploy ## Create cluster, install CRDs, deploy operator
	@echo ""
	@echo "=========================================="
	@echo "Lab environment is ready!"
	@echo "=========================================="
	@echo ""
	@echo "Start with Lab 01:"
	@echo "  cd 01-getting-started"
	@echo "  cat README.md"
	@echo "  make setup"

all-multi: cluster-multi-node install deploy ## Create multi-node cluster, install CRDs, deploy operator
	@echo ""
	@echo "=========================================="
	@echo "Multi-node lab environment is ready!"
	@echo "=========================================="
	@echo ""
	@echo "This setup supports node-drain experiments (Lab 04)"

# =============================================================================
# Status
# =============================================================================

status: ## Show cluster and operator status
	@echo "Cluster Status"
	@echo "=============="
	@kind get clusters 2>/dev/null | grep -q $(KIND_CLUSTER_NAME) && echo "Kind cluster: Running" || echo "Kind cluster: Not found"
	@echo ""
	@echo "Nodes:"
	@kubectl get nodes 2>/dev/null || echo "  Cannot connect to cluster"
	@echo ""
	@echo "Operator Status"
	@echo "==============="
	@kubectl get pods -n k8s-chaos-system 2>/dev/null || echo "  Operator not deployed"
	@echo ""
	@echo "CRDs:"
	@kubectl get crds 2>/dev/null | grep chaos || echo "  No chaos CRDs installed"
	@echo ""
	@echo "Experiments:"
	@kubectl get chaosexperiments -A 2>/dev/null || echo "  No experiments found"

# =============================================================================
# Cleanup
# =============================================================================

clean: cluster-delete ## Full cleanup - delete cluster
	@echo "Full cleanup complete."

reset: undeploy uninstall ## Reset operator (keep cluster)
	@echo "Operator reset complete. Run 'make install deploy' to reinstall."