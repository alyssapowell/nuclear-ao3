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
  - Status: Works (verified 2026-10-29 against a local database: register, log in, authenticated requests).
  - Welcome emails were **never sent** before the merge: registration posted to a `/api/events` route that didn't exist. They are now sent directly (seen in logs; not yet verified against a real mail server).
- **Sessions and security log**: see and revoke your active sessions, and see recent security events.
  - Intent _(inferred)_: let users notice and stop account takeover themselves.
  - Status: Unverified.
- **OAuth2 / OpenID Connect provider**: third-party apps can "Sign in with Nuclear AO3" and call the API on a user's behalf, with consent screens and app revocation.
  - Intent _(inferred)_: let readers' tools (readers, downloaders, rec bots) use the archive without scraping or sharing passwords.
  - Status: Unverified. This is a large surface area (about 2k lines) for a feature that may not be needed at launch. **Revisit: keep or defer?**
- **Admin user management**: list users and grant or revoke roles.
  - Status: Unverified.

## Works and reading

- **Post, edit and delete works and chapters**, with a rich-text editor.
  - Status: Unverified.
- **Legacy AO3 IDs**: works keep their original AO3 number, and old URLs redirect.
  - Intent: a migration from AO3 must not break the many existing links to works.
  - Status: the database column exists; the importer that would fill it does not (see Migration).
- **Kudos, comments (including guest comments), comment moderation**
  - Status: Unverified. Comment notifications **never worked** before the merge (the call to the notification service had no login token); they are now sent in-process.
- **Bookmarks, series, collections**
  - Status: Unverified.
- **Pseuds, gifting, orphaning, co-authors**: the core AO3 authorship features.
  - Status: Unverified.
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

- **Search works, tags, users, collections, series; advanced and "smart" filtered search; suggestions**
  - Status: Unverified. Runs on Elasticsearch today; planned to move to Postgres full-text search.
- **Search history, saved searches, search alerts**
  - Status: Unverified.
- **Search analytics**: popular terms, zero-result terms, trends, tag-quality insights.
  - Intent _(inferred)_: help wranglers and admins see what readers can't find.
  - Status: Unverified. **Revisit: needed at launch?**
- **Index management**: rebuild, optimize, index or delete individual records.
  - Status: Admin-only (fixed 2026-10-29; it used to be open to anyone).

## Notifications and subscriptions

- **In-app notifications, unread count, live updates over WebSocket**
  - Status: Unverified. Fixed 2026-10-29: the frontend called the wrong port, and auth accepted any token plus an `X-User-ID` header. Known issue: the WebSocket client list isn't protected by a lock, so simultaneous connections can race.
- **Notification preferences and rules; email notifications**
  - Status: Unverified.
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

### Still open

- **Grafana uses admin/admin and `:latest` images**, and is published on the monitoring domain. To fix in the monitoring step.
- **The works service logs SQL details and other debug output** at startup and on every request.
- **The login endpoint has no stricter limit** than general browsing (100 requests a minute per IP), which is loose for password guessing.
- **`X-Test-User-ID` logs anyone in when `GIN_MODE=test`.** It is off by default, but it shouldn't exist in production code.
- If a Hetzner deployment from the old `deploy-*` workflows is live, it is still running the old, vulnerable code until it's redeployed.

## Setup and maintenance problems (found 2026-10-29)

- **Migrations don't apply cleanly to an empty database.** `010`, `012`, `013`, `016`, `019` and `020` all error (duplicate tables, a syntax error, an unbuildable foreign key, a subquery in an index). Docker's Postgres setup stops at the first error, so a brand-new install probably can't create its database at all. There is no migration tool; files only run once, on first container start.
- **Notifications don't work at all on a fresh database.** Migration `013` fails, so the `notification_items` table is never created.
- **Posting works, chapters and comments fails on a fresh database.** The retention trigger from `016` reads `NEW.author_id` or `NEW.user_id`, which those tables don't have.
- **Posting a work doesn't create its first chapter**, even when `chapter_content` is sent.
- **Reading a single chapter always returns 400.** The route is `/chapters/:chapter_id`, but the handler reads `chapter_number`.
- **Block, mute and report crashed on every request** (500) because of a type mismatch; fixed 2026-10-29.
- **121 of 428 frontend unit tests already fail on `main`.** These are pre-existing; the merge didn't add any (compared test by test).
