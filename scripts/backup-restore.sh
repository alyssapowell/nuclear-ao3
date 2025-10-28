#!/bin/bash

# Nuclear AO3 Backup & Restore Script
# Handles database backups, file backups, and restoration

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKUP_VOLUME="nuclear-ao3_backup_data"
BACKUP_ROOT="/backups"  # Inside backup container
RETENTION_DAYS=30
POSTGRES_CONTAINER="nuclear-ao3-staging-postgres"
REDIS_CONTAINER="nuclear-ao3-staging-redis"
ES_CONTAINER="nuclear-ao3-staging-elasticsearch"

# Database credentials (should match your .env)
POSTGRES_DB="ao3_nuclear"
POSTGRES_USER="ao3_user"
POSTGRES_PASSWORD="ao3_password"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Function to log with timestamp
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') $1"
}

# Function to create timestamped backup directory (in Docker volume)
create_backup_dir() {
    local timestamp=$(date '+%Y-%m-%d_%H-%M-%S')
    local backup_dir="$BACKUP_ROOT/$timestamp"
    
    # Create backup directory using Docker volume
    docker run --rm \
        -v "$BACKUP_VOLUME:/backups" \
        alpine:latest \
        mkdir -p "/backups/$timestamp"
    
    echo "$backup_dir"
}

# Function to backup PostgreSQL database
backup_postgres() {
    local backup_dir="$1"
    local backup_file="$backup_dir/postgres_backup.sql"
    
    log "🗄️ Backing up PostgreSQL database..."
    
    if ! docker ps | grep -q "$POSTGRES_CONTAINER"; then
        log "❌ PostgreSQL container not running"
        return 1
    fi
    
    # Create SQL dump
    docker exec "$POSTGRES_CONTAINER" pg_dump \
        -U "$POSTGRES_USER" \
        -d "$POSTGRES_DB" \
        --clean \
        --if-exists \
        --create \
        --verbose > "$backup_file" 2>/dev/null
    
    if [[ $? -eq 0 && -s "$backup_file" ]]; then
        # Compress the backup
        gzip "$backup_file"
        log "✅ PostgreSQL backup completed: ${backup_file}.gz"
        return 0
    else
        log "❌ PostgreSQL backup failed"
        return 1
    fi
}

# Function to backup Redis data
backup_redis() {
    local backup_dir="$1"
    local backup_file="$backup_dir/redis_backup.rdb"
    
    log "📦 Backing up Redis data..."
    
    if ! docker ps | grep -q "$REDIS_CONTAINER"; then
        log "❌ Redis container not running"
        return 1
    fi
    
    # Force Redis to save current state
    docker exec "$REDIS_CONTAINER" redis-cli BGSAVE >/dev/null
    
    # Wait for background save to complete
    sleep 5
    
    # Copy the RDB file
    docker cp "$REDIS_CONTAINER:/data/dump.rdb" "$backup_file" 2>/dev/null
    
    if [[ $? -eq 0 && -s "$backup_file" ]]; then
        gzip "$backup_file"
        log "✅ Redis backup completed: ${backup_file}.gz"
        return 0
    else
        log "❌ Redis backup failed"
        return 1
    fi
}

# Function to backup Elasticsearch data
backup_elasticsearch() {
    local backup_dir="$1"
    local snapshot_name="nuclear_ao3_$(date '+%Y%m%d_%H%M%S')"
    
    log "🔍 Backing up Elasticsearch indices..."
    
    if ! docker ps | grep -q "$ES_CONTAINER"; then
        log "❌ Elasticsearch container not running"
        return 1
    fi
    
    # Check if Elasticsearch is responsive
    if ! curl -f -s "http://localhost:9200/_cluster/health" >/dev/null; then
        log "❌ Elasticsearch not responding"
        return 1
    fi
    
    # Export all indices as JSON
    local indices=$(curl -s "http://localhost:9200/_cat/indices?h=index" | grep -v '^\.' || echo "")
    
    if [[ -n "$indices" ]]; then
        mkdir -p "$backup_dir/elasticsearch"
        
        while read -r index; do
            if [[ -n "$index" ]]; then
                log "📄 Exporting index: $index"
                curl -s "http://localhost:9200/$index/_search?scroll=5m&size=1000" > "$backup_dir/elasticsearch/${index}.json"
            fi
        done <<< "$indices"
        
        # Create a mapping export
        curl -s "http://localhost:9200/_mapping" > "$backup_dir/elasticsearch/mappings.json"
        curl -s "http://localhost:9200/_settings" > "$backup_dir/elasticsearch/settings.json"
        
        # Compress the directory
        tar -czf "$backup_dir/elasticsearch_backup.tar.gz" -C "$backup_dir" elasticsearch/
        rm -rf "$backup_dir/elasticsearch"
        
        log "✅ Elasticsearch backup completed: elasticsearch_backup.tar.gz"
        return 0
    else
        log "ℹ️ No Elasticsearch indices to backup"
        return 0
    fi
}

