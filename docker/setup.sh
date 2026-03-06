#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Tenant Gateway Setup ===${NC}"
echo ""

# Wait for gateway to be ready
echo "Waiting for gateway to be ready..."
until curl -sf http://localhost:8080/health > /dev/null 2>&1; do
    sleep 1
done
echo -e "${GREEN}Gateway is ready!${NC}"
echo ""

# Bootstrap admin user
echo "Bootstrapping admin user..."
BOOTSTRAP_OUTPUT=$(docker compose exec -T gateway tenant-gateway -config /etc/gateway/config.yaml -bootstrap admin 2>&1)
ADMIN_KEY=$(echo "$BOOTSTRAP_OUTPUT" | tail -1)

if [[ "$ADMIN_KEY" == *"already exists"* ]]; then
    echo -e "${YELLOW}Admin user already exists. Please use existing API key or delete the postgres volume to start fresh.${NC}"
    echo "To reset: docker compose down -v && docker compose up -d"
    exit 1
fi

echo -e "${GREEN}Admin user created!${NC}"
echo ""

# Create tenant
echo "Creating tenant 'test-tenant'..."
TENANT_RESPONSE=$(curl -sf -X POST http://localhost:8080/admin/tenants \
    -H "Authorization: Bearer $ADMIN_KEY" \
    -H "Content-Type: application/json" \
    -d '{"name": "test-tenant"}')
TENANT_ID=$(echo "$TENANT_RESPONSE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo -e "${GREEN}Tenant created: $TENANT_ID${NC}"
echo ""

# Create a regular user for Grafana
echo "Creating user 'grafana-reader'..."
USER_RESPONSE=$(curl -sf -X POST http://localhost:8080/admin/users \
    -H "Authorization: Bearer $ADMIN_KEY" \
    -H "Content-Type: application/json" \
    -d '{"username": "grafana-reader", "is_admin": false}')
USER_ID=$(echo "$USER_RESPONSE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo -e "${GREEN}User created: $USER_ID${NC}"
echo ""

# Grant read access to tenant
echo "Granting read access to test-tenant..."
curl -sf -X POST "http://localhost:8080/admin/users/$USER_ID/tenants" \
    -H "Authorization: Bearer $ADMIN_KEY" \
    -H "Content-Type: application/json" \
    -d "{\"tenant_id\": \"$TENANT_ID\", \"can_read\": true, \"can_write\": false}" > /dev/null
echo -e "${GREEN}Access granted!${NC}"
echo ""

# Create API key for Grafana user
echo "Creating API key for grafana-reader..."
KEY_RESPONSE=$(curl -sf -X POST "http://localhost:8080/admin/users/$USER_ID/api-keys" \
    -H "Authorization: Bearer $ADMIN_KEY" \
    -H "Content-Type: application/json" \
    -d '{"name": "grafana"}')
GRAFANA_KEY=$(echo "$KEY_RESPONSE" | grep -o '"key":"[^"]*"' | cut -d'"' -f4)
echo ""

echo -e "${GREEN}=== Setup Complete ===${NC}"
echo ""
echo -e "${YELLOW}Admin API Key (save this):${NC}"
echo "$ADMIN_KEY"
echo ""
echo -e "${YELLOW}Grafana API Key (for datasource):${NC}"
echo "$GRAFANA_KEY"
echo ""
echo -e "${YELLOW}To configure Grafana:${NC}"
echo "1. Open http://localhost:3000"
echo "2. Go to Connections > Data sources > Loki (via Gateway)"
echo "3. Under 'HTTP Headers', set Authorization to: Bearer $GRAFANA_KEY"
echo "4. Set X-Scope-OrgID to: test-tenant"
echo "5. Click 'Save & test'"
echo ""
echo -e "${YELLOW}To push test data:${NC}"
echo "curl -X POST http://localhost:8080/loki/api/v1/push \\"
echo "  -H 'Authorization: Bearer $ADMIN_KEY' \\"
echo "  -H 'X-Scope-OrgID: test-tenant' \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -d \"{\\\"streams\\\": [{\\\"stream\\\": {\\\"app\\\": \\\"test\\\"}, \\\"values\\\": [[\\\"\$(date +%s)000000000\\\", \\\"Hello from tenant-gateway!\\\"]]}]}\""
echo ""
