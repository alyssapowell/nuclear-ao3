# What to run it on

The smallest setup that is still safe to run a public archive on, what it must keep no matter how small it is, and when to grow.

Numbers marked **measured** come from running the app. Numbers marked **estimate** are the usual needs of each piece of software in this setup, and should be replaced with measurements once real data is imported (see docs/IMPORTER.md).

## The smallest safe setup

One Linux server (a virtual machine is fine):

| | Minimum | Why |
|---|---|---|
| CPU | 2 cores | Postgres and the app can work at the same time; one core makes every slow query slow down the whole site |
| Memory | 4 GB | see the budget below |
| Disk | 40 GB SSD | database, 14 local backups, logs, 30 days of monitoring |
| OS | any current Linux with Docker and Docker Compose | |
| Network | a public IPv4 address, ports 80 and 443 open, everything else closed | Caddy gets and renews the HTTPS certificate on its own |

It runs everything in `docker-compose.yml` on that one machine, including monitoring.

### Memory budget

| Service | Memory | |
|---|---|---|
| app (the whole backend) | about 25 MB idle | **measured** with a small database; grows with traffic |
| Postgres | about 1 GB | **estimate**: `shared_buffers` is 256 MB, plus connections and the operating system's file cache, which Postgres relies on |
| Prometheus | about 200 to 400 MB | **estimate** for 30 days of this app's metrics |
| Grafana | about 100 to 200 MB | **estimate** |
| Caddy, node-exporter | under 100 MB together | **estimate** |
| Left for the operating system and file cache | about 2 GB | the more of the database fits in cache, the faster the site |

The app itself is one 33 MB program (**measured**), and the website is 1.4 MB of files (**measured**). Neither needs Node, Ruby or a separate worker process on the server.

### Don't go smaller by turning these off

Even the smallest archive needs all of the following. Without them, problems are found by readers instead of by you, and some can't be fixed afterwards.

1. **The database on a disk that survives restarts.** It's the `postgres_data` volume. Never run Postgres on a disk that's thrown away when the machine is replaced.
2. **Backups that leave the server.** The **ops-backup** workflow runs every day and keeps the newest 14 on the server, but a copy on the same machine is lost with the machine. Set up the off-server copy (docs/BACKUPS.md): every backup is then encrypted and copied to storage at another company, and the daily run goes red if the copy fails. Until it's set up, **ops-diagnose** reports it as a problem every day.
3. **A restore you have actually done.** Once a month, restore the latest backup onto a spare machine with **ops-restore**, and check that the site works. A backup that has never been restored can't be trusted.
4. **The signing key, saved away from the server** (see "Set up the signing key" in docs/RUNBOOK.md). It isn't in the backups on purpose.
5. **Logs kept for a while, with a limit.** Every service keeps about 100 MB of logs (5 files of 20 MB), which is days to weeks on a small site. That's enough to see what happened before a problem, and can't fill the disk.
6. **Monitoring.** Prometheus keeps 30 days of numbers, and the alerts point at runbook entries. Turning it off saves about 500 MB of memory and leaves you blind; get a bigger server instead.
7. **Someone who hears about problems.** Alerts only show in Grafana until notifications are set up (docs/MONITORING.md). Also add a free external uptime check on `https://your.archive/health`, so you hear about it even when the whole server is down, including its monitoring.
8. **HTTPS.** Caddy handles it. The app refuses to start if `SITE_URL` isn't `https://`.

## When to grow

Watch the Archive overview dashboard. Grow when one of these stays true for a week, not after one busy evening:

- **Slow pages (p95 over 1 second) while the CPU is busy.** Add cores.
- **Slow pages while the database is working hard, or memory is nearly all used.** Add memory first. Postgres gets faster when more of the database fits in memory.
- **The disk is over 70% full.** Add disk before it reaches the alert.

When one server isn't enough, these are the steps, in this order. Each is a change of settings, not of code:

1. **A separate database server.** Run Postgres on its own machine, or use a managed Postgres service, and point `DATABASE_URL` at it. This is the biggest single step. The database is what grows with the archive; the app doesn't.
2. **More copies of the app** behind Caddy. The app keeps no state of its own, and background jobs are shared safely between copies (see "How background jobs were done" in docs/DECISIONS.md).
3. **A read replica of the database**, for searches and reading. This needs code support that isn't written yet.
4. **A CDN in front of Caddy**, for the website's files and public pages. For a public archive this often comes first, not last; see the next section.

## Staying up when a whole data centre doesn't

Growing the server handles more readers. It doesn't help when the data centre itself has a problem, or when someone floods the site. For that, the archive needs to be in more than one place. In order of value for effort:

### 1. An edge network (CDN) in front

A CDN (Cloudflare, Fastly, Bunny and others) answers visitors from data centres around the world and passes only what it can't answer to your server. It absorbs most floods before they reach you, and serves cached public pages quickly everywhere. Most of an archive's traffic is logged-out reading, which is exactly what a CDN is good at.

Two settings matter once a CDN is in front:

- **Tell Caddy to believe the CDN about who the visitor is.** Every request now arrives from the CDN's addresses. Without this, the app sees a few CDN addresses instead of real visitors: rate limits would treat everyone as one visitor, and logs lose real addresses. In the `Caddyfile`, add a global block listing your CDN's published address ranges and its visitor header. For Cloudflare, for example:

  ```
  {
      servers {
          trusted_proxies static 173.245.48.0/20 103.21.244.0/22 ...
          client_ip_headers CF-Connecting-IP
      }
  }
  ```

  Copy the full, current list from the CDN's own page of published address ranges (they change now and then), and don't add anything else. The app already trusts only Caddy (`TRUSTED_PROXIES`), so the real address is carried all the way through.
