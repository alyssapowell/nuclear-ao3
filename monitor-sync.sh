#!/bin/bash

# Simple sync monitoring script
echo "🔍 Monitoring Elasticsearch sync..."
echo "Initial count: $(curl -s http://localhost:9200/works/_count | jq '.count')"

echo ""
echo "Now create a work in the frontend at http://localhost:3001/works/new"
echo "Press Enter after creating the work to check sync..."
read -r

echo ""
echo "Final count: $(curl -s http://localhost:9200/works/_count | jq '.count')"

echo ""
echo "Recent indexing activity:"
docker-compose logs work-service --tail=20 | grep -E "(indexing|DEBUG.*index|Successfully indexed)"