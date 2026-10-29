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
  - Status: Works (auth service).
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
  - Status: Unverified. Note that the auth exemption for guest comments matches *any* POST whose path ends in `/comments` (see Security).
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
  - Status: **Broken (security)**. Login and role checks in the tag service are empty placeholders, so anyone can wrangle, ban tags or add wranglers.
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
  - Status: **Broken (security)**. There is no real auth check, so anyone can rebuild or delete the index.

## Notifications and subscriptions

- **In-app notifications, unread count, live updates over WebSocket**
  - Status: **Broken**. The frontend calls port 8004, but the service runs on 8087. Its auth also accepts any bearer token plus an `X-User-ID` header, so anyone can read anyone's notifications.
- **Notification preferences and rules; email notifications**
  - Status: Unverified.
- **Subscriptions** to works, authors and series.
  - Status: implemented **twice**, once in the work service and once in the notification service. One copy needs to go.

## Export / download

- **Download works as EPUB and other formats**
  - Intent: readers can keep a copy, which is core to AO3's archival mission.
  - Status: **Stub**. `processExport` is a TODO. The API accepts requests but never produces a file, and it has no auth.

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

## Out of scope for this repo (for now)

- **Digital Asset Ownership Framework** (`DIGITAL_ASSET_OWNERSHIP.md`, `docs/ownership/`): a blockchain-agnostic creator-rights protocol. **Revisit: move to its own repo?**

---

## Security findings (found 2026-10-29)

These are about the current code. The single-binary merge (see [DECISIONS.md](DECISIONS.md)) replaces all of them with one real auth check.

1. **The work service trusts a client-supplied `X-User-ID` header**, and the gateway passes a client's `X-User-ID` straight through (`api-gateway/proxy.go`). Anyone can act as any user.
2. **Tag and search services have no-op `JWTAuthMiddleware` and `RequireRoleMiddleware`.** All of their "protected" and admin routes are public.
3. **The notification service accepts any bearer token** plus an `X-User-ID` header.
4. **The export service has no auth.**
5. **Every service port is published to the host** in `docker-compose.yml`, so the gateway can be bypassed entirely.
6. **The work service logs `DATABASE_URL`, including the password**, at startup.
7. **The guest-comment auth exemption** matches any POST path ending in `/comments`.
8. **Grafana uses admin/admin and `:latest` images**; Prometheus is published on 9090.

If a Hetzner deployment from the `deploy-*` workflows is live, it is exposed to all of the above.
