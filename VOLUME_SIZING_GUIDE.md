# Nuclear AO3 Volume Sizing Guide 📊

## 🗄️ Cost-Optimized Docker Volumes

All persistent data is stored in **5 Docker volumes** to minimize cloud provider costs (most charge per volume + per GB).

### **Volume Strategy: Large + Shared**

| Volume | Contents | Size | Monthly Cost* | Growth Rate |
|--------|----------|------|---------------|-------------|
| `postgres_data` | **Database only** | **20-50GB** | €2-5/month | ~1-5GB/month |
| `es_data` | **Search indexes only** | **10-20GB** | €1-2/month | ~500MB/month |
| `backup_data` | **Backups only** | **50-100GB** | €5-10/month | ~2-10GB/month |
| `monitoring_data` | **Shared**: Redis + Prometheus + Grafana + Loki | **20GB** | €2/month | ~1.5GB/month |
| `config_data` | **Shared**: Caddy SSL + config + logs | **10GB** | €1/month | ~100MB/month |

**\*Hetzner pricing: ~€0.10/GB/month for Block Storage*

### **Shared Volume Contents**

**`monitoring_data` (20GB shared):**
- Redis cache/sessions (~2-5GB)
- Prometheus metrics (~10GB)  
- Grafana dashboards (~500MB)
- Loki logs (~5-10GB)

**`config_data` (10GB shared):**
- Caddy SSL certificates (~100MB)
- Caddy configuration cache (~50MB)
- Caddy access logs (~2GB)
- Spare space for growth (~8GB)

## 💾 Total Storage Requirements & Costs

### **Cost-Optimized Production (Recommended)**
```
Volume Costs (Hetzner Block Storage):
├── postgres_data (30GB):    €3/month
├── es_data (15GB):          €1.50/month  
├── backup_data (75GB):      €7.50/month
├── monitoring_data (20GB):  €2/month
├── config_data (10GB):      €1/month
└── Total volume cost:       €15/month

Server Cost (CPX31):         €16/month
Total monthly cost:          €31/month
```

### **Volume Size Planning**

**Small Community Setup:**
- postgres_data: 20GB
- es_data: 10GB  
- backup_data: 50GB
- monitoring_data: 20GB
- config_data: 10GB
- **Total**: 110GB, **~€11/month** for volumes

**Medium Community Setup:**
- postgres_data: 30GB
- es_data: 15GB
- backup_data: 75GB  
- monitoring_data: 20GB
- config_data: 10GB
- **Total**: 150GB, **~€15/month** for volumes

**Large Community Setup:**
- postgres_data: 50GB
- es_data: 20GB
- backup_data: 100GB
- monitoring_data: 20GB
- config_data: 10GB
- **Total**: 200GB, **~€20/month** for volumes

## 🏗️ Hetzner Server Recommendations

### **Development/Testing**
```
CPX21: 3 vCPU, 4GB RAM, 80GB SSD
├── Sufficient for: Testing, small user base
├── Cost: ~€8/month
└── Limitations: Limited concurrent users
```

### **Production (Recommended)**
```
CPX31: 4 vCPU, 8GB RAM, 160GB SSD  
├── Sufficient for: 100-500 concurrent users
├── Cost: ~€16/month
└── Expansion: Can add volumes as needed
```

### **High-Traffic Production**
```
CPX41: 8 vCPU, 16GB RAM, 240GB SSD
├── Sufficient for: 500+ concurrent users  
├── Cost: ~€32/month
└── Features: Better performance, more headroom
```

## 📈 Growth Planning

### **Expected Data Growth**

**Small Community (50 active users):**
- Works: ~20 new/month = ~500MB/month
- Comments: ~100 new/month = ~50MB/month
- Search indexes: ~200MB/month
- **Total growth**: ~1GB/month

**Medium Community (200 active users):**
- Works: ~100 new/month = ~2.5GB/month
- Comments: ~500 new/month = ~250MB/month
- Search indexes: ~1GB/month
- **Total growth**: ~4GB/month

**Large Community (500+ active users):**
- Works: ~300 new/month = ~7.5GB/month
- Comments: ~1500 new/month = ~750MB/month
- Search indexes: ~3GB/month
- **Total growth**: ~12GB/month

### **Backup Storage**

Backups are automatically retained for **30 days** by default:

```
Backup Size = (PostgreSQL + Redis + Elasticsearch) × 30 days
Small:  (500MB + 50MB + 200MB) × 30 = ~22GB
Medium: (2.5GB + 200MB + 1GB) × 30 = ~110GB  
Large:  (7.5GB + 500MB + 3GB) × 30 = ~330GB
```

## 🔧 Volume Management Commands

### **Create Cost-Optimized Volumes**
```bash
# Create 5 volumes (instead of 10+ individual volumes)
docker volume create nuclear-ao3_postgres_data    # 20-50GB
docker volume create nuclear-ao3_es_data          # 10-20GB  
docker volume create nuclear-ao3_backup_data      # 50-100GB
docker volume create nuclear-ao3_monitoring_data  # 20GB (shared)
docker volume create nuclear-ao3_config_data      # 10GB (shared)

# For Hetzner Block Storage (production):
# 1. Create volumes in Hetzner Console
# 2. Attach to server
# 3. Format and mount:
sudo mkfs.ext4 /dev/sdb  # postgres_data
sudo mkfs.ext4 /dev/sdc  # es_data
sudo mkfs.ext4 /dev/sdd  # backup_data
sudo mkfs.ext4 /dev/sde  # monitoring_data
sudo mkfs.ext4 /dev/sdf  # config_data
```

### **Check Volume Usage**
```bash
# See all volumes and their sizes
docker system df -v

# Check specific volume
docker volume inspect nuclear-ao3_postgres_data

# See volume usage in detail
docker run --rm -v nuclear-ao3_postgres_data:/data alpine du -sh /data
```

### **Backup Volume Contents**
```bash
# Backup a volume to tar file
docker run --rm \
  -v nuclear-ao3_postgres_data:/source:ro \
  -v $(pwd):/backup \
  alpine tar czf /backup/postgres_backup.tar.gz -C /source .

# Restore volume from tar file  
docker run --rm \
  -v nuclear-ao3_postgres_data:/target \
  -v $(pwd):/backup \
  alpine tar xzf /backup/postgres_backup.tar.gz -C /target
```

### **Clean Up Old Data**
```bash
# Remove old backups (automated by backup script)
./scripts/backup-restore.sh cleanup

# Prune unused volumes (CAREFUL!)
docker volume prune -f

# Clean up Docker resources
docker system prune -f
```

## 🚨 Monitoring Disk Usage

### **Automated Monitoring**
The health monitor script (`scripts/health-monitor.sh`) automatically checks:
- Disk usage > 80% = warning
- Disk usage > 90% = critical alert
- Volume size growth trends

### **Manual Monitoring**
```bash
# Check overall disk usage
df -h

# Check Docker space usage
docker system df

# Check volume sizes
docker volume ls
docker run --rm -v nuclear-ao3_postgres_data:/data alpine du -sh /data
```

### **Grafana Dashboards**
Access monitoring at `https://your-domain.com:3001`:
- **System Overview**: Disk usage, volume growth
- **Database Metrics**: PostgreSQL size, query performance  
- **Backup Status**: Backup success rate, storage usage

## 📋 Pre-Deployment Checklist

- [ ] **Server has sufficient disk space** (minimum 100GB free)
- [ ] **Backup volume created** with adequate size
- [ ] **Monitoring enabled** for disk usage alerts
- [ ] **Backup retention configured** (default 30 days)
- [ ] **Growth projections calculated** for your expected usage
- [ ] **Scaling plan ready** for when limits are approached

## 🔄 Volume Migration & Scaling

### **Adding More Storage**
```bash
# Option 1: Add external volume (Hetzner Block Storage)
# Create 100GB volume in Hetzner console, then:
sudo mkfs.ext4 /dev/sdb
sudo mount /dev/sdb /mnt/extra-storage

# Option 2: Upgrade server to larger disk
# Use Hetzner console to resize server + disk

# Option 3: Move volumes to external storage
docker run --rm \
  -v nuclear-ao3_postgres_data:/source:ro \
  -v /mnt/extra-storage:/target \
  alpine cp -a /source/. /target/
```

### **Volume Backup Strategy**
1. **Automated daily backups** (handled by `backup-restore.sh`)
2. **Weekly full system snapshots** (Hetzner server snapshots)
3. **Monthly off-site backups** (download critical data)
4. **Before major updates** (manual backup)

---

**💡 Tip**: Start with the **Production** setup (CPX31, 160GB) - it provides good headroom for growth and only costs ~€16/month. You can always scale up later!