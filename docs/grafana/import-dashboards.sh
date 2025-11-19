#!/bin/bash
#
# Script to import K8s Chaos Grafana dashboards
# Usage: ./import-dashboards.sh [grafana-url] [api-key]
#
# Examples:
#   ./import-dashboards.sh http://localhost:3000 admin:admin
#   ./import-dashboards.sh https://grafana.example.com $GRAFANA_API_KEY
#

set -e

GRAFANA_URL="${1:-http://localhost:3000}"
GRAFANA_AUTH="${2:-admin:admin}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}K8s Chaos - Grafana Dashboard Importer${NC}"
echo "==========================================="
echo ""
echo "Grafana URL: $GRAFANA_URL"
echo "Dashboard directory: $SCRIPT_DIR"
echo ""

# Function to import a dashboard
import_dashboard() {
    local dashboard_file=$1
    local dashboard_name=$(basename "$dashboard_file" .json)

    echo -n "Importing $dashboard_name... "

    # Read dashboard JSON and wrap it in the import API format
    dashboard_json=$(cat "$dashboard_file")
    import_json=$(jq -n --argjson dashboard "$dashboard_json" '{
        dashboard: $dashboard,
        overwrite: true,
        folderId: 0
    }')

    # Import to Grafana
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -u "$GRAFANA_AUTH" \
        -d "$import_json" \
        "$GRAFANA_URL/api/dashboards/db")

    # Check if import was successful
    if echo "$response" | jq -e '.status == "success"' > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Success${NC}"
        dashboard_uid=$(echo "$response" | jq -r '.uid')
        echo "  Dashboard UID: $dashboard_uid"
        echo "  URL: $GRAFANA_URL/d/$dashboard_uid"
    else
        echo -e "${RED}✗ Failed${NC}"
        echo "  Error: $(echo "$response" | jq -r '.message // .error')"
        return 1
    fi
    echo ""
}

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo -e "${RED}Error: jq is required but not installed.${NC}"
    echo "Please install jq: https://stedolan.github.io/jq/download/"
    exit 1
fi

# Check if Grafana is accessible
echo "Checking Grafana connectivity..."
if ! curl -s -f -u "$GRAFANA_AUTH" "$GRAFANA_URL/api/health" > /dev/null; then
    echo -e "${RED}Error: Cannot connect to Grafana at $GRAFANA_URL${NC}"
    echo "Please check:"
    echo "  1. Grafana is running"
    echo "  2. URL is correct"
    echo "  3. Authentication credentials are valid"
    exit 1
fi
echo -e "${GREEN}✓ Connected to Grafana${NC}"
echo ""

# Import dashboards
echo "Importing dashboards..."
echo ""

import_dashboard "$SCRIPT_DIR/chaos-experiments-overview.json"
import_dashboard "$SCRIPT_DIR/chaos-experiments-detailed.json"
import_dashboard "$SCRIPT_DIR/chaos-safety-monitoring.json"

echo "==========================================="
echo -e "${GREEN}Dashboard import complete!${NC}"
echo ""
echo "Access your dashboards at: $GRAFANA_URL/dashboards"
echo ""
echo -e "${YELLOW}Note:${NC} Make sure Prometheus is configured as a datasource in Grafana"
echo "Default datasource URL: http://prometheus-operated.monitoring.svc:9090"
