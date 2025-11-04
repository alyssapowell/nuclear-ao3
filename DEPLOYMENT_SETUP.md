# 🚀 Nuclear AO3 Deployment Setup Guide

## Quick Server Setup Options

### Option 1: DigitalOcean Droplet (Recommended)

**Cost**: $6/month | **Setup**: 5 minutes

1. **Create Account**: [DigitalOcean](https://digitalocean.com)
2. **Create Droplet**:
   - **Image**: Ubuntu 22.04 LTS
   - **Size**: Basic $6/month (1GB RAM, 1 vCPU, 25GB SSD)
   - **Region**: Choose closest to you
   - **Authentication**: SSH keys (recommended) or password
3. **Note the IP address** after creation

### Option 2: Hetzner Cloud (Most Cost-Effective)

**Cost**: €4/month | **Setup**: 5 minutes

1. **Create Account**: [Hetzner Cloud](https://console.hetzner.cloud)
2. **Create Server**:
   - **Image**: Ubuntu 22.04
   - **Type**: CX11 (€4.51/month)
   - **Location**: Choose closest to you
   - **SSH Key**: Upload your public key
3. **Note the IP address** after creation

### Option 3: Test with ngrok (Free, Immediate)

**Cost**: Free | **Setup**: 2 minutes

1. **Install ngrok**: [ngrok.com](https://ngrok.com)
2. **Expose local deployment**:
   ```bash
   # In one terminal, run your local deployment
   docker-compose up
   
   # In another terminal, expose it
   ngrok http 8088
   ```
3. **Use the ngrok URL** as your domain

## 🔑 GitHub Secrets Configuration

Once you have a server, configure these secrets in your GitHub repository:

**Go to**: GitHub Repository → Settings → Secrets and variables → Actions

### Required Secrets:

```
SERVER_SSH_KEY     = Your private SSH key (entire key including headers)
SERVER_HOST        = Your server IP address (e.g., 64.23.45.67)
SERVER_USER        = SSH username (usually 'root' or 'ubuntu')
DOMAIN             = Your domain name (e.g., nuclear-ao3.com or ngrok URL)
JWT_SECRET         = Random secure string (generate with: openssl rand -base64 32)
```

### Example SSH Key Format:
```
-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAFwAAAAdzc2gtcn...
[rest of your private key]
-----END OPENSSH PRIVATE KEY-----
```

## 🚀 Deployment Commands

### Manual Deployment
```bash
# Deploy to staging
gh workflow run "🚀 Deploy Nuclear AO3" \
  --field environment=staging \
  --field git_ref=main \
  --field strategy=docker-compose

# Deploy to production  
gh workflow run "🚀 Deploy Nuclear AO3" \
  --field environment=production \
  --field git_ref=main \
  --field strategy=docker-compose
```

### Web Interface
1. Go to: **Actions** tab in your GitHub repository
2. Click: **🚀 Deploy Nuclear AO3**
3. Click: **Run workflow**
4. Choose your options and click **Run workflow**

## 🔧 Server Preparation (One-time setup)

### For DigitalOcean/Hetzner:

```bash
# SSH into your server
ssh root@YOUR_SERVER_IP

# Update system
apt update && apt upgrade -y

# Install essential packages
apt install -y curl git ufw

# Configure firewall
ufw allow ssh
ufw allow 80
ufw allow 443
ufw allow 8088
ufw --force enable

# Create deployment user (optional but recommended)
adduser deploy
usermod -aG sudo deploy
# Copy SSH keys to deploy user if desired
```

## 🌐 Domain Setup (Optional)

### With Custom Domain:
1. **Buy domain** from any registrar (Namecheap, Cloudflare, etc.)
2. **Point A record** to your server IP
3. **Use domain** in DOMAIN secret

### With ngrok (Free):
1. **Run ngrok**: `ngrok http 8088`
2. **Copy URL**: e.g., `https://abc123.ngrok.io`
3. **Use URL** in DOMAIN secret

## ✅ Testing Your Deployment

After deployment succeeds:

```bash
# Test API health
curl https://YOUR_DOMAIN/health

# Check service status on server
ssh root@YOUR_SERVER_IP
cd /opt/nuclear-ao3
docker-compose ps
```

## 🎯 Quick Start Checklist

- [ ] Choose hosting option (DigitalOcean recommended)
- [ ] Create server and note IP address
- [ ] Generate SSH key if needed
- [ ] Configure GitHub secrets
- [ ] Run deployment workflow
- [ ] Test application health
- [ ] Set up domain (optional)

## 🆘 Troubleshooting

### Common Issues:

**Deployment fails with SSH error**:
- Check SSH key format (include headers/footers)
- Ensure key has no passphrase
- Verify server IP and username

**Services won't start**:
- Check server has enough RAM (minimum 1GB)
- Verify ports aren't blocked by firewall
- Check Docker is installed and running

**Health check fails**:
- Services may need more time to start
- Check individual service logs: `docker-compose logs SERVICE_NAME`
- Verify database initialization completed

### Getting Help:
1. Check deployment logs in GitHub Actions
2. SSH to server and run: `docker-compose logs`
3. Check individual service health: `curl localhost:8088/health`

---

🎉 **Ready to deploy Nuclear AO3 to the world!**