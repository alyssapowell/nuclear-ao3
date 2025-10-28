#!/bin/bash

# Nuclear AO3 - Hetzner Server Setup Script
# Run this on a fresh Hetzner Ubuntu 22.04 VPS

set -e

echo "🚀 Setting up Nuclear AO3 server..."

# Update system
echo "📦 Updating system packages..."
apt update && apt upgrade -y

# Install essential packages
echo "🔧 Installing essential packages..."
apt install -y \
    curl \
    wget \
    git \
    ufw \
    fail2ban \
    htop \
    unzip \
    jq \
    bc \
    software-properties-common \
    apt-transport-https \
    ca-certificates \
    gnupg \
    lsb-release

# Install Docker
echo "🐳 Installing Docker..."
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
apt update
apt install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

# Start and enable Docker
systemctl start docker
systemctl enable docker

# Add user to docker group (if deployment user provided)
if [ ! -z "$1" ]; then
    usermod -aG docker $1
    echo "✅ Added user $1 to docker group"
fi

# Install Caddy (reverse proxy with automatic HTTPS)
echo "🔒 Installing Caddy web server..."
apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
apt update
apt install -y caddy

# Configure firewall
echo "🔥 Configuring firewall..."
ufw default deny incoming
ufw default allow outgoing
ufw allow ssh
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable

# Configure fail2ban
echo "🛡️ Configuring fail2ban..."
cp /etc/fail2ban/jail.conf /etc/fail2ban/jail.local

# Create deployment directory
echo "📁 Creating deployment directories..."
mkdir -p /opt/nuclear-ao3
mkdir -p /var/log/nuclear-ao3
chown -R $1:$1 /opt/nuclear-ao3 /var/log/nuclear-ao3 2>/dev/null || true

# Create Caddyfile for Nuclear AO3
echo "⚙️ Creating Caddy configuration..."
cat > /etc/caddy/Caddyfile << 'EOF'
# Nuclear AO3 Caddy Configuration
# Automatic HTTPS, load balancing, and reverse proxy

