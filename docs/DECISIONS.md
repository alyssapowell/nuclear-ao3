# Decisions

Why the project is shaped the way it is. Newest first. Each entry says what was decided, why, and what it costs.

## The guiding constraint

AO3 is run by volunteers. Its outages over the last two years have come from upgrades, moving parts between systems, and recovery that needed outside vendors. Every choice here should make the system easier for a **beginner volunteer** to run, upgrade, restore and extend. When two options are close, pick the one with fewer moving parts.

---

## 2026-10-29: Anyone can create new tags when posting, including fandoms

**Decision:** When a work is posted with a tag that doesn't exist yet, of any type including fandom, the tag is created unwrangled. Wranglers decide later whether it becomes canonical or a synonym.

**Why:** That's how AO3 works. The old code limited new fandoms to admins, but AO3 has no such rule, and it would stop people from posting in new or small fandoms. The general rule here: match what AO3 does, and don't add restrictions it doesn't have.

**Cost:** Wranglers get typo fandoms to clean up, just as they do on AO3.

## 2026-10-29: The app builds and upgrades its own database

**Decision:** The schema lives in numbered SQL files in `backend/migrations/`, built into the app. On startup (or with `nuclear migrate`) the app applies the ones the database hasn't had, each in its own transaction, and records them in `schema_migrations`. A Postgres lock means only one copy migrates at a time. Sample data is a separate file that is never loaded automatically. There are no built-in accounts; `nuclear grant` makes the first admin.

**Why:** Before, Postgres ran the files once, on the very first start, and stopped at the first error. Six of them failed, so new installs couldn't set up a database at all, and nothing recorded which files had run. Upgrades had no path. A volunteer upgrading should only have to start the new version.

**Cost:** Migrations must never be edited once released, only added. Existing files were edited in place this once, because none of them had ever applied cleanly anywhere. A database built the old way can't be upgraded automatically (see [RUNBOOK.md](RUNBOOK.md)).

## 2026-10-30: Accounts work the way AO3's do, and logins can be revoked

**Decision:** Build real account security, following AO3: activation by email before the first login, log in by username or email, 6-to-40-character passwords, and password reset by an emailed link. Every login is a session stored in the database that can be ended.

**Why:** Almost all of it was placeholder code that reported success. A stolen login token couldn't be cancelled, and someone who changed their password after a compromise was told they were safe when nothing had changed.

**How:**
- **Tokens:** a login token is still a signed JWT, but it names its session (`sid`), and every request checks that the session is still active. That's one indexed query per request, with no Redis needed. Logging out, changing a password (other sessions only) and resetting a password (all sessions) end sessions.
- **Lifetime:** logins last 14 days, with no refresh tokens, which keeps the frontend and the API simple.
- **Activation and reset links:** they carry a random secret, and only its SHA-256 hash is stored, so a database dump can't be used to take over accounts through them.
- **Mail:** activation and reset emails are queued in the same transaction as the account change, so they can't be lost.

**Cost:** existing unactivated accounts must activate (or reset their password) before they can log in again. The smoke test now needs a mail catcher with MailHog's API, which the compose setup includes.

## 2026-10-30: The server refuses to run in an insecure setup

**Decision:** Insecure settings stop the server at startup with a plain message, instead of relying on someone to read a checklist. The first check: `GIN_MODE=test`, which turned off rate limits and auto-approved OAuth consent.

**Why:** A checklist only works if someone reads it, every time, through every change of who runs the site. A check the server does itself at every start never gets skipped.

**Next:** default or weak passwords in `.env`, Grafana's admin/admin, email links pointing at localhost in production, and a signing key that isn't the one the admin saved.

## 2026-10-29: Monitoring ships ready to use

**Decision:** Keep Prometheus, Grafana and Loki, but store their data sources, dashboards and alerts as files in the repo so they load automatically. There will be one "Is the archive OK?" dashboard, and every alert links to a runbook section that says what to do. Images are pinned, the admin password is generated, and the monitoring ports are bound to localhost.

**Why:** Volunteers without monitoring find out about problems from users. Volunteers with monitoring they have to build themselves don't get it built.

**Cost:** Three more containers than a bare setup. They're optional for local development.

## 2026-10-29: Postgres for search and background jobs

**Decision:** Replace Elasticsearch with Postgres full-text search, and run background jobs (email, notifications, exports) from a jobs table in Postgres.

**Why:** A search cluster is the hardest part of the stack to keep healthy through upgrades. A jobs table means a stuck backlog can be seen and retried from an admin page instead of being invisible.

**Cost:** Postgres search is less flexible for relevance tuning. This gets confirmed after auditing which Elasticsearch features the search code really uses.

