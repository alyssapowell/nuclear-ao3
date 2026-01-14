# Elasticsearch Security Guide

## Overview

This document explains how Elasticsearch has been secured in the Nuclear AO3 project.

## Security Measures Implemented

### 1. Docker Port Binding (docker-compose.yml)
**Changed**: Elasticsearch port binding from `9200:9200` to `127.0.0.1:9200:9200`

This ensures Elasticsearch only listens on localhost and is not accessible from external networks, even if firewall rules are misconfigured.

### 2. UFW Firewall Rules (Server-Side)
Additional layer of security blocking ports 9200 and 9300 at the firewall level.

## How Services Connect

All services connect via Docker's internal network using the hostname `elasticsearch:9200`:
- **Search Service** (port 8084) - Uses `ELASTICSEARCH_URL=http://elasticsearch:9200`
- **Tag Service** (port 8083) - Uses `ELASTICSEARCH_URL=http://elasticsearch:9200`

Docker's internal DNS resolution allows containers to communicate regardless of port binding restrictions.

## Setup Instructions

### On Your Hetzner Server

1. **SSH into your server**:
   ```bash
   ssh your-server
   ```

2. **Navigate to project directory**:
   ```bash
   cd /path/to/nuclear-ao3
   ```

3. **Run the UFW security script**:
   ```bash
   sudo bash scripts/secure-elasticsearch-ufw.sh
   ```

4. **Pull latest docker-compose changes** (if deploying via git):
   ```bash
   git pull origin main
   ```

5. **Restart Elasticsearch with new binding**:
   ```bash
   docker-compose up -d elasticsearch
   ```

6. **Verify security**:
   ```bash
   bash scripts/verify-elasticsearch-security.sh
   ```

## Verification

### Test External Access is Blocked

From your **local machine** (not the server):
```bash
curl -v http://your-server-ip:9200
```
**Expected**: Connection timeout or "Connection refused"

### Test Local Access Works

From the **server** via SSH:
```bash
curl http://localhost:9200
```
**Expected**: JSON response with Elasticsearch cluster info

### Test Application Functionality

1. Visit your application in a browser
2. Test search functionality
3. Create a new work and verify it's searchable
4. Check tag suggestions work

## Scripts Reference

### `scripts/secure-elasticsearch-ufw.sh`
Sets up UFW firewall rules to block external access to ports 9200 and 9300.

**Usage**:
```bash
sudo bash scripts/secure-elasticsearch-ufw.sh
```

**What it does**:
- Checks current UFW status
- Adds deny rules for ports 9200 and 9300
- Verifies rules were added
- Tests local Elasticsearch access
- Shows summary of changes

### `scripts/verify-elasticsearch-security.sh`
Comprehensive security verification that checks:
- UFW rules (if configured)
- Local Elasticsearch access
- Cluster health
- Docker container status
- Service logs for connection errors
- Port bindings
- Index accessibility

**Usage**:
```bash
bash scripts/verify-elasticsearch-security.sh
```

## Troubleshooting

### Elasticsearch Not Accessible After Changes

1. Check container is running:
   ```bash
   docker ps | grep elasticsearch
   ```

2. Check container logs:
   ```bash
   docker logs nuclear-ao3-elasticsearch --tail 50
   ```

3. Restart the container:
   ```bash
   docker-compose restart elasticsearch
   ```

### Search Not Working in Application

1. Check search service logs:
   ```bash
   docker logs nuclear-ao3-search-service --tail 50
   ```

2. Check tag service logs:
   ```bash
   docker logs nuclear-ao3-tags --tail 50
   ```

3. Verify services can reach Elasticsearch:
   ```bash
   docker exec nuclear-ao3-search-service curl http://elasticsearch:9200
   ```

### Need to Remove UFW Rules

```bash
sudo ufw status numbered
sudo ufw delete [RULE_NUMBER]
```

## Response to Hetzner

After implementing these security measures, you can respond to Hetzner:

> **Issue Resolved**: Elasticsearch port 9200 has been secured using two layers of defense:
>
> 1. **UFW firewall rules** blocking external access to ports 9200 and 9300
> 2. **Docker port binding** restricted to localhost only (127.0.0.1:9200:9200)
>
> The service is now only accessible locally for internal application communication. External access has been verified as blocked.

## Security Best Practices

### Current Status
- ✅ Port binding restricted to localhost
- ✅ UFW firewall rules in place
- ✅ Docker network isolation
- ⚠️ Authentication disabled (xpack.security.enabled=false)
- ⚠️ No TLS/SSL encryption

### Future Improvements (Optional)
For additional security, consider:
1. Enabling Elasticsearch xpack.security with authentication
2. Adding TLS/SSL encryption for connections
3. Implementing Elasticsearch API keys
4. Regular security audits

**Note**: For internal Docker communication with proper firewall rules, the current configuration is sufficient for most use cases.

## Files Modified

- `docker-compose.yml` (line 70) - Changed port binding to `127.0.0.1:9200:9200`
- `scripts/secure-elasticsearch-ufw.sh` - Created (UFW setup script)
- `scripts/verify-elasticsearch-security.sh` - Created (verification script)
- `ELASTICSEARCH_SECURITY.md` - Created (this file)
