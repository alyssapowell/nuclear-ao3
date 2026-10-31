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
| Redis | at most 256 MB | capped in `docker-compose.yml` |
| Prometheus | about 200 to 400 MB | **estimate** for 30 days of this app's metrics |
| Grafana | about 100 to 200 MB | **estimate** |
| Caddy, node-exporter | under 100 MB together | **estimate** |
| Left for the operating system and file cache | about 2 GB | the more of the database fits in cache, the faster the site |

The app itself is one 33 MB program (**measured**), and the website is 1.4 MB of files (**measured**). Neither needs Node, Ruby or a separate worker process on the server.

### Don't go smaller by turning these off

Even the smallest archive needs all of the following. Without them, problems are found by readers instead of by you, and some can't be fixed afterwards.

1. **The database on a disk that survives restarts.** It's the `postgres_data` volume. Never run Postgres on a disk that's thrown away when the machine is replaced.
2. **Backups that leave the server.** The **ops-backup** workflow runs every day and keeps the newest 14 on the server, but a copy on the same machine is lost with the machine. Copy `backups/` somewhere else every day: another provider's object storage, or another machine. **This copy isn't automated yet.**
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
4. **A CDN in front of Caddy**, for the website's files and public pages.

## What hasn't been measured

Nobody has run this with an archive the size of AO3's yet. The numbers above are for a small archive. Before anyone relies on this at large scale:

- Import a real copy of the data (docs/IMPORTER.md), and measure database size, memory and query times with it.
- Load-test the busiest pages (work pages, tag listings, search) at realistic traffic.
- Measure how long backup and restore take at that size, because that sets how long an outage can last.

Until then, treat the large-scale steps as the plan, not as tested advice.