**How search was done (2026-10-30):** The audit found that nothing ever reached Elasticsearch: the code that sent works to it had no callers, so search was empty. The website used only two endpoints, work search and suggestions.
- `backend/internal/search` is now about 500 lines instead of about 5,300. It queries the works table directly. A `search_vector` column, which Postgres keeps current by itself (migration 028), weights matches in the title first, then tags, then the summary. Chapter text isn't searched; AO3 doesn't do that either.
- Visibility goes through the same `can_user_view_work` function as the work page, so search can't show a draft, a work restricted to logged-in users, or a work by someone who blocked the reader. Searching by creator never matches anonymous works.
- Results name creators by pseud only (id and name), never by account or username.
- Filters follow AO3: words, title, creator, the four tag types, excluded tags, rating, warnings, categories, language, word count, kudos and comment minimums, complete or in progress, crossovers, single chapter, and updated within. The website's "Relationship focus", "Relationship scope", "Hide gen fic", "Exclude poorly tagged" and "Quality score" options were dropped. AO3 has none of them and the server never applied them.
- Removed endpoints: tag, user, collection and series search; "advanced" and "smart" search; facets; quality analysis; popular and trending terms; search history; saved searches and alerts; search analytics; index management. Nothing on the website used them, and AO3 has none of them except tag and people search, which can come back as plain queries when the website needs them.
- Filter counts are exact, computed over every matching work. Paging stops at result 10,000. At AO3's size (millions of works), broad searches will need measuring: if counting gets slow, count over the top few thousand results instead.

**How background jobs were done (2026-10-30):** Jobs live in a `jobs` table (migration 029), and the app runs them itself (`backend/internal/jobs`, about 250 lines). There's no separate worker program or queue server.
- A job is added in the same database transaction as the change that caused it. A new account and its welcome email are saved together: if the account fails to save, no email exists, and once it's saved the email can't be lost.
- A failed job is retried after 30 seconds, then 1, 2, 4 minutes and so on, for 10 tries (about four hours). Then it's marked failed and kept, with its error, until someone retries it.
- Any number of copies of the app can run jobs at once without doing one twice (`FOR UPDATE SKIP LOCKED`). A job whose worker died is picked up again after 10 minutes, so a job must be safe to run twice.
- `nuclear jobs` shows what's waiting, running and failed, and why. `nuclear jobs retry [id]` sends failed jobs again.
- Email is the first kind of job. `shared/email` sends it: a 30-second limit per email, TLS whenever the mail server offers it, a password only over TLS, and headers built so nothing in a title or username can add headers. Notification emails are plain text, because they quote titles and names anyone can write.
- Removed: `shared/messaging`, about 4,900 lines for channels, templates, telemetry and rate limits. Nothing was registered with it, so it delivered nothing, and no notification email was ever sent.
- Still to move onto the queue: exports (still a stub) and notification batching, which keeps waiting notifications in memory and loses them on restart.

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

**Decision:** Build the frontend as static files with Vite, React and React Router, served by Caddy. There is no Node server in production.

**Why:**
- The app didn't use Next.js's server features. It had no API routes or server actions, most components were client-side, and the only Next-specific code was a cookie-redirect middleware.
- Next.js's 2025 server-side vulnerabilities (including a middleware auth bypass that matches this repo's `middleware.ts`) make it a risk with no benefit here.
- Contributors only need to know plain React.

**Cost:** No server-side rendering. Search engines and no-JavaScript readers get less. Revisit if public pages need it.

**How it was done (2026-10-30):**
- Every page lives in `frontend/src/pages/`, and `frontend/src/App.tsx` lists every page and its address. Logged-in-only pages are grouped there behind one `AuthGuard`, which replaces both Next's middleware and the per-page guards.
- Caddy serves the built files, so the `frontend` container is gone. `frontend/Dockerfile` builds the files into the Caddy image. Fingerprinted files under `/assets/` are cached forever; everything else (`index.html` above all) is checked every time, so an upgrade reaches everyone at once.
- Pages load on first visit (`React.lazy`), so the first visit downloads only what it needs.
- Tests run on Vitest instead of Jest: same API, no separate build setup. Nine Playwright configs became one.
- ESLint was dropped. Most of its rules were switched off, CI ignored its result, and it needed Next's config. `npm run build` now runs the TypeScript check, and the build fails on a type error. Before, Next skipped it.
- Dependencies in `package.json` are pinned to exact versions.

## 2026-10-29: Repo cleanup

**Decision:** Remove committed binaries, generated data, test reports, debug pages and one-off scripts.

**Why:** These made the repo large and confusing, and some of them (like `/debug-auth`) didn't belong in production.
