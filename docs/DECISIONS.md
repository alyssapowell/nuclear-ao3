# Decisions

Why the project is shaped the way it is. Newest first. Each entry says what was decided, why, and what it costs.

## The guiding constraint

AO3 is run by volunteers. Its outages over the last two years have come from upgrades, moving parts between systems, and recovery that needed outside vendors. Every choice here should make the system easier for a **beginner volunteer** to run, upgrade, restore and extend. When two options are close, pick the one with fewer moving parts.

---

## 2026-10-30: Moderation state follows AO3; admin actions are append-only audited

**Decision:** Hidden, spam, unreviewed, frozen, banned and suspended state use AO3's meanings and are decided in one place in the database (`can_user_view_work` and its siblings for chapters, series, bookmarks and comments, migration 033), which every listing, search and page calls. Every admin action, from the site or from `nuclear`, writes a row to `admin_actions` (migration 034) in the same transaction as the change. The table refuses UPDATE, DELETE and TRUNCATE.

**Why:** The people running the archive have to be able to answer "who hid that, and why" long after the fact, and to show that no one quietly edited the answer. Putting the rule in the database means a new page can't forget it; putting the audit row in the transaction means there is never a change without a record. Only staff can see hidden work, only admins can act on staff, and nobody can act on themselves, so one stolen moderator login can't promote itself or clear its own ban.

**What it costs:** A superuser or the table owner can still drop the triggers, so backups are the real record. Visibility functions run per row, which is fine at this size but is something to measure at AO3's. Sanction checks use the admin and moderator roles; AO3's finer roles (policy and abuse, support) are not modelled.

---

## 2026-10-30: Redis removed

**Decision:** The stack is Postgres, the app and Caddy (plus monitoring). Redis is gone.

**Why:** Nothing used its speed. It held rate-limit counters, ten-minute OAuth prompts and two small caches in front of queries Postgres answers in milliseconds. It was still a service to run, back up, size and keep healthy, and the app refused to start without it.

**What replaced each use:**
- **Rate limits:** counters in the app's memory (`internal/auth/ratelimit.go`). Fixed windows per client, the same limits and `X-RateLimit-*` headers as before. Idle keys are dropped every minute, and at most 100,000 keys are tracked; when that is full the least recently used are dropped, and requests are never refused for it.
- **OAuth sign-in and consent state:** the `oauth_pending` table (migration 032), with an expiry time of ten minutes. Writes are upserts, expired rows are ignored when read, a consent is used up with `DELETE ... RETURNING` so it can only be answered once, and the jobs cleanup removes old rows. The `session:<id>` cookie lookup was removed because nothing ever wrote that key.
- **Tag and work caches:** removed. Handlers query Postgres directly, and so the invalidation code is gone too.
- **Work view counts:** counted in memory and added to `work_statistics.hits` in one batched statement every 30 seconds and on shutdown. The old per-view write named a column (`collections`) that does not exist, so it failed and no views were recorded; the new write works.

**Cost:**
- **Each copy of the app enforces rate limits on its own.** With N copies behind Caddy, a client can make up to N times the limit. A restart gives every client a fresh window. For one server this changes nothing. If copies are added, lower the limits to match.
- A crash loses up to 30 seconds of view counts.
- Pages are never served from a cache, so every read goes to Postgres.

## 2026-10-30: User HTML follows AO3's allowlist, is cleaned when saved, and cleaned again in the browser

**Decision:** The tags, attributes and link protocols users may use in summaries, notes, chapters, comments and profiles are AO3's (`config/initializers/gem-plugin_config/sanitizer_config.rb`): no scripts, event handlers, `style`, `iframe`, `object`, `embed` or `form`, and links get `rel="nofollow noopener"`. One Go policy, `internal/html`, cleans every such field when it is created or updated, so the database only ever holds clean HTML. Rows saved before that are cleaned by a background job queued by migration 031. The job rewrites a row only if cleaning changes it, and does not touch `updated_at` or send notifications: the triggers that do those things skip work while the job's transaction has `nuclear.maintenance` set. The browser cleans again with DOMPurify, configured with the same list, in `frontend/src/lib/safeHtml.ts`; every place that shows HTML goes through it, and a test fails if any other file uses `dangerouslySetInnerHTML`.

**Why:** A summary with `<img onerror=...>` ran for every visitor to the works list. Cleaning on write fixes the data once, instead of relying on every page that shows it. Using AO3's list, and not our own, keeps the same stories rendering the same way and avoids inventing rules. The second layer covers old rows until the job has run, text built in the browser, and mistakes in the first layer. Titles, tag names, pseud names and usernames are plain text and are not cleaned; React escapes them.

**Cost:** Two copies of the allowlist (Go and TypeScript) must be kept in step. Text is cleaned once when saved, so a future change to the list needs another cleanup job. Embeds, CSS classes and `ftp:` links, which AO3 allows in some fields, are not supported.

## 2026-10-30: The login signing key is a file the admin holds

**Decision:** One RSA key signs logins. The admin creates it with `nuclear keys generate`, keeps a copy outside the server, and points the app at it with `SIGNING_KEY_FILE`. It is never stored in the database. To change it, set `SIGNING_KEY_FILE` to a new file; for a planned change, list the old file in `SIGNING_KEY_PREVIOUS_FILES` for 14 days (the longest a login lasts) so people stay logged in. Each key's ID is its RFC 7638 thumbprint, tokens carry it, and a token with an unknown ID is refused. The app won't start without a valid key (RSA, at least 2048 bits) and never makes one itself.

**Why:** The app used to create the key on first start and keep the private half as plain text in the database. Anyone who got a database backup or a read-only SQL injection could forge a login for any account, including an admin's, and nothing would show it. Backups get copied to more places than anything else, so the key must not be in them. A key in a file the admin controls can be changed with a config edit and a restart, with no code or database step.

**Cost:** The admin has a new thing to look after: if every copy of the key is lost, everyone is logged out once and a new key is needed. Migration 030 drops the old table, so on upgrade everyone logs in again once. Old backups still contain the old key.

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

**Also checked at startup (2026-10-30):** unless `ENVIRONMENT=development`, the server refuses to start with an example or short database password, a `SITE_URL` that isn't public `https://`, or mail going to the development mail catcher. An unset `ENVIRONMENT` counts as production. Every message names the setting and how to fix it. Separately, the server won't start without the admin's signing key, and Compose won't start Grafana without a password.

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
