#!/bin/bash

# Nuclear AO3 Health Monitor & Auto-Recovery Script
# This script monitors all Nuclear AO3 services and can auto-restart failed components

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="/var/log/nuclear-ao3-health.log"
ALERT_EMAIL=""  # Set this if you want email alerts
SLACK_WEBHOOK=""  # Set this if you want Slack alerts
HEALTH_CHECK_INTERVAL=60  # seconds
MAX_RESTART_ATTEMPTS=3

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to log with timestamp
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') $1" | tee -a "$LOG_FILE"
}

# Function to check if a service is running and healthy
check_service() {
    local service_name="$1"
    local port="$2"
    local health_endpoint="${3:-/health}"
    
    # Check if container is running
    if ! docker ps --filter "name=$service_name" --filter "status=running" | grep -q "$service_name"; then
        echo "FAILED: Container $service_name not running"
        return 1
    fi
    
    # Check if port is responsive
    if ! nc -z localhost "$port" 2>/dev/null; then
        echo "FAILED: Port $port not responding"
        return 1
    fi
    
    # Check health endpoint if provided
    if [[ "$health_endpoint" != "none" ]]; then
        if ! curl -f -s "http://localhost:$port$health_endpoint" >/dev/null 2>&1; then
            echo "FAILED: Health endpoint $health_endpoint not responding"
            return 1
        fi
    fi
    
    echo "OK"
    return 0
}

# Function to restart a service
restart_service() {
    local container_name="$1"
    log "🔄 Restarting container: $container_name"
    
    if docker restart "$container_name" >/dev/null 2>&1; then
        log "✅ Successfully restarted $container_name"
        sleep 10  # Give service time to start
        return 0
    else
        log "❌ Failed to restart $container_name"
        return 1
    fi
}

# Function to send alert
send_alert() {
    local message="$1"
    local severity="$2"  # INFO, WARNING, CRITICAL
    
    log "🚨 ALERT [$severity]: $message"
    
    # Email alert
    if [[ -n "$ALERT_EMAIL" ]]; then
        echo "$message" | mail -s "Nuclear AO3 Alert [$severity]" "$ALERT_EMAIL" 2>/dev/null || true
    fi
    
    # Slack alert
    if [[ -n "$SLACK_WEBHOOK" ]]; then
        curl -s -X POST -H 'Content-type: application/json' \
            --data "{\"text\":\"🚨 Nuclear AO3 Alert [$severity]: $message\"}" \
            "$SLACK_WEBHOOK" >/dev/null 2>&1 || true
    fi
}

# Function to check system resources
check_system_resources() {
    local warnings=()
    
    # Check disk space
    local disk_usage=$(df / | awk 'NR==2 {print $5}' | sed 's/%//')
    if [[ "$disk_usage" -gt 90 ]]; then
        warnings+=("Disk usage critical: ${disk_usage}%")
    elif [[ "$disk_usage" -gt 80 ]]; then
        warnings+=("Disk usage high: ${disk_usage}%")
    fi
    
    # Check memory usage
    local mem_usage=$(free | awk 'NR==2{printf "%.0f", $3*100/$2}')
    if [[ "$mem_usage" -gt 90 ]]; then
        warnings+=("Memory usage critical: ${mem_usage}%")
    elif [[ "$mem_usage" -gt 80 ]]; then
        warnings+=("Memory usage high: ${mem_usage}%")
    fi
    
    # Check CPU load
    local cpu_load=$(uptime | awk -F'load average:' '{print $2}' | awk '{print $1}' | sed 's/,//')
    local cpu_cores=$(nproc)
    if (( $(echo "$cpu_load > $cpu_cores * 2" | bc -l) )); then
        warnings+=("CPU load critical: $cpu_load (cores: $cpu_cores)")
    fi
    
    # Return warnings
    for warning in "${warnings[@]}"; do
        echo "$warning"
    done
}