# Main site - production
{$DOMAIN:nuclear-ao3.com} {
    # Enable compression
    encode gzip

    # Security headers
    header {
        # Prevent clickjacking
        X-Frame-Options "DENY"
        # Prevent MIME type sniffing
        X-Content-Type-Options "nosniff"
        # XSS protection
        X-XSS-Protection "1; mode=block"
        # Referrer policy
        Referrer-Policy "strict-origin-when-cross-origin"
        # Remove server info
        -Server
    }

    # Health check endpoint
    handle /health {
        reverse_proxy localhost:8080
    }

    # GraphQL API endpoint
    handle /graphql {
        reverse_proxy localhost:8080
    }

    # REST API endpoints - route through API Gateway
    handle /api/* {
        reverse_proxy localhost:8080
    }

    # Static assets and frontend
    handle /* {
        reverse_proxy localhost:3000
    }

    # Request logging
    log {
        output file /var/log/caddy/nuclear-ao3.log {
            roll_size 100MB
            roll_keep 5
        }
        format json
    }
}

# Monitoring dashboard (Grafana)
monitoring.{$DOMAIN:nuclear-ao3.com} {
    reverse_proxy localhost:3001
    
    # Basic auth for monitoring
    basicauth {
        admin {$GRAFANA_PASSWORD_HASH}
    }
    
    log {
        output file /var/log/caddy/monitoring.log
        format json
    }
}

# Staging site
staging.{$DOMAIN:nuclear-ao3.com} {
    reverse_proxy localhost:8080
    
    # Same security headers
    header {
        X-Frame-Options "DENY"
        X-Content-Type-Options "nosniff"
        X-XSS-Protection "1; mode=block"
        Referrer-Policy "strict-origin-when-cross-origin"
        -Server
    }
    
    encode gzip
    
    log {
        output file /var/log/caddy/staging.log
        format json
    }
}

# Redirect www to non-www
www.{$DOMAIN:nuclear-ao3.com} {
    redir https://{$DOMAIN:nuclear-ao3.com}{uri} permanent
}
EOF

# Create log directory for Caddy
mkdir -p /var/log/caddy
chown caddy:caddy /var/log/caddy

# Reload Caddy
systemctl reload caddy
systemctl enable caddy

# Create systemd service for Nuclear AO3
cat > /etc/systemd/system/nuclear-ao3.service << 'EOF'
[Unit]
Description=Nuclear AO3 Platform
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/opt/nuclear-ao3
ExecStart=/usr/bin/docker-compose up -d
ExecStop=/usr/bin/docker-compose down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
EOF

# Set up log rotation
cat > /etc/logrotate.d/nuclear-ao3 << 'EOF'
/var/log/nuclear-ao3/*.log {
    daily
    missingok
    rotate 14
    compress
    delaycompress
    notifempty
    create 0644 nuclear-ao3 nuclear-ao3
}
EOF

# Create monitoring script
cat > /opt/nuclear-ao3/monitor.sh << 'EOF'
#!/bin/bash
# Nuclear AO3 monitoring script

check_services() {
    services=("nuclear-ao3-frontend" "nuclear-ao3-gateway" "nuclear-ao3-auth" "nuclear-ao3-works" "nuclear-ao3-tags" "nuclear-ao3-search")
    
    for service in "${services[@]}"; do
        if docker ps | grep -q "$service"; then
            echo "✅ $service is running"
        else
            echo "❌ $service is down"
            # Restart service
            docker start "$service" || echo "Failed to start $service"
        fi
    done
}

check_health() {
    if curl -f http://localhost:8080/health >/dev/null 2>&1; then
        echo "✅ API Gateway health check passed"
    else
        echo "❌ API Gateway health check failed"
    fi
    
    if curl -f http://localhost:3000 >/dev/null 2>&1; then
        echo "✅ Frontend health check passed"
    else
        echo "❌ Frontend health check failed"
    fi
}

echo "$(date): Nuclear AO3 Health Check"
check_services
check_health
echo "---"
EOF

chmod +x /opt/nuclear-ao3/monitor.sh

# Create backup script for Nuclear AO3
cat > /opt/nuclear-ao3/backup.sh << 'EOF'
#!/bin/bash
# Backup script for Nuclear AO3

BACKUP_DIR="/opt/nuclear-ao3/backups"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR

echo "Creating backup for Nuclear AO3 - $DATE..."

# Backup PostgreSQL database
docker exec nuclear-ao3-postgres pg_dump -U ao3_user ao3_nuclear | gzip > $BACKUP_DIR/postgres_$DATE.sql.gz

# Backup Docker volumes
docker run --rm -v nuclear-ao3_postgres_data:/data -v $BACKUP_DIR:/backup ubuntu tar czf /backup/postgres_data_$DATE.tar.gz -C /data .
docker run --rm -v nuclear-ao3_es_data:/data -v $BACKUP_DIR:/backup ubuntu tar czf /backup/elasticsearch_data_$DATE.tar.gz -C /data .
docker run --rm -v nuclear-ao3_redis_data:/data -v $BACKUP_DIR:/backup ubuntu tar czf /backup/redis_data_$DATE.tar.gz -C /data .

# Export running containers (for restore reference)
docker ps --format "table {{.Names}}\t{{.Image}}\t{{.Ports}}" > $BACKUP_DIR/containers_$DATE.txt

# Compress old backups (keep last 7 days)
find $BACKUP_DIR -name "*.tar.gz" -mtime +7 -delete
find $BACKUP_DIR -name "*.sql.gz" -mtime +7 -delete

echo "Backup completed: $DATE"
EOF

chmod +x /opt/nuclear-ao3/backup.sh

# Add monitoring and backup to crontab
(crontab -l 2>/dev/null; echo "*/2 * * * * /opt/nuclear-ao3/enhanced-monitoring.sh main >> /var/log/nuclear-ao3/monitor.log 2>&1") | crontab -
(crontab -l 2>/dev/null; echo "0 2 * * * /opt/nuclear-ao3/backup.sh >> /var/log/nuclear-ao3/backup.log 2>&1") | crontab -

echo "🎉 Nuclear AO3 server setup complete!"
echo ""
echo "Next steps:"
echo "1. Point your domain DNS to this server's IP"
echo "2. Set up GitHub Actions secrets:"
echo "   - HETZNER_HOST: $(curl -s ifconfig.me)"
echo "   - HETZNER_USER: $(whoami)"
echo "   - HETZNER_SSH_KEY: (your private SSH key)"
echo "   - DOMAIN: your-domain.com"
echo "3. Push to main branch to trigger deployment"
echo ""
echo "📊 Server Status:"
echo "Docker: $(docker --version)"
echo "Caddy: $(caddy version)"
echo "UFW: $(ufw status | head -1)"
echo ""
echo "🔗 Services will be available at:"
echo "Production: https://your-domain.com"
echo "Staging: https://staging.your-domain.com"
echo "Monitoring: https://monitoring.your-domain.com"
echo ""
echo "🔐 Set environment variables:"
echo "export DOMAIN=your-domain.com"
echo "export GRAFANA_PASSWORD_HASH=\$(caddy hash-password your-password)"