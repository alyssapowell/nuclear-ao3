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

## On a hosting platform

Some platforms run one container from your git repository, give it an HTTPS address, and offer a managed Postgres (Shebaka is one; there are many). The `Dockerfile` at the top of the repository is for them: it builds the website and the backend into one image, and the app serves both, so no Caddy is needed. The app answers only the same addresses Caddy would send it, and the website's files get the same cache rules.

This is the quickest way to try the archive out. It isn't the smallest safe setup above: the platform's own backups and monitoring replace the ones in `docker-compose.yml`, and the ops workflows (which reach the server over SSH) don't apply.

### Settings

Set these in the platform's settings for the app. Put the keys in its secret settings if it has them.

| Setting | Value |
|---|---|
| `DATABASE_URL` | the connection address of the platform's managed Postgres. Add `?sslmode=require` if it doesn't already say. |
| `SITE_URL` | the app's HTTPS address, such as `https://your-app.apps.example.com` |
| `SIGNING_KEY` | the text of a key made with `nuclear keys generate` (below) |
| `OIDC_SIGNING_KEY` | the text of a second key, made the same way |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | a real mail service. The app refuses to start without one, because nobody could confirm their account. |
| `SUPPORT_EMAIL` | optional. Where the Contact and Feedback forms are emailed. Every message is kept as a ticket either way, and admins and moderators read and answer them at `/admin/support`. |

The app listens on port 8080. If the platform checks a port, set it to 8080.

### The keys

Make them on your own computer, not on the platform, so you have the copy:

```sh
cd backend
go run ./cmd/nuclear keys generate login-key.pem
go run ./cmd/nuclear keys generate oidc-key.pem
```

Paste the whole of each file, including the `-----BEGIN` and `-----END` lines, into `SIGNING_KEY` and `OIDC_SIGNING_KEY`. If the platform's secrets box takes one `KEY=value` per line, a key with line breaks won't fit; give each key as one line of base64 instead. This puts both lines on the clipboard, ready to paste, without showing them:

```sh
printf 'SIGNING_KEY=%s\nOIDC_SIGNING_KEY=%s\n' \
  "$(base64 -i login-key.pem | tr -d '\n')" "$(base64 -i oidc-key.pem | tr -d '\n')" | pbcopy
```

(`pbcopy` is macOS; on Linux use `xclip -selection clipboard`.) The app also copes with keys whose line breaks were turned into spaces, `\n` or nothing, or that were put in quotes. Save both files in your password manager, then delete them from the computer. A key given as a setting works exactly like one given as a file (`SIGNING_KEY_FILE`); setting both is refused. To change keys later, put the old key's text in `SIGNING_KEY_PREVIOUS` for 14 days, as in docs/RUNBOOK.md.

### Check after the first deploy

- `https://YOUR-ADDRESS/health` answers.
- Register an account and confirm it from the email.
- The visitor's address reaches the app. Rate limits and the login guard count per visitor, so the app must see each visitor's own address, not the platform's. The app believes `X-Forwarded-For` only from private network addresses (`TRUSTED_PROXIES`), which is where most platforms' proxies are. If every visitor gets rate-limited together, the platform's proxy is somewhere else: ask the platform which addresses its proxy uses and set `TRUSTED_PROXIES` to them.

### Trying it out with no email

To try the archive on a platform that has no email service, set `TRIAL_NO_EMAIL=true`. Then:

- Startup no longer requires `SMTP_HOST`. Every other check (an `https://` `SITE_URL`, a strong database password, and so on) still applies and still stops the server in production.
- New accounts are confirmed at once and can log in right after signing up. Sign-up gives the same answer whether or not the address already has an account.
- No email is ever sent: not password resets, not notifications. Queued emails are dropped, and the log records only that one was not sent and its subject.
- Every page shows a banner saying email is off, and the log repeats a warning at every start.

This also removes the protection against someone signing up with another person's address. Normally an account only works once its owner has used the password and the emailed link, so a stranger cannot claim an address that is not theirs. With no email, nobody checks the address, and a lost password cannot be reset. Use it only for a private trial with people you know. Before real users arrive, turn it off and set the `SMTP_*` settings.

#### Demo data

To have something to look at, the trial can be filled with made-up accounts and works. The platform needs no shell for this:

1. Set `DEMO_PASSWORD` to a password of 12 to 40 characters, and set `DEMO_DATA=true`.
2. Deploy. After the database is updated, the app loads the demo data once; the log says what was loaded. It takes a few seconds.
3. Remove `DEMO_DATA` and deploy again. (Leaving it set does no harm while `TRIAL_NO_EMAIL` is `true`: if the demo accounts are already there, nothing is loaded. If `TRIAL_NO_EMAIL` is no longer `true`, the app refuses to start with `DEMO_DATA` set, so this can never put demo accounts in a real archive.)

With a shell, `nuclear demo-data` does the same thing. There is no command to take the demo data out again; to start clean, empty the trial's database.

What it makes: 25 accounts, `demo01@example.org` to `demo25@example.org` (example.org never receives mail), some with two or three pseuds; about 150 works in about 400 chapters, in public-domain fandoms (Sherlock Holmes, Pride and Prejudice, Les Misérables, Dracula, Frankenstein, Little Women, Arthurian Mythology) and Original Work, with their tags; about 600 comments, including replies and guest comments; kudos from members and guests; about 200 bookmarks; 10 series; three collections (a normal one, an anonymous one and an unrevealed one); and a few anonymous works, members-only works, drafts, subscriptions, blocks and mutes. Dates are spread over the last two years. All text is made up from word lists. The data is the same every time.

**Every demo account has the password in `DEMO_PASSWORD`, so anyone who knows it can log in as any of them.** Pick a real password and do not reuse one from anywhere else. None of the accounts has a role: none is an admin, moderator or tag wrangler. To make an admin, register a new account and use `nuclear grant` (see docs/RUNBOOK.md).

### What the platform must provide instead

- **Backups of the database, kept off the database's own machine**, and a restore you have tried. Find out how far back a restore goes (some platforms take one full copy a day, so up to a day of changes can be lost).
- **Somewhere to see the app's logs.**
- **An uptime check** on `/health`, from outside the platform.

Background jobs (emails, exports) run inside the app, so nothing else needs to be set up for them.

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
