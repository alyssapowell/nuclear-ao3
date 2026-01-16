# GitHub Actions Workflow Cleanup Plan

## Current State: DUPLICATES AND CONFUSION

You currently have **5 deployment workflows**, but **2 are duplicates** with the same name!

### Active Workflows (from `gh workflow list`)

| Workflow Name | File | Lines | ID | Status |
|---------------|------|-------|-----|--------|
| Deploy Application Services | `deploy-application.yml` | 113 | 204998789 | ✅ Keep |
| **Deploy to Hetzner** | `deploy-hetzner.yml` | **227** | 203893216 | ⚠️ DUPLICATE |
| Deploy Individual Service | `deploy-individual-service.yml` | 236 | 204998790 | ✅ Keep |
| Deploy Infrastructure | `deploy-infrastructure.yml` | 111 | 204998791 | ✅ Keep |
| **Deploy to Hetzner** | `deploy-manual.yml` | **165** | 203899997 | ⚠️ DUPLICATE |

### Extra Files
- `deploy-hetzner.yml.backup` - Old backup file (can delete)

---

## Comparison: deploy-hetzner.yml vs deploy-manual.yml

Both workflows:
- ✅ Have the **same name**: "Deploy to Hetzner"
- ✅ Have the **same purpose**: Full deployment (everything)
- ✅ Accept same inputs: environment (staging/production), git_ref
- ✅ Deploy to the same server

**Key Differences:**

| Feature | deploy-hetzner.yml | deploy-manual.yml |
|---------|-------------------|-------------------|
| Lines of code | 227 lines | 165 lines |
| Has SSH test step | ✅ Yes | ❌ No |
| Has detailed logging | ✅ Yes (more verbose) | ⚠️ Less verbose |
| Docker build | `docker-compose build` (sequential, FIXED) | `docker-compose build --parallel` (can crash) |
| Health checks | More comprehensive | Basic |
| Error handling | Better | Basic |
| Last updated | Jan 14, 2026 (TODAY - fixed) | Nov 4, 2025 (OLD) |

**Verdict**: `deploy-hetzner.yml` is **better** - more robust, recently updated, has the memory fix

---

## Recommended Action: Delete Redundant Workflows

### ✅ KEEP These 4 Workflows

1. **`deploy-infrastructure.yml`** - Deploy databases/monitoring only
   - postgres, redis, elasticsearch, prometheus, grafana, loki
   - Use when: Database config changes, monitoring updates

2. **`deploy-application.yml`** - Deploy application services only
   - frontend, api-gateway, auth, work, tag, search, notification, export, caddy
   - Use when: Code changes, most deployments

3. **`deploy-individual-service.yml`** - Deploy single service
   - Any individual service with optional force rebuild
   - Use when: Hotfix one service, testing, debugging
   - **MOST USEFUL FOR DAILY WORK**

4. **`deploy-hetzner.yml`** - Full deployment (nuclear option)
   - Everything at once
   - Use when: Major updates, initial deployment, things are broken
   - **Recently fixed for memory issues**

### ❌ DELETE These Files

1. **`deploy-manual.yml`** - Duplicate of deploy-hetzner.yml (older, worse version)
2. **`deploy-hetzner.yml.backup`** - Old backup file (no longer needed)

---

## Why This Cleanup Matters

### Current Problems:
1. **User confusion**: Two workflows with identical names - which one to run?
2. **Maintenance burden**: Updating deployment logic in two places
3. **Safety risk**: Old workflow (`deploy-manual.yml`) still has memory exhaustion bug
4. **Wasted CI quota**: Two workflows doing the same thing

### After Cleanup:
1. ✅ Clear purpose for each workflow
2. ✅ No duplicate names
3. ✅ All workflows have latest fixes
4. ✅ Easier to maintain

---

## Migration Plan

### Step 1: Check Which Workflow People Use
```bash
gh run list --workflow=deploy-hetzner.yml --limit 10
gh run list --workflow=deploy-manual.yml --limit 10
```

### Step 2: Delete Redundant Files
```bash
# Delete the older duplicate
git rm .github/workflows/deploy-manual.yml

# Delete the backup file
git rm .github/workflows/deploy-hetzner.yml.backup

# Commit
git commit -m "chore: remove duplicate deployment workflows

- Remove deploy-manual.yml (duplicate of deploy-hetzner.yml)
- Remove deploy-hetzner.yml.backup (old backup)
- Keep deploy-hetzner.yml as the canonical full deployment workflow"
```

### Step 3: Update Documentation
Update any docs/READMEs that reference `deploy-manual.yml` to use `deploy-hetzner.yml`

---

## Final Workflow Structure (Clean) ✅ COMPLETED

```
.github/workflows/
├── deploy-hetzner.yml             # "Redeploy Everything (Full Stack)" - nuclear option
├── deploy-infrastructure.yml      # Databases & monitoring only
├── deploy-application.yml         # Application services only
├── deploy-individual-service.yml  # Single service (most useful)
└── ci.yml                         # Test/CI workflows
```

**4 clear deployment options**, no confusion, no duplicates.

### Changes Applied:
- ✅ Renamed `deploy-hetzner.yml` to "Redeploy Everything (Full Stack)"
- ✅ Deleted `deploy-manual.yml` (duplicate)
- ✅ Deleted `deploy-hetzner.yml.backup` (old backup)

---

## Quick Reference: Which Workflow to Use?

| I Need To... | Use This Workflow |
|--------------|-------------------|
| Deploy just Elasticsearch | `deploy-individual-service.yml` (service: elasticsearch) |
| Deploy code changes | `deploy-application.yml` |
| Update database config | `deploy-infrastructure.yml` |
| Fix one broken service | `deploy-individual-service.yml` |
| Initial server setup | `deploy-hetzner.yml` |
| Everything is broken | `deploy-hetzner.yml` |
| Test a hotfix | `deploy-individual-service.yml` |

**90% of the time**: Use `deploy-individual-service.yml` or `deploy-application.yml`
