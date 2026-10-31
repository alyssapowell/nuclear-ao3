# Features

What Nuclear AO3 does, why each piece exists, and what state it's in.

This is a working document. **Intent** lines marked _(inferred)_ were read from the code and still need confirming. **Status** is based on reading the code, not on running it, unless it says otherwise.

Status key:
- **Works**: code is complete and the path is wired end to end
- **Partial**: some of it is real, some is missing
- **Stub**: the endpoint or UI exists but does nothing real
- **Broken**: wired wrong; won't work as deployed
- **Unverified**: looks complete; not yet tested

---

## Accounts and sign-in

- **Register, log in, log out, password reset, email verification**
  - Intent: standard accounts that work without JavaScript tricks or third-party login.
  - Status: Works (verified 2026-10-30 by the smoke test and in a headless browser, against a local database and mail server). Follows AO3:
    - A new account must be activated from its email before it can log in. The link lasts 14 days, and a new one can be requested.
    - Log in with a username or an email.
    - Passwords are 6 to 40 characters.
    - Forgotten passwords are reset by an emailed link that works once and lasts 6 hours. Resetting logs the account out everywhere.
    - Changing a password needs the current one, and logs out every other session.
    - Asking for an activation or reset email never reveals whether an address has an account, and sends at most one email per account every 5 minutes.
  - Before 2026-10-30 almost all of this was placeholder code that reported success without doing anything. Change password said it worked and changed nothing, logout revoked nothing, and no email was ever sent.
- **Sessions**: every login is a session that can be ended. Logging out ends it on the server, so the token stops working at once, not when it expires. A user can see their active sessions (when, where from, which browser) and log any of them out. Logins last 14 days.
  - Intent: let users notice and stop account takeover themselves.
  - Status: Works (verified 2026-10-30). The "security events" log was removed; AO3 has no such page.
- **OAuth2 / OpenID Connect provider**: third-party apps can "Sign in with Nuclear AO3" and call the API on a user's behalf, with consent screens and app revocation.
  - Intent _(inferred)_: let readers' tools (readers, downloaders, rec bots) use the archive without scraping or sharing passwords.
  - Status: Unverified. This is a large surface area (about 2k lines) for a feature that may not be needed at launch. **Revisit: keep or defer?**
- **Admin user management**: list users and grant or revoke roles.
  - Status: **Not built.** The endpoints used to report success without doing anything; they now answer "not implemented". Roles are granted with `make grant`.

## Works and reading

- **Post, edit and delete works and chapters**, with a rich-text editor.
  - Status: posting works (verified 2026-10-29 on a fresh database). A new work is saved as a draft with its first chapter, its tags and its author's pseud in one step. Only its authors can see it until they publish it (`PUT /works/:id` with `"status": "posted"`), which also publishes its chapters. Edit and delete: Unverified.
  - Fixed 2026-10-29: posting never saved the first chapter; tags were sent to a `tag-service` address that no longer exists, so they were never linked; publishing sent a status the database rejects; readers kept seeing the cached draft after publishing.
- **Legacy AO3 IDs**: works keep their original AO3 number, and old URLs redirect.
  - Intent: a migration from AO3 must not break the many existing links to works.
  - Status: the database column exists; the importer that would fill it does not (see Migration).
- **Kudos, comments (including guest comments), comment moderation**
  - Status: kudos, comments and guest comments work (verified 2026-10-29). A work's authors are notified of every comment on it, and a commenter is notified of replies, whether or not they subscribed. Moderation: Unverified.
  - Fixed 2026-10-29: a logged-in user could comment **as anyone else's pseud**; you could comment on works you aren't allowed to see; guest comments always failed; notification listing always returned an empty list; and delivering a notification **crashed the whole server**.
- **Bookmarks, series, collections**
  - Status: Unverified.
- **Pseuds, gifting, orphaning, co-authors**: the core AO3 authorship features.
  - Status: Unverified.
  - Known problem: there are **two pseud tables**. `pseuds` is used for authorship (works, visibility); `user_pseudonyms` is used for comments and profiles. A user's default pseud is created in each on first use. They need merging into one.