# Function to check Docker health
check_docker_health() {
    if ! docker info >/dev/null 2>&1; then
        echo "CRITICAL: Docker daemon not responding"
        return 1
    fi
    
    # Check Docker disk usage
    local docker_usage=$(docker system df --format "table {{.Type}}\t{{.Size}}" 2>/dev/null | grep -E "(Images|Containers)" | awk '{sum+=$2} END {print sum}' || echo "0")
    
    echo "OK"
    return 0
}

# Function to cleanup Docker resources
cleanup_docker() {
    log "🧹 Cleaning up Docker resources..."
    
    # Remove stopped containers
    docker container prune -f >/dev/null 2>&1 || true
    
    # Remove unused images (keep recent ones)
    docker image prune -f >/dev/null 2>&1 || true
    
    # Remove unused volumes (be careful!)
    # docker volume prune -f >/dev/null 2>&1 || true
    
    log "✅ Docker cleanup completed"
}

# Main health check function
perform_health_check() {
    log "🔍 Starting Nuclear AO3 health check..."
    
    local services=(
        "nuclear-ao3-staging-api-gateway:8080"
        "nuclear-ao3-staging-auth-service:8081"
        "nuclear-ao3-staging-work-service:8082"
        "nuclear-ao3-staging-tag-service:8083"
        "nuclear-ao3-staging-search-service:8084"
        "nuclear-ao3-staging-notification-service:8085"
        "nuclear-ao3-staging-export-service:8086"
        "nuclear-ao3-staging-frontend:3000:/health:none"
        "nuclear-ao3-staging-postgres:5432:none"
        "nuclear-ao3-staging-redis:6379:none"
        "nuclear-ao3-staging-elasticsearch:9200:/_cluster/health"
    )
    
    local failed_services=()
    local warnings=()
    
    # Check each service
    for service_info in "${services[@]}"; do
        IFS=':' read -r service_name port health_endpoint <<< "$service_info"
        
        printf "Checking %-35s " "$service_name..."
        result=$(check_service "$service_name" "$port" "$health_endpoint")
        
        if [[ "$result" == "OK" ]]; then
            echo -e "${GREEN}$result${NC}"
        else
            echo -e "${RED}$result${NC}"
            failed_services+=("$service_name: $result")
        fi
    done
    
    # Check system resources
    echo ""
    echo "System Resources:"
    resource_warnings=$(check_system_resources)
    if [[ -n "$resource_warnings" ]]; then
        while IFS= read -r warning; do
            echo -e "${YELLOW}⚠️  $warning${NC}"
            warnings+=("$warning")
        done <<< "$resource_warnings"
    else
        echo -e "${GREEN}✅ System resources OK${NC}"
    fi
    
    # Check Docker health
    echo ""
    printf "Checking Docker daemon... "
    docker_result=$(check_docker_health)
    if [[ "$docker_result" == "OK" ]]; then
        echo -e "${GREEN}$docker_result${NC}"
    else
        echo -e "${RED}$docker_result${NC}"
        failed_services+=("Docker: $docker_result")
    fi
    
    # Summary
    echo ""
    if [[ ${#failed_services[@]} -eq 0 ]]; then
        log "✅ All Nuclear AO3 services are healthy"
        
        # Send warnings if any
        if [[ ${#warnings[@]} -gt 0 ]]; then
            warning_msg="Nuclear AO3 has warnings: $(IFS='; '; echo "${warnings[*]}")"
            send_alert "$warning_msg" "WARNING"
        fi
    else
        log "❌ Nuclear AO3 health check failed: ${#failed_services[@]} service(s) down"
        
        # Attempt auto-recovery if enabled
        if [[ "${AUTO_RESTART:-false}" == "true" ]]; then
            for failed_service in "${failed_services[@]}"; do
                service_name=$(echo "$failed_service" | cut -d':' -f1)
                
                # Track restart attempts
                restart_count_file="/tmp/nuclear-ao3-restart-$service_name"
                restart_count=0
                if [[ -f "$restart_count_file" ]]; then
                    restart_count=$(cat "$restart_count_file")
                fi
                
                if [[ "$restart_count" -lt "$MAX_RESTART_ATTEMPTS" ]]; then
                    restart_service "$service_name"
                    echo $((restart_count + 1)) > "$restart_count_file"
                else
                    log "⚠️ Max restart attempts reached for $service_name"
                fi
            done
        fi
        
        # Send critical alert
        alert_msg="Nuclear AO3 services failed: $(IFS='; '; echo "${failed_services[*]}")"
        send_alert "$alert_msg" "CRITICAL"
        
        return 1
    fi
    
    return 0
}

# Function to show service status
show_status() {
    echo "Nuclear AO3 Service Status"
    echo "=========================="
    echo ""
    
    # Show containers
    echo "Running Containers:"
    docker ps --filter "name=nuclear-ao3" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || echo "No containers found"
    
    echo ""
    
    # Show resource usage
    echo "System Resources:"
    echo "Memory: $(free -h | awk 'NR==2{printf "%s/%s (%.0f%%)", $3,$2,$3*100/$2}')"
    echo "Disk: $(df -h / | awk 'NR==2{printf "%s/%s (%s)", $3,$2,$5}')"
    echo "Load: $(uptime | awk -F'load average:' '{print $2}' | sed 's/^ *//')"
    
    echo ""
    
    # Show recent logs
    echo "Recent Health Check Logs (last 10):"
    if [[ -f "$LOG_FILE" ]]; then
        tail -10 "$LOG_FILE"
    else
        echo "No logs found"
    fi
}

# Function to run continuous monitoring
monitor_continuous() {
    log "🚀 Starting continuous Nuclear AO3 monitoring..."
    log "Health check interval: ${HEALTH_CHECK_INTERVAL}s"
    log "Auto-restart: ${AUTO_RESTART:-false}"
    
    while true; do
        perform_health_check
        
        # Cleanup Docker resources periodically (every hour)
        if [[ $(($(date +%s) % 3600)) -lt $HEALTH_CHECK_INTERVAL ]]; then
            cleanup_docker
        fi
        
        sleep "$HEALTH_CHECK_INTERVAL"
    done
}

# Show usage
show_usage() {
    echo "Nuclear AO3 Health Monitor"
    echo ""
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  check       - Perform one-time health check"
    echo "  monitor     - Run continuous monitoring"
    echo "  status      - Show current service status"
    echo "  restart     - Restart specific service"
    echo "  cleanup     - Clean up Docker resources"
    echo ""
    echo "Options:"
    echo "  --auto-restart    - Enable automatic service restart"
    echo "  --interval N      - Set check interval in seconds (default: 60)"
    echo ""
    echo "Examples:"
    echo "  $0 check"
    echo "  $0 monitor --auto-restart --interval 30"
    echo "  $0 restart nuclear-ao3-staging-api-gateway"
    echo ""
}

# Parse command line arguments
AUTO_RESTART=false
while [[ $# -gt 0 ]]; do
    case $1 in
        check)
            COMMAND="check"
            shift
            ;;
        monitor)
            COMMAND="monitor"
            shift
            ;;
        status)
            COMMAND="status"
            shift
            ;;
        restart)
            COMMAND="restart"
            SERVICE_NAME="$2"
            shift 2
            ;;
        cleanup)
            COMMAND="cleanup"
            shift
            ;;
        --auto-restart)
            AUTO_RESTART=true
            shift
            ;;
        --interval)
            HEALTH_CHECK_INTERVAL="$2"
            shift 2
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Ensure log directory exists
sudo mkdir -p "$(dirname "$LOG_FILE")"
sudo touch "$LOG_FILE"
sudo chown "$(whoami):$(whoami)" "$LOG_FILE" 2>/dev/null || true

# Execute command
case "${COMMAND:-check}" in
    check)
        perform_health_check
        ;;
    monitor)
        monitor_continuous
        ;;
    status)
        show_status
        ;;
    restart)
        if [[ -z "$SERVICE_NAME" ]]; then
            echo "Error: Service name required for restart command"
            echo "Usage: $0 restart <service_name>"
            exit 1
        fi
        restart_service "$SERVICE_NAME"
        ;;
    cleanup)
        cleanup_docker
        ;;
    *)
        show_usage
        exit 1
        ;;
esac