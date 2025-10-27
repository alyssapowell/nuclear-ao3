#!/bin/bash

# Docker Health Monitor Script
# Monitors service health and auto-restarts unhealthy containers

LOG_FILE="/tmp/docker-health-monitor.log"
NOTIFY_WEBHOOK=""  # Optional: Add Slack/Discord webhook for notifications

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

check_and_restart() {
    local service_name="$1"
    local container_name="$2"
    
    # Get container health status
    health_status=$(docker inspect --format='{{.State.Health.Status}}' "$container_name" 2>/dev/null)
    
    if [ "$health_status" = "unhealthy" ]; then
        log "🚨 UNHEALTHY: $service_name ($container_name) - Restarting..."
        
        # Get container logs before restart
        docker logs "$container_name" --tail 20 >> "$LOG_FILE" 2>&1
        
        # Restart the container
        docker restart "$container_name"
        
        if [ $? -eq 0 ]; then
            log "✅ RESTARTED: $service_name successfully restarted"
            
            # Wait and check if restart was successful
            sleep 30
            new_status=$(docker inspect --format='{{.State.Health.Status}}' "$container_name" 2>/dev/null)
            log "📊 POST-RESTART STATUS: $service_name is now $new_status"
        else
            log "❌ RESTART FAILED: Could not restart $service_name"
        fi
        
    elif [ "$health_status" = "healthy" ]; then
        log "✅ HEALTHY: $service_name ($container_name)"
    elif [ "$health_status" = "starting" ]; then
        log "🔄 STARTING: $service_name ($container_name)"
    elif [ -z "$health_status" ]; then
        log "⚠️  NO HEALTH CHECK: $service_name ($container_name) - no health check configured"
    else
        log "❓ UNKNOWN STATUS: $service_name ($container_name) - status: $health_status"
    fi
}

log "🚀 Starting Docker Health Monitor..."

# Main monitoring loop
while true; do
    log "🔍 Checking service health..."
    
    # Check critical services
    check_and_restart "API Gateway" "ao3_api_gateway"
    check_and_restart "Work Service" "ao3_work_service"
    check_and_restart "Auth Service" "ao3_auth_service"
    check_and_restart "Tag Service" "ao3_tag_service"
    check_and_restart "Search Service" "ao3_search_service"
    check_and_restart "Notification Service" "ao3_notification_service"
    check_and_restart "PostgreSQL" "ao3_postgres"
    check_and_restart "Redis" "ao3_redis"
    check_and_restart "Elasticsearch" "ao3_elasticsearch"
    
    log "📊 Health check cycle completed"
    echo "---" >> "$LOG_FILE"
    
    # Wait 60 seconds before next check
    sleep 60
done