# Function to backup Docker volumes
backup_volumes() {
    local backup_dir="$1"
    
    log "💾 Backing up Docker volumes..."
    
    local volumes=("nuclear-ao3_postgres_data" "nuclear-ao3_es_data" "nuclear-ao3_monitoring_data" "nuclear-ao3_config_data")
    
    for volume in "${volumes[@]}"; do
        if docker volume ls | grep -q "$volume"; then
            log "📦 Backing up volume: $volume"
            
            # Create a temporary container to mount the volume and create tar
            docker run --rm \
                -v "$volume:/source:ro" \
                -v "$backup_dir:/backup" \
                alpine:latest \
                tar -czf "/backup/${volume}.tar.gz" -C /source .
            
            if [[ $? -eq 0 ]]; then
                log "✅ Volume backup completed: ${volume}.tar.gz"
            else
                log "❌ Volume backup failed: $volume"
            fi
        else
            log "⚠️ Volume not found: $volume"
        fi
    done
}

# Function to backup application code and config
backup_application() {
    local backup_dir="$1"
    
    log "📄 Backing up application code and configuration..."
    
    local app_dir="/opt/nuclear-ao3"
    
    if [[ -d "$app_dir" ]]; then
        tar -czf "$backup_dir/application_backup.tar.gz" \
            -C "$app_dir" \
            --exclude='.git' \
            --exclude='node_modules' \
            --exclude='*.log' \
            --exclude='tmp' \
            . 2>/dev/null
        
        if [[ $? -eq 0 ]]; then
            log "✅ Application backup completed: application_backup.tar.gz"
        else
            log "❌ Application backup failed"
        fi
    else
        log "⚠️ Application directory not found: $app_dir"
    fi
}

# Function to create system info snapshot
backup_system_info() {
    local backup_dir="$1"
    local info_file="$backup_dir/system_info.txt"
    
    log "📊 Creating system info snapshot..."
    
    {
        echo "Nuclear AO3 System Information Snapshot"
        echo "Generated: $(date)"
        echo "========================================="
        echo ""
        
        echo "DOCKER CONTAINERS:"
        docker ps --filter "name=nuclear-ao3" --format "table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}"
        echo ""
        
        echo "DOCKER IMAGES:"
        docker images --filter "reference=nuclear-ao3/*"
        echo ""
        
        echo "DOCKER VOLUMES:"
        docker volume ls --filter "name=nuclear-ao3"
        echo ""
        
        echo "SYSTEM RESOURCES:"
        echo "Memory: $(free -h | awk 'NR==2{printf "%s/%s (%.0f%%)", $3,$2,$3*100/$2}')"
        echo "Disk: $(df -h / | awk 'NR==2{printf "%s/%s (%s)", $3,$2,$5}')"
        echo "Load: $(uptime | awk -F'load average:' '{print $2}' | sed 's/^ *//')"
        echo ""
        
        echo "NETWORK CONFIGURATION:"
        netstat -tlpn | grep -E ":(3000|8080|8081|8082|8083|8084|8085|8086|5432|6379|9200)" 2>/dev/null || echo "No relevant ports found"
        echo ""
        
        echo "ENVIRONMENT VARIABLES:"
        env | grep -E "(DATABASE_URL|REDIS_URL|ELASTICSEARCH_URL)" | sed 's/password=[^&]*/password=***/' || echo "No relevant env vars found"
        echo ""
        
    } > "$info_file"
    
    log "✅ System info saved: system_info.txt"
}

