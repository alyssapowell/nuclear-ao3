#!/bin/bash

# Elasticsearch Security - UFW Configuration Script
# This script blocks external access to Elasticsearch ports 9200 and 9300
# while maintaining internal Docker network communication

set -e

echo "=========================================="
echo "Elasticsearch Security - UFW Setup"
echo "=========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Error: This script must be run as root or with sudo${NC}"
    echo "Usage: sudo bash scripts/secure-elasticsearch-ufw.sh"
    exit 1
fi

# Step 1: Check current UFW status
echo -e "${YELLOW}Step 1: Checking current UFW status...${NC}"
ufw status verbose
echo ""

# Step 2: Block Elasticsearch ports
echo -e "${YELLOW}Step 2: Blocking external access to Elasticsearch ports...${NC}"
echo "Adding rule: deny 9200/tcp (Elasticsearch HTTP)"
ufw deny 9200/tcp

echo "Adding rule: deny 9300/tcp (Elasticsearch Transport)"
ufw deny 9300/tcp
echo ""

# Step 3: Verify rules were added
echo -e "${YELLOW}Step 3: Verifying rules were added...${NC}"
ufw status numbered
echo ""

# Step 4: Test local access
echo -e "${YELLOW}Step 4: Testing local Elasticsearch access...${NC}"
if curl -s -o /dev/null -w "%{http_code}" http://localhost:9200 | grep -q "200"; then
    echo -e "${GREEN}✓ Success: Elasticsearch is accessible from localhost${NC}"
else
    echo -e "${RED}✗ Warning: Elasticsearch is not responding on localhost${NC}"
    echo "  Check if Elasticsearch container is running:"
    echo "  docker ps | grep elasticsearch"
fi
echo ""

# Step 5: Check Docker services
echo -e "${YELLOW}Step 5: Checking Elasticsearch container status...${NC}"
if docker ps | grep -q "nuclear-ao3-elasticsearch"; then
    echo -e "${GREEN}✓ Elasticsearch container is running${NC}"
    docker ps | grep elasticsearch
else
    echo -e "${YELLOW}⚠ Warning: Elasticsearch container not found${NC}"
    echo "  Start it with: docker-compose up -d elasticsearch"
fi
echo ""

# Summary
echo "=========================================="
echo -e "${GREEN}UFW Configuration Complete!${NC}"
echo "=========================================="
echo ""
echo "Elasticsearch ports are now blocked from external access."
echo "Services within Docker can still access via: http://elasticsearch:9200"
echo ""
echo "Next steps:"
echo "1. Test external access (should fail):"
echo "   curl http://YOUR_SERVER_IP:9200"
echo ""
echo "2. Test internal access (should work):"
echo "   curl http://localhost:9200"
echo ""
echo "3. Verify application search functionality still works"
echo ""
echo "To remove these rules if needed:"
echo "   sudo ufw status numbered"
echo "   sudo ufw delete [RULE_NUMBER]"
echo ""
