# Nuclear AO3 Deployment Guide 🚀

Complete step-by-step guide for deploying Nuclear AO3 to Hetzner Cloud with zero-downtime blue-green deployment.

## 📋 Prerequisites

### Required Information
- **Hetzner Cloud Account** with API access
- **Domain name** (e.g., `your-domain.com`)
- **GitHub repository** with proper access
- **Email for SSL certificates** (Let's Encrypt)

### Server Requirements
- **Recommended**: Hetzner CPX31 (4 vCPU, 8GB RAM, 160GB SSD)
- **Minimum**: CPX21 (3 vCPU, 4GB RAM, 80GB SSD) - for testing only
- **Operating System**: Ubuntu 22.04 LTS

## 🔧 Phase 1: Server Setup

### 1. Create Hetzner Server

```bash
# Using Hetzner CLI (optional)
hcloud server create \
  --name nuclear-ao3-prod \
  --type cpx31 \
  --image ubuntu-22.04 \
  --ssh-key your-ssh-key \
  --location nbg1

# Or use the Hetzner Console: https://console.hetzner.cloud
```

### 2. Configure DNS
Point your domain to the server's IP address:
```
A    your-domain.com    → YOUR.SERVER.IP.ADDRESS
A    staging.your-domain.com → YOUR.SERVER.IP.ADDRESS (optional)
```

### 3. Initial Server Setup

```bash
# SSH into your server
ssh root@YOUR-SERVER-IP

# Download and run the setup script
wget https://raw.githubusercontent.com/YOUR-ORG/nuclear-ao3/main/scripts/setup-server.sh
chmod +x setup-server.sh
sudo ./setup-server.sh
```

**The setup script will:**
- Install Docker and Docker Compose
- Install Caddy web server
- Create system users and directories
- Configure firewall rules
- Set up log rotation
- Install monitoring tools

## 🔑 Phase 2: GitHub Actions Setup

The `ops-*` workflows (Actions tab) run diagnose, backup, upgrade, restore and
logout on the server over SSH. The server only accepts a fixed list of
operations for that key. Follow `docs/OPS_PIPELINES.md` once to set up the
`ops` user, the SSH key, the four secrets (`OPS_SSH_KEY`, `OPS_HOST`,
`OPS_USER`, `OPS_KNOWN_HOSTS`) and the two environments.

## 🌊 Phase 3: First Deployment

### 1. Manual Server Preparation

```bash
# SSH into your server
ssh root@YOUR-SERVER-IP

# Create deployment directory and clone repository
sudo mkdir -p /opt/nuclear-ao3
sudo chown deploy:deploy /opt/nuclear-ao3
sudo -u deploy git clone https://github.com/YOUR-ORG/nuclear-ao3.git /opt/nuclear-ao3
cd /opt/nuclear-ao3

# Set up environment file
sudo -u deploy cp .env.example .env
sudo -u deploy nano .env  # Edit configuration
```

**Key environment variables to set:**
```bash
ENVIRONMENT=production
DOMAIN=your-domain.com
POSTGRES_PASSWORD=your-secure-password
```

### 2. Initial Database Setup

```bash
# Start infrastructure services
cd /opt/nuclear-ao3
docker-compose up -d postgres

# Wait for services to be ready
sleep 30

# Database migrations run automatically when the app starts.
# To run them by hand, or to check what's pending:
docker-compose run --rm app nuclear migrate
docker-compose run --rm app nuclear migrate status
```

### 3. Trigger GitHub Actions Deployment

```bash
# Moving the server to a new version: GitHub -> Actions -> ops-upgrade ->
# Run workflow, and type the tag or branch. One-time setup: docs/OPS_PIPELINES.md.
# Or on the server:
make ops-upgrade REF=v1.0.0
```

## 🔄 Phase 4: Ongoing Operations

### Health Monitoring

```bash
# Check everything (containers, health, migrations, jobs, disk, backup, key)
make ops-diagnose
# or run the "ops-diagnose" workflow in the Actions tab. It also runs daily.
```

### Backup Management

```bash
# Back up the database (also runs daily from the "ops-backup" workflow)
make ops-backup

# List available backups
ls -l backups/

# Replace the database with a backup (asks you to confirm)
make ops-restore FILE=nuclear-20260115-143000.sql.gz
```

### Log Management

```bash
# View application logs
docker logs nuclear-ao3-staging-api-gateway --tail 100 -f

# Check disk usage
df -h
docker system df
```

## 🔧 Phase 5: Configuration Management

### Caddy Configuration

Edit `/etc/caddy/Caddyfile` for custom routing:

```caddyfile
# Production
your-domain.com {
    reverse_proxy localhost:8080
    
    # Security headers
    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
        X-Content-Type-Options "nosniff"
        X-Frame-Options "DENY"
        X-XSS-Protection "1; mode=block"
    }
    
    # Rate limiting
    rate_limit {
        zone dynamic {
            key    {remote_host}
            events 100
            window 1m
        }
    }
}

# Staging (optional)
staging.your-domain.com {
    reverse_proxy localhost:8080
}
```

### Database Configuration

For production tuning, edit PostgreSQL settings:

```bash
# Connect to database container
docker exec -it nuclear-ao3-postgres bash

# Edit postgresql.conf
vi /var/lib/postgresql/data/postgresql.conf

# Key settings for 8GB RAM server:
shared_buffers = 2GB
effective_cache_size = 6GB
maintenance_work_mem = 512MB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
```

## 📊 Phase 6: Monitoring & Alerts

### System Monitoring

```bash
# Install and configure monitoring (optional)
# Prometheus + Grafana setup
docker run -d \
  --name nuclear-ao3-grafana \
  -p 3001:3000 \
  -v grafana-storage:/var/lib/grafana \
  grafana/grafana:latest

# Access Grafana at: http://your-server:3001
# Default login: admin/admin
```

### Alert Configuration

The `ops-diagnose` workflow runs every day. When it finds a problem the run
fails, and GitHub emails you about it. See docs/OPS_PIPELINES.md.

## 🚀 Deployment Workflows

### Staging Deployment (Automatic)
- **Trigger**: Push to `main` branch
- **Environment**: `staging.your-domain.com`
- **Strategy**: Blue-green deployment
- **Downtime**: Zero

### Production Deployment (Manual)
- **Trigger**: Manual workflow dispatch
- **Environment**: `your-domain.com`
- **Strategy**: Blue-green deployment
- **Downtime**: Zero
- **Rollback**: Automatic on failure

### Deployment Process
1. **Test Phase**: Run full test suite
2. **Build Phase**: Build all 6 microservices + frontend
3. **Deploy Phase**: Blue-green container swap
4. **Health Check**: Validate all services
5. **Traffic Switch**: Update load balancer
6. **Cleanup**: Remove old containers

## ⚡ Quick Commands Reference

### Development
```bash
# Local development
make dev-start        # Start all services locally
make dev-test         # Run test suite
make dev-logs         # View logs

# Database
make migrate          # Run migrations
make db-seed          # Seed test data
make db-reset         # Reset database
```

### Production
```bash
# Server management
make ops-diagnose
make ops-backup
docker ps | grep nuclear-ao3

# Deployment
# Use the "ops-upgrade" workflow in the Actions tab, or:
make ops-upgrade REF=v1.0.0
```

### Emergency Procedures
```bash
# Emergency rollback: make ops-upgrade REF=<the previous version>

# Emergency backup
make ops-backup

# Service restart
docker restart nuclear-ao3-staging-api-gateway
```

## 🔒 Security Checklist

- [ ] **Firewall configured** (ports 22, 80, 443 only)
- [ ] **SSH keys only** (no password auth)
- [ ] **SSL certificates** (auto-renewed via Let's Encrypt)
- [ ] **Database passwords** (strong, unique)
- [ ] **JWT secrets** (random, secure)
- [ ] **Rate limiting** (configured in Caddy)
- [ ] **Security headers** (HSTS, CSP, etc.)
- [ ] **Regular updates** (OS and Docker images)
- [ ] **Backup encryption** (sensitive data)
- [ ] **Log monitoring** (suspicious activity)

## 🆘 Troubleshooting

### Common Issues

**Services won't start:**
```bash
# Check logs
docker logs nuclear-ao3-staging-api-gateway

# Check disk space
df -h

# Check memory
free -h

# Restart services
docker-compose restart
```

**Database connection failed:**
```bash
# Check PostgreSQL status
docker exec nuclear-ao3-postgres pg_isready

# Test connection
docker exec nuclear-ao3-postgres psql -U ao3_user -d ao3_nuclear -c "SELECT 1;"

# Reset database (CAUTION!)
docker-compose down
docker volume rm nuclear-ao3-postgres-data
docker-compose up -d postgres
```

**Health checks failing:**
```bash
# Check API Gateway
curl http://localhost:8080/health

# Check individual services
curl http://localhost:8081/health  # auth-service
curl http://localhost:8082/health  # work-service

# View detailed logs
docker-compose logs --tail 100 app
```

### Getting Help

1. **Check everything**: `make ops-diagnose`
2. **Review GitHub Actions**: Check failed deployment logs
3. **Server resources**: `htop`, `df -h`, `docker stats`
4. **Network connectivity**: `curl`, `ping`, `netstat`

## 🎉 Success Checklist

After deployment, verify:

- [ ] **Website loads**: `https://your-domain.com`
- [ ] **SSL certificate**: Green lock in browser
- [ ] **Health endpoints**: `/health` returns 200
- [ ] **User registration**: Can create account
- [ ] **Work upload**: Can post a work
- [ ] **Search functionality**: Can search works
- [ ] **Comments system**: Can post comments
- [ ] **Export features**: Can export works
- [ ] **Notification system**: Receives notifications
- [ ] **Mobile responsive**: Works on mobile devices
- [ ] **Performance**: Page loads < 2 seconds

## 📞 Support

- **Documentation**: This guide + README.md
- **Health, backup, restore, upgrade**: `scripts/ops/` (see docs/OPS_PIPELINES.md)
- **GitHub Actions**: `.github/workflows/`
- **Issues**: GitHub Issues tab

---

**🎊 Congratulations!** You now have a production-ready Nuclear AO3 deployment with zero-downtime blue-green deployments, comprehensive monitoring, and automated backups.