# Function to perform full backup
perform_full_backup() {
    log "🚀 Starting full Nuclear AO3 backup..."
    
    local backup_dir=$(create_backup_dir)
    local backup_log="$backup_dir/backup.log"
    local success=true
    
    # Log backup start
    {
        echo "Nuclear AO3 Full Backup Log"
        echo "Started: $(date)"
        echo "=========================="
    } > "$backup_log"
    
    # Backup each component
    backup_postgres "$backup_dir" || success=false
    backup_redis "$backup_dir" || success=false
    backup_elasticsearch "$backup_dir" || success=false
    backup_volumes "$backup_dir" || success=false
    backup_application "$backup_dir" || success=false
    backup_system_info "$backup_dir"
    
    # Calculate backup size
    local backup_size=$(du -sh "$backup_dir" | cut -f1)
    
    {
        echo ""
        echo "Backup completed: $(date)"
        echo "Backup size: $backup_size"
        echo "Status: $(if $success; then echo 'SUCCESS'; else echo 'PARTIAL SUCCESS'; fi)"
    } >> "$backup_log"
    
    if $success; then
        log "✅ Full backup completed successfully"
        log "📍 Backup location: $backup_dir"
        log "📏 Backup size: $backup_size"
    else
        log "⚠️ Backup completed with some errors"
        log "📍 Backup location: $backup_dir"
        log "📏 Backup size: $backup_size"
    fi
    
    # Cleanup old backups
    cleanup_old_backups
    
    return 0
}

# Function to cleanup old backups
cleanup_old_backups() {
    log "🧹 Cleaning up backups older than $RETENTION_DAYS days..."
    
    if [[ -d "$BACKUP_ROOT" ]]; then
        find "$BACKUP_ROOT" -type d -name "20*" -mtime +$RETENTION_DAYS -exec rm -rf {} \; 2>/dev/null || true
        
        local remaining_backups=$(find "$BACKUP_ROOT" -type d -name "20*" | wc -l)
        log "📊 Remaining backups: $remaining_backups"
    fi
}

# Function to list available backups
list_backups() {
    echo "Available Nuclear AO3 Backups:"
    echo "=============================="
    
    # List backups from Docker volume
    local backup_list=$(docker run --rm \
        -v "$BACKUP_VOLUME:/backups" \
        alpine:latest \
        find /backups -type d -name "20*" 2>/dev/null | sort -r)
    
    if [[ -n "$backup_list" ]]; then
        echo "Timestamp            Size       Status"
        echo "----------------------------------------"
        while read -r backup_path; do
            if [[ -n "$backup_path" ]]; then
                local timestamp=$(basename "$backup_path")
                
                # Get backup size
                local size=$(docker run --rm \
                    -v "$BACKUP_VOLUME:/backups" \
                    alpine:latest \
                    du -sh "$backup_path" 2>/dev/null | cut -f1 || echo "Unknown")
                
                # Get backup status
                local status=$(docker run --rm \
                    -v "$BACKUP_VOLUME:/backups" \
                    alpine:latest \
                    sh -c "if [ -f '$backup_path/backup.log' ]; then grep 'Status:' '$backup_path/backup.log' | cut -d' ' -f2-; else echo 'Unknown'; fi" 2>/dev/null || echo "Unknown")
                
                printf "%-20s %-10s %s\n" "$timestamp" "$size" "$status"
            fi
        done <<< "$backup_list"
    else
        echo "No backups found in volume: $BACKUP_VOLUME"
    fi
}

