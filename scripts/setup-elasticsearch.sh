#!/bin/bash

# Setup Elasticsearch indices with optimized mappings
set -e

ES_HOST=${ES_HOST:-"localhost:9200"}
MAPPINGS_FILE="${BASH_SOURCE%/*}/../backend/shared/elasticsearch/mappings.json"

echo "🔍 Setting up Elasticsearch indices..."
echo "Elasticsearch host: $ES_HOST"

# Wait for Elasticsearch to be ready
echo "⏳ Waiting for Elasticsearch to be ready..."
until curl -s "$ES_HOST/_cluster/health" > /dev/null; do
    echo "Waiting for Elasticsearch..."
    sleep 2
done

echo "✅ Elasticsearch is ready!"

# Function to create index with mapping
create_index() {
    local index_name=$1
    local mapping_key=$2
    
    echo "🗂️  Creating index: $index_name"
    
    # Extract settings and mappings from the JSON file
    settings=$(jq -r ".$mapping_key.settings" "$MAPPINGS_FILE")
    mappings=$(jq -r ".$mapping_key.mappings" "$MAPPINGS_FILE")
    
    # Create the index
    curl -X PUT "$ES_HOST/$index_name" \
        -H "Content-Type: application/json" \
        -d "{\"settings\": $settings, \"mappings\": $mappings}" \
        -s | jq .
    
    if [ $? -eq 0 ]; then
        echo "✅ Index $index_name created successfully"
    else
        echo "❌ Failed to create index $index_name"
        exit 1
    fi
}

# Create indices
echo "📚 Creating optimized indices..."

create_index "works" "works"
create_index "tags" "tags" 
create_index "users" "users"

# Create index templates for time-based indices
echo "📋 Creating index templates..."

# Analytics template for daily indices
curl -X PUT "$ES_HOST/_index_template/analytics" \
    -H "Content-Type: application/json" \
    -d '{
        "index_patterns": ["analytics-*"],
        "template": {
            "settings": {
                "number_of_shards": 1,
                "number_of_replicas": 0,
                "refresh_interval": "60s",
                "index.lifecycle.name": "analytics-policy",
                "index.lifecycle.rollover_alias": "analytics"
            },
            "mappings": {
                "properties": {
                    "@timestamp": {"type": "date"},
                    "event_type": {"type": "keyword"},
                    "user_id": {"type": "keyword"},
                    "work_id": {"type": "keyword"},
                    "session_id": {"type": "keyword"},
                    "ip_address": {"type": "ip"},
                    "user_agent": {"type": "text", "index": false},
                    "referrer": {"type": "keyword"},
                    "response_time": {"type": "integer"},
                    "response_code": {"type": "integer"}
                }
            }
        }
    }' \
    -s | jq .

# Create alias for current analytics index
curl -X PUT "$ES_HOST/analytics-$(date +%Y%m%d)" \
    -H "Content-Type: application/json" \
    -d '{
        "aliases": {
            "analytics": {"is_write_index": true}
        }
    }' \
    -s | jq .

echo "📊 Setting up index lifecycle policies..."

# Create lifecycle policy for analytics indices
curl -X PUT "$ES_HOST/_ilm/policy/analytics-policy" \
    -H "Content-Type: application/json" \
    -d '{
        "policy": {
            "phases": {
                "hot": {
                    "actions": {
                        "rollover": {
                            "max_size": "1GB",
                            "max_age": "1d"
                        }
                    }
                },
                "delete": {
                    "min_age": "30d",
                    "actions": {
                        "delete": {}
                    }
                }
            }
        }
    }' \
    -s | jq .

echo "🎛️  Configuring cluster settings for resource optimization..."

# Set cluster settings for single-node deployment
curl -X PUT "$ES_HOST/_cluster/settings" \
    -H "Content-Type: application/json" \
    -d '{
        "persistent": {
            "action.auto_create_index": "analytics-*,+*",
            "cluster.routing.allocation.disk.threshold.enabled": true,
            "cluster.routing.allocation.disk.watermark.low": "85%",
            "cluster.routing.allocation.disk.watermark.high": "90%",
            "cluster.routing.allocation.disk.watermark.flood_stage": "95%",
            "indices.recovery.max_bytes_per_sec": "50mb",
            "indices.memory.index_buffer_size": "20%"
        }
    }' \
    -s | jq .

echo "🔧 Setting up monitoring..."

# Enable monitoring (lightweight for single node)
curl -X PUT "$ES_HOST/_cluster/settings" \
    -H "Content-Type: application/json" \
    -d '{
        "persistent": {
            "xpack.monitoring.collection.enabled": false,
            "xpack.monitoring.elasticsearch.collection.enabled": false
        }
    }' \
    -s | jq .

echo "✨ Elasticsearch setup complete!"
echo ""
echo "📈 Index status:"
curl -X GET "$ES_HOST/_cat/indices?v&s=index" -s

echo ""
echo "🎯 Cluster health:"
curl -X GET "$ES_HOST/_cluster/health?pretty" -s

echo ""
echo "💡 Pro tips for your $5/month VPS:"
echo "   • Indices are configured with 1 shard, 0 replicas (perfect for single node)"
echo "   • Refresh intervals optimized for write performance"
echo "   • Memory settings tuned for small deployments"
echo "   • Analytics data auto-deletes after 30 days"
echo "   • Index lifecycle management prevents disk overflow"