- **Authors are shown by pseud**: the public API identifies an author by pseud, not by account ID or username. Readers contact an author through the API (comments, or a relayed message), never by being given an email address.
  - Status: **Broken** (found by reading the code). Work responses include `user_id` and the account username.
- **Anonymous works** (as in AO3's anonymous collections and challenges): the author is hidden until revealed.
  - Status: **Broken** (found by reading the code). The work API still returns the author's user ID and username when `is_anonymous` is true, and the search index stores the author IDs.
- **Reader preferences**: font, size, theme, reading progress, and swipe navigation on mobile.
  - Intent: comfortable long-form reading on any device.
  - Status: frontend only; Unverified.
- **Offline reading (PWA)**, with consent controls.
  - Intent: read downloaded works offline, but only when the *author* has allowed it.
  - Status: Unverified.

## Tags

- **Tag search and autocomplete; fandom, character and relationship browsing; hierarchies**
  - Status: Unverified.
- **Tag wrangling**: work queue, canonical tags, synonyms, parent tags, merges, reports.
  - Intent: the volunteer tag-wrangler workflow that AO3 depends on.
  - Status: Unverified. Until 2026-10-29 the login and role checks here were empty placeholders; they are real now (verified: anonymous gets 401, a normal user 403, an admin 200).
- **Tag prominence**: authors mark which tags are central to a work and which are minor.
  - Intent: cut down "tag spam" so filtering by tag returns works that are actually about it.
  - Status: Unverified.
- **Follow tags**
  - Status: Unverified.

## Search

- **Work search** with AO3's filters and sort orders, plus counts of the most common ratings and tags in the results (`GET /api/v1/search/works`)
  - Status: **Working** (2026-10-30). Runs in Postgres; tested against a real database for drafts, restricted works, blocks, anonymity and each filter (`backend/internal/search/works_test.go`).
- **Search-box suggestions**: works, tags and pseuds that start with what's typed (`GET /api/v1/search/suggestions`)
  - Status: **Working** (2026-10-30).
- **Tag search and people search** (AO3 has both)
  - Status: **Not built.** Tag autocomplete exists (`/api/v1/tags/autocomplete`); full tag and people search pages don't.
- Removed 2026-10-30 because nothing used them and AO3 doesn't have them: search history, saved searches and alerts, search analytics, "smart" and quality-scored search, and index management. See DECISIONS.md.

## Notifications and subscriptions

- **In-app notifications, unread count, live updates over WebSocket**
  - Status: Unverified. Fixed 2026-10-29: the frontend called the wrong port, and auth accepted any token plus an `X-User-ID` header. Known issue: the WebSocket client list isn't protected by a lock, so simultaneous connections can race.
- **Notification preferences and rules; email notifications**
  - Status: preferences and rules unverified. Emails were **never sent**: the delivery framework had no email sender registered. Since 2026-10-30 they're queued as background jobs and sent through SMTP, with retries; the sending and retry path is tested against a local mail server, but no notification email has been checked end to end yet.
  - Known issue: when batching is on (the default, `ENABLE_BATCHING`), waiting notifications are kept in memory and lost if the app restarts.
  - Known issue: comment notification links go to `/works/<id>/comments/<id>`, which the website has no page for.
- **Subscriptions** to works, authors and series.
  - Status: Unverified. There used to be two copies; the notification-service copy was removed and the works copy (the one the frontend uses) stays.

## Export / download

- **Download works as EPUB and other formats**
  - Intent: readers can keep a copy, which is core to AO3's archival mission.
  - Status: **Stub**. `processExport` is a TODO. The API accepts requests but never produces a file. It now requires login, and the requester comes from the token instead of the request body.

## Privacy and safety

- **Block and mute users; report works, comments and tags**
  - Status: Unverified.
- **Privacy settings wizard, GDPR consent, data rights** (migration 016)
  - Status: Unverified.

## Accessibility

- **WCAG 2.1 AA as the goal**: keyboard navigation, screen-reader support, ARIA patterns in search and tag inputs.
  - Intent: accessibility is a requirement, not a nice-to-have.
  - Status: there are axe/Playwright accessibility tests; the compliance claim in the README is not independently verified.

## Internationalization

- **UI translations**: English, Spanish, French.
  - Status: Partial (`frontend/src/lib/i18n.ts`).

## Migration from AO3

- **Importer from otwarchive (MySQL/Rails)**
  - Intent: give OTW a credible path to move over.
  - Status: **Stub**. `tools/ao3-importer/` contains only a README. This is the single most important missing piece for adoption.

## Search on the frontend

- **"Smart suggestions"** (missing-tag recommendations) came only from the GraphQL gateway and were dropped with it. Search itself still works through the REST API. **Revisit: rebuild on Postgres search, or drop?**
- **Crossover detection** calls `/api/v1/tags/universe-families`, which no backend has ever served, so it always falls back.

## Out of scope for this repo (for now)

- **Digital Asset Ownership Framework** (`DIGITAL_ASSET_OWNERSHIP.md`, `docs/ownership/`): a blockchain-agnostic creator-rights protocol. **Revisit: move to its own repo?**

---

## Security findings (found and fixed 2026-10-29)

Found and fixed 2026-10-30:
- ~~Logins couldn't be revoked~~: tokens lasted 30 days and logout did nothing, so a stolen token worked until it expired. **Fixed:** every token belongs to a server-side session that logout, a password change or a reset ends.
- ~~Change password and password reset reported success without doing anything.~~ **Fixed.**
- ~~A request header (`X-Test-User-ID`) logged anyone in as any user when the server ran in test mode~~, and test mode also turned off rate limits and approved OAuth app access without asking the user. One environment setting (`GIN_MODE=test`) made all of that live. **Fixed:** the header is gone, and the server refuses to start in test mode.
- ~~The key that signs every login was generated by the app and stored as plain text in the database~~, so anyone with a database dump or backup could log in as any account, admins included, and nobody held a copy to change it from. **Fixed:** an admin generates the key (`make keys`), saves a copy off the server, and the app loads it from a file. The server refuses to start without one. The old table is dropped (migration 030). Changing the key is a setting change: planned (current logins keep working) or emergency (every login ends). Backups taken before 2026-10-30 still contain the old key and should be destroyed if a leak is suspected.
- ~~Someone could sign up with another person's email address~~, and if that person clicked the activation link, log in to the account with their own password. **Fixed:** activating also asks for the account's password. The real owner takes such an account over with "forgot password".
- ~~Changing or resetting a password left OAuth apps connected~~, along with reset links sent earlier. **Fixed:** both are cancelled.
- ~~A username could be someone else's email address~~, so a login could match the wrong account. **Fixed:** usernames follow AO3's rule (3 to 40 letters, numbers and underscores), so they can't contain `@`, and login looks up an email only when there's an `@`.
- ~~Stored cross-site scripting: HTML users wrote (work summaries and notes, chapters, comments, series, collections, bookmarks, profiles, pseuds) was saved as sent and shown with `dangerouslySetInnerHTML`~~, so `<img src=x onerror=...>` in a summary ran in the browser of everyone who opened the public works list. **Fixed:** the server cleans these fields when they are saved, to AO3's own allowlist of tags, attributes and link protocols (`backend/internal/html`). Migration 031 queues a one-time job that cleans what was saved before. The browser cleans again before showing it (`frontend/src/lib/safeHtml.ts`), and a test fails if any other file uses `dangerouslySetInnerHTML`. Not supported yet, though AO3 allows them: embedded audio and video (from allowed sites), CSS classes in chapter text and notes, and `ftp:` links.

Found while fixing the database setup:

- ~~Every install created an `admin` account with the password `password123`~~, plus three other accounts with that password. **Fixed:** sample accounts live in `backend/migrations/dev/seed.sql`, loaded only by `make seed`. The first admin is made with `nuclear grant <username> admin`.
- ~~The built-in `orphan_account` also had the password `password123`~~, so anyone could log in as it and edit every orphaned work. **Fixed:** it has a password hash that can never match.
- ~~Postgres (with its default password), Redis, mail and monitoring ports were published on every network interface~~, which on a server puts them on the internet (Docker bypasses the `ufw` firewall). **Fixed:** everything but Caddy is bound to `127.0.0.1`.
- ~~Commenting as another user's pseud~~ (see Kudos and comments).
- ~~A reply could name a comment on a different work~~, which attached it to an unrelated thread and notified that comment's author even if they had blocked the replier. **Fixed:** replies must be to a comment on the same work, and nobody is notified about anything done by someone they've blocked, including through subscriptions. The block check is in the shared notification service, so it covers every event.
- ~~Works restricted to logged-in users were readable by anyone logged out.~~ The app stores the setting in `works.restricted`, but the database's visibility check read `restricted_to_users`, which nothing writes. **Fixed** in migration 027 (it checks both); the smoke test covers it.

Found while merging the services:

These were fixed by merging the backend into one program with one login check (`backend/internal/platform`). Each was checked against a running server.

1. ~~The work service trusted a client-supplied `X-User-ID` header~~, and the gateway passed it straight through, so anyone could act as any user. **Fixed:** identity comes only from a verified token; forged headers get 401.
2. ~~Tag and search had no-op login and role checks.~~ **Fixed:** wrangler and admin routes check `user_roles`.
3. ~~**Every** "admin only" check let any logged-in user through~~; even auth's was a TODO. **Fixed:** the same real role check is used everywhere.
4. ~~The notification service accepted any bearer token~~, and its WebSocket took `user_id` from the URL. **Fixed.**
5. ~~The export service had no auth~~ and took the user ID from the request body. **Fixed.**
6. ~~Every service port was published to the host.~~ **Fixed:** one app container, published on localhost only; Caddy is the public entrance. Metrics are on a separate private port.
7. ~~The guest-comment auth exemption matched any POST ending in `/comments`.~~ **Removed:** guest comments go through optional login like the rest of the public routes.
8. ~~CORS reflected any origin, with credentials, when `GIN_MODE` was unset.~~ **Fixed:** there is an explicit allowlist (`CORS_ORIGINS`).
9. ~~The chapter list skipped its visibility check~~ ("TODO: re-enable after debugging"), so anyone could read restricted works and draft chapters. **Fixed:** the check is back, and draft chapters are shown only to the work's authors.
10. ~~Posting a work with no login used a hard-coded "testuser" account.~~ **Fixed:** it returns 401.
11. ~~The rate limiter believed client headers~~ like `X-Client-Admin: true` (up to 500x the normal limit) and the raw `X-Forwarded-For`. **Fixed:** limits are per verified user or per IP, and the IP is trusted only from Caddy.
12. ~~Any token signed by the archive counted as a login~~, including OIDC ID tokens issued to third-party apps. **Fixed:** only tokens with the archive's own audience and issuer are accepted. `golang-jwt` was upgraded to 5.2.2 (CVE-2025-30204).
13. ~~The OAuth authorize step redirected to an unchecked `redirect_uri`~~ on errors, an open redirect. **Fixed:** errors before the redirect URI is verified return JSON.

14. ~~Any logged-in user could delete anyone's notification~~ by ID. **Fixed:** the delete only matches the owner's rows, and returns 404 otherwise.
15. ~~An OAuth consent request wasn't tied to the user who saw it~~, so another user could approve it. **Fixed:** the consent records its user, is checked on approval, and can be used only once.

- ~~The login and sign-up pages followed any `?redirect=` address~~, so a crafted link could send people to another site after they logged in, or run code in their session with `javascript:`. **Fixed 2026-10-30:** only addresses on this site are followed (`safeRedirect` in `frontend/src/utils/auth.ts`).

### Still open

- ~~**Grafana used admin/admin and `:latest` images**, and was published on the monitoring domain.~~ **Fixed 2026-10-30:** Compose refuses to start without `GRAFANA_ADMIN_PASSWORD`, every image is pinned, and Grafana is only on 127.0.0.1 (reached through an SSH tunnel).
- **Most top-level guides describe the old six-service system** (APP_EXPLORER_GUIDE, AUTHENTICATION_GUIDE, DEPLOYMENT_GUIDE, DEPLOYMENT_SETUP, DEVELOPMENT_WORKFLOW, TESTING, VOLUME_SIZING_GUIDE). To be rewritten or removed in the docs pass. SERVICES.md, which described services that no longer exist, was removed 2026-10-30, along with the unused `shared/middleware`, `shared/server`, `shared/database` and `shared/elasticsearch` code.
- **Backups stay on the server.** ops-backup keeps 14 daily copies in `backups/`, but nothing copies them anywhere else, so losing the server loses them. Needs an automated, encrypted copy to storage somewhere else, with its key held by the admin like the signing key.
- **Redis is still required.** The app won't start without it, though it only holds rate limits, OAuth consent prompts and two caches. Making it optional (rate limits in Postgres or in memory) would remove a service from the smallest setup.
- ~~Most services' logs had no size limit~~, which on a small disk eventually stops the database. **Fixed 2026-10-30:** every service keeps at most 5 files of 20 MB.
- **The works service logs SQL details and other debug output** at startup and on every request.
- **The login endpoint has no stricter limit** than general browsing (100 requests a minute per IP), which is loose for password guessing.
- **`X-Test-User-ID` logs anyone in when `GIN_MODE=test`.** It is off by default, but it shouldn't exist in production code.
- If a Hetzner deployment from the old `deploy-*` workflows is live, it is still running the old, vulnerable code until it's redeployed.

## Setup and maintenance problems (found 2026-10-29)

- ~~Migrations didn't apply to an empty database~~, and Postgres's first-start scripts stopped at the first error, so a new install couldn't create its database. **Fixed:** every migration applies to an empty database, and the app applies them itself on startup (`nuclear migrate`), one transaction per file, recording each in `schema_migrations`. CI checks this on every change. A database built by the old method is refused with a pointer to [RUNBOOK.md](RUNBOOK.md).
- ~~The code used seven tables that no migration created~~ (notifications, subscriptions, notification and profile preferences, collection membership, hit history). **Fixed** in migration 022. A test now fails if the code queries a table that doesn't exist.
- ~~Database triggers broke posting~~: a retention trigger read columns that don't exist, the chapter counter rejected every new draft, the tag summary joined on a missing column, and the user-stats trigger failed for guests. **Fixed** in migrations 016 and 023–026. The retention trigger was removed rather than fixed: it would have expired works after seven years, which AO3 doesn't do.
- **Reading a single chapter always returns 400.** The route is `/chapters/:chapter_id`, but the handler reads `chapter_number`.
- **Block, mute and report crashed on every request** (500) because of a type mismatch; fixed 2026-10-29.
- **118 of 422 frontend unit tests fail.** They all failed before the Vite move too (compared test by test). Two suites that tested components that don't exist (`LoginForm`) were deleted, and two Next.js config checks were replaced by a check that every page has a route.
- ~~The dashboard, subscriptions page, work page and subscribe/gift buttons never knew who was logged in~~: they used a stub `useAuth` that always returned nobody, so the dashboard showed "Loading" forever. **Fixed 2026-10-30:** they use the real one (`src/utils/auth.ts`). It now returns the same objects between renders, because the subscriptions page re-fetched in a loop until rate-limited.
- ~~Search never returned anything~~: works were never sent to Elasticsearch (the code that did it had no callers), and the website's search client crashed on its own address handling. A search with no matches also showed every work instead. **Fixed 2026-10-30** by the move to Postgres search.
- **The works and auth test suites use the developer's own Postgres** (`localhost:5432/ao3_nuclear`) instead of a fresh database, so results depend on that machine. They should use `internal/testdb`, which builds a migrated database per test, as the search tests do.
- **Work gifts need a login to read** (`GET /works/:work_id/gifts` is behind `RequireUser`). On AO3, gifts show to everyone.
- **API endpoints the website calls that fail** (seen 2026-10-30 with a fresh database): `GET /my/bookmarks` returns 500 ("Failed to count bookmarks"); the profile and pseud endpoints return 404; work gifts return 401 for logged-out readers, so the work page shows "Error loading gifts".