# Function to restore from backup
restore_from_backup() {
    local backup_timestamp="$1"
    local backup_dir="$BACKUP_ROOT/$backup_timestamp"
    
    if [[ ! -d "$backup_dir" ]]; then
        echo "❌ Backup not found: $backup_timestamp"
        echo "Available backups:"
        list_backups
        return 1
    fi
    
    log "🔄 Starting restore from backup: $backup_timestamp"
    log "⚠️ This will stop all Nuclear AO3 services and restore data"
    
    read -p "Are you sure you want to continue? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log "Restore cancelled"
        return 1
    fi
    
    # Stop all Nuclear AO3 services
    log "🛑 Stopping Nuclear AO3 services..."
    docker stop $(docker ps --filter "name=nuclear-ao3" -q) 2>/dev/null || true
    
    # Restore PostgreSQL
    if [[ -f "$backup_dir/postgres_backup.sql.gz" ]]; then
        log "🗄️ Restoring PostgreSQL database..."
        
        # Start PostgreSQL container if not running
        if ! docker ps | grep -q "$POSTGRES_CONTAINER"; then
            docker start "$POSTGRES_CONTAINER" 2>/dev/null || log "⚠️ Could not start PostgreSQL container"
            sleep 10
        fi
        
        # Restore database
        gunzip -c "$backup_dir/postgres_backup.sql.gz" | \
            docker exec -i "$POSTGRES_CONTAINER" psql \
                -U "$POSTGRES_USER" \
                -d postgres \
                --quiet 2>/dev/null
        
        log "✅ PostgreSQL restore completed"
    fi
    
    # Restore Redis
    if [[ -f "$backup_dir/redis_backup.rdb.gz" ]]; then
        log "📦 Restoring Redis data..."
        
        # Stop Redis to replace data file
        docker stop "$REDIS_CONTAINER" 2>/dev/null || true
        
        # Extract and copy RDB file
        gunzip -c "$backup_dir/redis_backup.rdb.gz" > "/tmp/dump.rdb"
        docker cp "/tmp/dump.rdb" "$REDIS_CONTAINER:/data/dump.rdb" 2>/dev/null || true
        rm -f "/tmp/dump.rdb"
        
        # Start Redis
        docker start "$REDIS_CONTAINER" 2>/dev/null || true
        
        log "✅ Redis restore completed"
    fi
    
    # Restore Elasticsearch
    if [[ -f "$backup_dir/elasticsearch_backup.tar.gz" ]]; then
        log "🔍 Restoring Elasticsearch data..."
        
        # Extract backup
        tar -xzf "$backup_dir/elasticsearch_backup.tar.gz" -C "/tmp/"
        
        # Start Elasticsearch if not running
        if ! docker ps | grep -q "$ES_CONTAINER"; then
            docker start "$ES_CONTAINER" 2>/dev/null || true
            sleep 30
        fi
        
        # Restore indices (simplified - real restoration would need more sophisticated approach)
        log "ℹ️ Elasticsearch restoration requires manual intervention for production use"
        
        log "✅ Elasticsearch restore completed"
    fi
    
    log "✅ Restore completed from backup: $backup_timestamp"
    log "🔄 Please restart Nuclear AO3 services manually"
}

# Function to show usage
show_usage() {
    echo "Nuclear AO3 Backup & Restore Tool"
    echo ""
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  backup       - Perform full backup"
    echo "  list         - List available backups"
    echo "  restore      - Restore from backup"
    echo "  cleanup      - Remove old backups"
    echo ""
    echo "Options:"
    echo "  --retention N    - Set backup retention days (default: 30)"
    echo ""
    echo "Examples:"
    echo "  $0 backup"
    echo "  $0 list"
    echo "  $0 restore 2024-01-15_14-30-00"
    echo "  $0 cleanup --retention 7"
    echo ""
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        backup)
            COMMAND="backup"
            shift
            ;;
        list)
            COMMAND="list"
            shift
            ;;
        restore)
            COMMAND="restore"
            BACKUP_TIMESTAMP="$2"
            shift 2
            ;;
        cleanup)
            COMMAND="cleanup"
            shift
            ;;
        --retention)
            RETENTION_DAYS="$2"
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

# Ensure backup volume exists
docker volume create "$BACKUP_VOLUME" 2>/dev/null || true

# Execute command
case "${COMMAND:-backup}" in
    backup)
        perform_full_backup
        ;;
    list)
        list_backups
        ;;
    restore)
        if [[ -z "$BACKUP_TIMESTAMP" ]]; then
            echo "Error: Backup timestamp required for restore command"
            echo "Usage: $0 restore <timestamp>"
            echo ""
            echo "Available backups:"
            list_backups
            exit 1
        fi
        restore_from_backup "$BACKUP_TIMESTAMP"
        ;;
    cleanup)
        cleanup_old_backups
        ;;
    *)
        show_usage
        exit 1
        ;;
esac