- **Only let the CDN reach the server.** In the server's firewall, accept ports 80 and 443 only from the CDN's ranges. Otherwise someone who finds the server's own address can go around the CDN.

The app marks logged-out API answers as cacheable, so a CDN can use them. See "Caching at the CDN" below.

#### Caching at the CDN

**What is cached.** Only successful (200) answers to GET requests with no `Authorization` header, on a fixed list of public read routes: the works list, a work and its chapters and comments, search and suggestions, tag pages and tag lists, series, collections, and a user's public bookmarks. These carry `Cache-Control: public, max-age=60, stale-while-revalidate=300`: the CDN may reuse an answer for a minute, and for up to five more minutes serve it while fetching a fresh one. The list is `publicRoutes` in `backend/internal/platform/cache.go`.

**What is never cached.** Everything else, including every answer to a logged-in request, every error and redirect, anything under login, account, admin, notifications, subscriptions and exports, and kudos (which says whether your address already gave kudos). These carry `Cache-Control: private, no-store`. Every answer carries `Vary: Authorization`. Login uses a token in the `Authorization` header and the app sets no login cookies, so the CDN needs no cookie rules. Don't add a CDN rule that caches `/api` regardless of origin headers ("cache everything"); the app's headers are what keep personal answers out of the cache.

**Hidden works stay visible for a short time.** When a work is hidden or newly restricted, logged-out visitors can still be served the old copy by the CDN for up to a minute, plus the five-minute stale window. For something harmful, purge it (below) rather than wait.

**Purging a URL in an emergency.** Every CDN has a way to remove one address from its cache. In Cloudflare it is Caching, Configuration, "Purge Cache", "Custom Purge", then enter the full URLs. The website's own pages are plain files and hold no work text; only the API addresses need purging. For a work with id `W`, purge `https://YOUR-DOMAIN/api/v1/work/W`, `https://YOUR-DOMAIN/api/v1/works/W`, `.../chapters`, `.../chapters/<chapter id>`, `.../comments`, and the same `/work/W/...` addresses. Search results, tag pages and lists that showed the work clear on their own within the cache time; purge those too if they show a title or summary that must go. Query strings count as part of the address.

**Not cached: reading a chapter.** Reading a chapter counts one view (a hit), and a copy answered by the CDN would never reach the app to be counted, so that one address is always passed through. Caching chapter text too, while still counting views, needs the reading page to report the view in a separate small request that is never cached. That isn't built yet.

### 2. A warm standby in another region

A second server in a different region, with a copy of the database that Postgres keeps up to date by itself (streaming replication, usually a few seconds behind), and the app ready to run. If the main region fails, switch to the standby. The archive is back in minutes, with at most the last few seconds of changes lost.

This needs:

- Streaming replication from the main Postgres to the standby.
- A failover step that promotes the standby, points the app and the CDN at it, and refuses to run while the old main is still accepting writes (two databases both taking changes is the one mistake that can't be cleaned up). This should be an **ops-failover** workflow like the others, not a page of instructions. It isn't written yet.
- Practising it, like the monthly restore drill. A failover that has never been done is a plan, not a standby.

Off-server backups (docs/BACKUPS.md) are the fallback when even the standby is lost: slower to restore, but in a third place.

### 3. Read-only copies in several regions

Read-only copies of the database near readers in other parts of the world, kept current from the primary's write-ahead log (WAL). Worth doing once measurements show distant readers are slow even with the CDN. What makes it work, and what goes wrong without each piece:

- **Replication slots, with a limit.** A slot makes the primary keep WAL until a replica has it, so a lagging replica can always catch up. But a replica that is down then makes the primary keep WAL until its disk is full, which takes the primary down too. Set `max_slot_wal_keep_size`, and alert on slot lag.
- **WAL archived to object storage as well** (with a tool made for it, such as WAL-G or pgBackRest). Replicas can catch up from the archive after a long gap without the primary holding WAL for them. It also allows point-in-time recovery: restoring to any second, where the daily dumps in docs/BACKUPS.md can lose up to a day. This is worth having even with no replicas.
- **Asynchronous across regions.** Waiting for a distant replica to confirm every write would add its round trip to every save.
- **Chained replicas.** Far-away replicas copy from a nearer replica, so the primary sends its WAL across regions once rather than once per replica.
- **`hot_standby_feedback` off**, for this app's short read queries: on, it protects long replica queries at the cost of the primary keeping old row versions longer (table bloat).
- **Reading your own writes.** Someone who just posted a chapter must not be sent to a replica that is a second behind and doesn't have it yet.

What still needs code:
- A second database setting for the replicas.
- Sending logged-out reads (work pages, search, tag pages) to the nearest replica while its lag is under a few seconds, and to the primary otherwise.
- Sending a user's reads to the primary for a short time after they change anything.
- A replica-lag metric and alert on the dashboard.

### What isn't recommended

Running the archive "active-active", accepting changes in several regions at once. Postgres has one place that accepts writes, by design. Systems that accept writes everywhere have to settle conflicting changes forever after, and that is a full-time specialist job. A CDN, a warm standby and backups in a third place cover nearly all of the same failures.

## What hasn't been measured

Nobody has run this with an archive the size of AO3's yet. The numbers above are for a small archive. Before anyone relies on this at large scale:

- Import a real copy of the data (docs/IMPORTER.md), and measure database size, memory and query times with it.
- Load-test the busiest pages (work pages, tag listings, search) at realistic traffic.
- Measure how long backup and restore take at that size, because that sets how long an outage can last.

Until then, treat the large-scale steps as the plan, not as tested advice.
