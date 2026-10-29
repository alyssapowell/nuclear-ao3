# Decisions

Why the project is shaped the way it is. Newest first. Each entry says what was decided, why, and what it costs.

## The guiding constraint

AO3 is run by volunteers. Its outages over the last two years have come from upgrades, moving parts between systems, and recovery that needed outside vendors. Every choice here should make the system easier for a **beginner volunteer** to run, upgrade, restore and extend. When two options are close, pick the one with fewer moving parts.

---

## 2026-10-29: Monitoring ships ready to use

**Decision:** Keep Prometheus, Grafana and Loki, but store their data sources, dashboards and alerts as files in the repo so they load automatically. There will be one "Is the archive OK?" dashboard, and every alert links to a runbook section that says what to do. Images are pinned, the admin password is generated, and the monitoring ports are bound to localhost.

**Why:** Volunteers without monitoring find out about problems from users. Volunteers with monitoring they have to build themselves don't get it built.

**Cost:** Three more containers than a bare setup. They're optional for local development.

## 2026-10-29: Postgres for search and background jobs

**Decision:** Replace Elasticsearch with Postgres full-text search, and run background jobs (email, notifications, exports) from a jobs table in Postgres.

**Why:** A search cluster is the hardest part of the stack to keep healthy through upgrades. A jobs table means a stuck backlog can be seen and retried from an admin page instead of being invisible.

**Cost:** Postgres search is less flexible for relevance tuning. This gets confirmed after auditing which Elasticsearch features the search code really uses.

## 2026-10-29: One Go binary instead of six services and a gateway

**Decision:** Merge auth, works, tags, search, notifications and export into one Go program with one package per area. Remove the API gateway and GraphQL.

**Why:**
- All six services already share one database, so splitting them added network hops, seven deploys and seven copies of middleware without any isolation.
- Login checking was reimplemented per service, and in most of them it doesn't actually check anything (see the security findings in [FEATURES.md](FEATURES.md)).
- One binary means one auth check, one log stream and one thing to restart.

**What changed for API callers:**
- Everything is on one origin under `/api/v1`, plus `/ws` and `/health`.
- `GET /api/v1/admin/statistics` (tags) is now `/api/v1/admin/tag-statistics`. The works one keeps the old path; the two used to collide.
- Notification `/preferences` and `/rules` are now `/notification-preferences` and `/notification-rules`. The test and process-event endpoints moved under `/api/v1/admin/notifications/`.
- `GET /api/v1/exports/user/:user_id` is now `GET /api/v1/my/exports`.
- GraphQL (`/graphql`) is gone; the three frontend screens that used it (login, register, search) use REST.

**Cost:** Areas can't be scaled independently. At AO3's size that's fine: run more copies of the one binary behind Caddy.

## 2026-10-29: Vite + React instead of Next.js

**Decision:** Build the frontend as static files with Vite, React and React Router, served by the Go binary or Caddy. There is no Node server in production.

**Why:**
- The app didn't use Next.js's server features. It had no API routes or server actions, most components were client-side, and the only Next-specific code was a cookie-redirect middleware.
- Next.js's 2025 server-side vulnerabilities (including a middleware auth bypass that matches this repo's `middleware.ts`) make it a risk with no benefit here.
- Contributors only need to know plain React.

**Cost:** No server-side rendering. Search engines and no-JavaScript readers get less. Revisit if public pages need it.

## 2026-10-29: Repo cleanup

**Decision:** Remove committed binaries, generated data, test reports, debug pages and one-off scripts.

**Why:** These made the repo large and confusing, and some of them (like `/debug-auth`) didn't belong in production.
