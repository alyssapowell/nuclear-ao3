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
  - Status: **Kept (decided 2026-11-02) and published (2026-11-02, audit phase 5).** It's one of the community's biggest asks and something AO3 can't offer today. What an app's token may reach, per route and scope, with the routes apps can never reach and the rate limits, is in [docs/API_SCOPES.md](API_SCOPES.md).
  - **Published (2026-11-02, audit phase 5):** Caddy sends exactly `/.well-known/openid-configuration`, `/.well-known/oauth-authorization-server`, `/auth/authorize`, `/auth/token`, `/auth/userinfo`, `/auth/introspect`, `/auth/revoke`, `/auth/jwks` and `/auth/consent/*` to the app, by exact path, never `/auth/*` (the website's `/auth/login` and other pages stay the website's). `/auth/register-client` is admin-only and stays unpublished; admins use the website. The smoke test checks `/auth/jwks` and the discovery document through Caddy.
  - **Hardened (2026-11-02, audit phases 0 and 1):**
    - Removed: the client-credentials grant, and PKCE `plain`. Every app, public or confidential, must use PKCE with S256 (a 43-character challenge; verifiers of 43 to 128 characters).
    - Only `authorization_code` and `refresh_token` grants and the `code` response type are accepted, at registration, admin update and the token endpoint. A refresh token is issued only to an app allowed the `refresh_token` grant.
    - Admin-only scopes (`admin`, `tags:wrangle`) can never be granted: they are refused at registration, admin update, authorize, token and refresh. One list (`models.AO3OAuthScopes`) is the source; a test checks the `oauth_scopes` table agrees.
    - Lifetimes are clamped in code: access tokens 5 minutes to 1 hour, refresh tokens up to 30 days with a 90-day cap on a whole chain, authorization codes 60 seconds.
    - Both discovery documents list only what exists (code flow, query mode, RS256, S256, real non-admin scopes, no registration address, the `ao3_*` claims not advertised; phase 2 below made the subject types `pairwise`). The issuer is the archive's `SITE_URL`; discovery and ID tokens are refused if it is unset or an example address.
    - Tokens and codes are stored only as SHA-256 hashes. A code is used up atomically and bound to its redirect address; using a code twice revokes what the first use made. Refresh tokens rotate atomically in families: reusing an old one revokes the whole family, and a refresh also needs a live approval, an active app and an account in good standing.
    - Revoking an approval, removing an app, switching an app off or deleting it, and resetting its secret revoke the access and refresh tokens and unexchanged codes. Access tokens stop working when the app is off or the person is banned, suspended or inactive.
    - Client authentication: HTTP Basic first (URL-decoded), form body otherwise, never the query string (a secret in the address is refused). Unknown apps cost the same time as known ones. Public apps skip the secret only at the token endpoint (and, from phase 2, when revoking). Revocation is limited to the calling app's own tokens (and takes the linked token with it); introspection is for confidential apps and answers `active: false` for another app's token.
    - Rate limits on authorize, token, introspect and revoke (per address, and per app id); size caps on `state`, `nonce`, `login_hint` and `redirect_uri`; nothing is stored for an unknown app. Redirect addresses are https only (loopback http in development; reverse-DNS schemes for public apps), with no fragments or credentials, and error redirects are built safely.
    - Admin app API: bounded paging, admins can see switched-off apps, updates are validated, and every create, update, trust change, secret reset, delete and token revoke is written to the admin action log.
  - **Identity and privacy (2026-11-02, audit phase 2):**
    - No linking across apps: every app gets its own random, stable `sub` for a person (`oauth_subjects`, migration 046), in the ID token, userinfo and introspection. It is deleted with the app. Discovery says `subject_types_supported: ["pairwise"]`. `auth_time` is the time of the sign-in with that app, not the person's last login (which would be the same in every app).
    - Claims follow the pseud rules: under `openid` only `sub` (plus iss, aud, azp, exp, iat, auth_time, nonce when sent, at_hash). Under `profile`, `pseud: {id, name}` and `name`: the one pseud the person chose. Under `email`, `email` and `email_verified`, and only if the approval explicitly included email. The account id, username, other pseuds, roles, counts, join and update dates and the profile link are gone; `claims_supported` lists exactly what remains.
    - Choosing a pseud: the consent request takes `pseud_id` (default: the default pseud; someone else's pseud is refused with 403). `prompt=consent` shows the screen again so the choice can be changed. If the chosen pseud is deleted the app gets no pseud (never another one) and has to ask again.
    - Trusted apps skip the consent screen only for `openid` alone; `profile`, `email` and everything else need an explicit yes.
    - The consent screen's JSON (`GET /auth/consent/:id`) now needs the signed-in person it belongs to (404 for anyone else, 401 with no login), is shaped for the future screen (app name, website, logo, scopes with descriptions, the person's own pseuds as `{id, name}`), and never holds account ids or usernames. The `c.HTML` call that had no templates is gone, and the pending request keeps only ids (it used to keep the app's record, secret hash included).
    - ID tokens carry `kid`, `azp` and `at_hash`, leave out `nonce` when none was sent, and are signed with their own key (`OIDC_SIGNING_KEY_FILE`, made with `make keys-oidc`; falls back to the login key with a logged warning). Only that key is published at `/auth/jwks`, so the login key is never shared.
    - Public apps can revoke their own tokens with just their `client_id` (RFC 7009); introspection is still confidential-only.
  - **App vetting (2026-11-02, audit phase 3):**
    - Every app has a status: `pending`, `approved`, `rejected` or `suspended` (`oauth_clients.status`, with `reviewed_by`, `reviewed_at` and `review_note`; migration 047). Only an approved app can authorize, exchange a code, refresh, or have its tokens accepted. `is_active` is kept equal to "approved" by a database trigger, so nothing can switch an app on except an approval. Apps that existed before are approved (those already switched off are suspended).
    - Anyone signed in and in good standing registers an app with `POST /api/v1/apps`. It is created `pending`, owned by them, with the code grant (and optionally refresh), response type `code`, no admin scopes, clamped lifetimes and the usual redirect rules; PKCE is required as for every app. Public apps are allowed. `is_trusted`, `is_first_party`, `status` and any other field are refused. Limits: 10 apps and 5 waiting per person. The secret of a confidential app is shown once.
    - Owners: `GET /api/v1/apps` (their own apps only, never the secret, with the status and the reviewer's note), `PATCH /api/v1/apps/:id` (name, description, website, logo, redirect addresses, scopes), `POST /api/v1/apps/:id/reset-secret`, `DELETE /api/v1/apps/:id` (revokes everything first). Changing the redirect addresses or scopes of an approved app sends it back to `pending` and ends every token, code and consent it holds (see docs/DECISIONS.md). Owners cannot change the status.
    - Admins: `GET /api/v1/admin/apps?status=pending` is the queue (default `pending`, `all` for every app); the owner appears as their default pseud `{id, name}` only. `POST /api/v1/admin/apps/:id/approve`, `/reject` and `/suspend` (note required for the last two) and `/reinstate`. Reject and suspend revoke tokens, codes and consents. Every action is an audit row; the owner is emailed about an approval, rejection or suspension (not about their own action). `/api/v1/auth/admin/oauth/clients` still works and shows the status; `is_active: false` there means suspend, and it cannot approve a waiting app. An app made by an admin (`/auth/register-client`) is approved at once.
    - `nuclear apps [--status pending]`, `nuclear apps approve <id>`, `nuclear apps reject <id> --note "..."` and `nuclear apps suspend <id> --note "..."` do the same from the server, audited as `cli:<user>`. See docs/INCIDENTS.md.
  - **Browser sign-in flow and settings pages (2026-11-02, audit phase 4):**
    - The login is a Bearer token in localStorage, which a browser redirect can't carry, so the flow goes through the website. The app sends the person to `https://SITE/oauth/authorize?<the standard query>`. That page sends a logged-out person to `/auth/login?redirect=/oauth/authorize?...`, then calls `GET /api/v1/oauth/authorize?<same query>` with the Bearer header. The answer is always JSON: `{redirect}` (the app's redirect_uri with a code, or with an error the spec says to report to the app), `{consent: {...}}`, or `{error, error_description}` (400; unknown app or bad redirect_uri, shown on the page and never sent to the app). The page shows the consent screen (app name, website, logo, scopes, pseud picker only for `profile`, an unticked email box only for `email`, Allow and Deny) and posts to `POST /api/v1/oauth/consent/:id` with `{approved, pseud_id, share_email}`; the answer is `{redirect}` and the page does `window.location.assign`. If the person leaves the email box unticked the app is allowed without the `email` scope (and if email was all it asked for, it hears `access_denied`). `/auth/authorize` and `/auth/consent/:id` still work for compatibility; logged out, `/auth/authorize` now redirects to `/oauth/authorize?<same query>` and stores nothing. The unused login-resume code was removed.
    - Connected apps: `GET` and `DELETE /api/v1/oauth/authorized-applications[/:client_id]` (the `/auth/...` paths still work). The list now holds every app the person has an approval with or a live access or refresh token for, with scopes, the shared pseud `{id, name}`, whether email is shared, when granted and last used. Disconnecting uses the full revoke path (tokens, refresh tokens, codes, approval).
    - Decisions apply to the version the admin looked at: `oauth_clients.revision` (migration 048) goes up on every change to an app (a trigger counts it). The queue returns it; approve, reject, suspend, reinstate and the two display decisions require it in the body, and answer 409 `app_changed` if it differs. `nuclear apps ...` skips the check (it shows the current settings and acts at once). Editing an app re-reads and locks its row first, and any change to redirect addresses or scopes sets it to pending (suspended apps can't be edited) and revokes everything.
    - Display changes on an approved app (name, website, logo, description) are kept as a proposal (`proposed_*`, `display_review_pending`). People see the approved details until an admin approves; tokens and approvals are untouched. `POST /api/v1/admin/apps/:id/approve-display` and `/reject-display` (note required for reject), audited and emailed; `?status=display` lists them; `nuclear apps approve-display|reject-display`. Any owner edit drops the app's trusted mark and writes an `oauth_client.owner_update` audit row.
    - Pages: `/oauth/authorize`, `/profile/apps` (Connected apps), `/profile/my-apps` (Your apps: register, status and review note, edit, one-time secret on register and reset, delete), `/admin/apps` (App review: waiting for review, display changes side by side, approve, reject, suspend, reinstate; non-admins see "for the archive's admins"; no menu link yet). Login no longer writes the token to a cookie; logout still clears a leftover one.
  - **Resource side (2026-11-02, audit phase 5):** an app's opaque access token is resolved (`ResolveAppToken`: unexpired, unrevoked, app approved, person active and in good standing, not staff) by a gate that runs on every request. Default deny: an app token reaches only routes listed in a route-to-scope table (52 routes across `read`, `comments:write`, `bookmarks:manage` and the new `subscriptions:manage`); account, sign-in, admin, app-management, block/mute, pseud-management, `/my`, export and notification routes are on a hard denylist. Missing scope gives `403 insufficient_scope` with the RFC 6750 header. App writes go under the one pseud shared with the app. Per-(app, person) limit of 120 requests a minute (`APP_RATE_LIMIT_PER_MINUTE`); `last_used` is kept up to date. **Reader mode (2026-11-02, migration 050):** apps always get the reader's view of the archive and never the person's creator view. The visibility functions take an `as_reader` flag (members-only works show, blocks and mutes apply, every creator, owner and staff exception is off), every route in the allowlist gets its viewer from one helper (`platform.ReaderViewer`), and the pseud pages use the same helper instead of their own stranger rule. This closes the old "known limit" (a draft or hidden work answering 200 to the app of its creator). A guard test compares every open read route for an app and for an ordinary reader. Details, the denylist and the `/my` decision are in [docs/API_SCOPES.md](API_SCOPES.md).
  - **Still to do:** phase 6, the audit log table, cleanup of expired tokens and codes in the jobs queue, `at_hash` and `azp`; opening work, chapter, series and collection writes to apps (needs the "as a reader, not an author" flag in the visibility functions, see API_SCOPES.md "Known limits").
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
  - Status: pseuds verified 2026-10-30 (tests and smoke test); gifting, orphaning and co-authors: Unverified.
  - Fixed 2026-10-30: there was a second pseud table (`user_pseudonyms`, used for comments). It is merged into `pseuds` (migration 037). Pseuds follow AO3: names of 1 to 40 characters with letters, numbers, spaces, underscores and dashes in any script, unique per user ignoring case; description up to 500 characters; icon address, alt text (250) and comment text (50); exactly one default pseud per user. Every account gets its default pseud, named after the username, when it is created.
  - `POST /pseuds`, `GET /my/pseuds`, `PUT /pseuds/:id` and `DELETE /pseuds/:id` manage a user's own pseuds. Deleting a pseud moves its works, comments and gifts to the default pseud; the default pseud and the one named after the username can't be deleted (AO3 `pseuds_controller.rb`).
  - Not built yet: uploading an icon (the address is stored, the file is not), pseud pages, and a screen for managing pseuds that works with these endpoints.
- **Authors are shown by pseud**: the public API identifies an author by pseud, not by account ID or username. Readers contact an author through the API (comments, or a relayed message), never by being given an email address.
  - Status: fixed 2026-10-30 (tests and smoke test), apart from the open items at the end.
  - Fixed 2026-10-30 for works, work pages, comments, gifts and the authors list: an author or commenter is `{id, name}` and nothing else, with no `user_id`, `username`, guest email or IP address. A work that is anonymous, in an anonymous collection, or in a collection that hasn't revealed its creators names no author to anyone, and the creator's own comments on it appear as "Anonymous Creator". Those works are also left out of the creator's page, series lists, series counts, and the profile totals (works, words, last posted, kudos received), except for the creator.
  - ~~Series, bookmarks and the user pages are keyed by account (`/users/:user_id/...`) and series responses carry `user_id` and `username`.~~ **Fixed:** `GET /pseuds/:pseud_id/works`, `/series` and `/bookmarks` replace the three account pages (see DECISIONS.md). Series list their creators as `authors: [{id, name}]`, from their works. Bookmarks belong to a pseud (`POST /works/:id/bookmark` takes an optional `pseud_id`, migration 040).
  - ~~Kudos lists show usernames.~~ **Fixed:** each giver is the default pseud `{id, name}`; guests are counted in `guest_count`.
  - ~~Author subscriptions use an account id as `target_id` and the username as `target_name`.~~ **Fixed:** `target_id` is a pseud id, `target_name` the pseud's name; asking with an account id is a 404. No notification is sent for author subscriptions today (nothing fills in a work's authors when an event is sent), so none can name the author of a work that hides its creator. Whoever adds it has to leave those works out.
  - ~~Comment notifications named the commenter's username and account id, even when the commenter was a creator of a work that hides its creators.~~ **Fixed** (migration 039): a hidden creator is "Anonymous Creator" with no id, the name is the pseud's name, then the guest name, then "Someone", and no account id is in the payload. Mention notifications no longer carry the author's account id. The `notification_items` stored by the Go notifier drop the actor id for a hidden creator. (This also fixed `@name` mentions failing outright, and guest comments never notifying the work's creator.)
  - ~~A pseud named "Anonymous Creator" would look like a hidden creator.~~ **Fixed:** the name is reserved (API and database), existing ones were renamed, and `anonymous_creator` comes from the creator-hidden test in the query.
  - ~~Collections show their maintainer's `user_id` and `username`.~~ **Fixed** (migration 041): collection answers (`/collections`, a collection page, `/my/collections`, create and update) name the organiser as a pseud, `organisers: [{id, name, role}]`, as AO3 records collection participants by pseud. `collections.pseud_id` is the creator's default pseud, or the `pseud_id` they chose; deleting that pseud moves the collection to the default pseud. Only an owner exists so far; AO3's moderators and members are not built.
  - ~~Block and mute routes and lists used account ids.~~ **Fixed** (migration 041): `POST /api/v1/blocks {pseud_id}`, `GET /api/v1/blocks`, `DELETE /api/v1/blocks/:pseud_id`, and the same under `/api/v1/mutes`. Lists show `{id, name}` of the pseud that was picked and nothing else. The old `/users/:user_id/block`, `/mute`, `/mute-status` and `/my/muted-users` are gone.
  - ~~`@name` mentions look up usernames.~~ **Removed** (migration 041): AO3 has no @mention feature. The trigger function, `create_mentions`, `extract_mentions`, the `mentions` table and any unsent mention notifications are gone. A comment keeps its text as written and notifies nobody for a name in it.
  - Still open: series write routes (create, update, delete, add and remove works) were all failing on a column that doesn't exist and a wrong ownership check; they are fixed, but series editing is otherwise still Unverified.
- **Anonymous works** (as in AO3's anonymous collections and challenges): the author is hidden until revealed.
  - Status: partly fixed 2026-10-30. The work, comment, series and creator-page answers no longer name the creator (see Authors are shown by pseud), with tests for the three ways a work can hide its creator. Search already left these works out of creator searches and showed no authors for them.
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
  - Status: **Working** (2026-10-30). Runs in Postgres; tested against a real database for drafts, restricted works, blocks (which don't hide works), anonymity and each filter (`backend/internal/search/works_test.go`).
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
  - Status: blocks and mutes **Working** (2026-10-30, tested against a real database: `backend/internal/works/blocks_test.go`, `backend/migrations/blocks_mutes_test.go`); reports Unverified.
  - **Blocks** (see DECISIONS.md): between two accounts, both ways, across all pseuds; the blocker picks a pseud. The blocked person still finds and reads the blocker's works and chapters. On those works the work page says `comments_disabled_for_you: true` ("Comments are disabled"), the comment list is empty for them, and comments, replies, edits and kudos are 403 (`comments_disabled`, `kudos_disabled`). The blocker doesn't see the blocked account's comments anywhere. A blocked person can't reply to the blocker's comments on other people's works (`replies_disabled`). Bookmarks and subscriptions are not affected. Nobody is notified about what an account they blocked does. AO3 references: `app/models/block.rb`, `app/validators/not_blocked_validator.rb:18-21`, `app/models/comment.rb:40-55`, `app/models/kudo.rb:14`, `app/models/work.rb:194-216`, `app/controllers/comments_controller.rb:37-51`, `app/helpers/block_helper.rb:17-33`. AO3's block is one way and hides nothing from the blocked person; both differ here on purpose.
  - **Mutes** are per pseud and hide that pseud's works and comments from the muter only (AO3: `app/models/mute.rb`, `app/helpers/mute_helper.rb`; AO3 mutes a user). A work or comment that hides its creator is never hidden by a mute. Bookmarks and series by a muted pseud are not hidden yet (AO3 hides those too).
  - **Silent on works that hide their creator** (migration 042, 2026-10-30, `backend/internal/works/silent_blocks_test.go`): refusing a blocked person on an anonymous work, or one in an anonymous or unrevealed collection, would tell them the hidden creator blocked them. So there nothing is refused and nothing looks different: the comment list, count and work page are exactly what any reader gets, with no `comments_disabled_for_you`. A comment, reply or kudos from the blocked person gets the normal answer but is stored as `shadowed`: only they and moderators see it, it is left out of the public comment, reply and kudos counts and the kudos list, and it notifies nobody (not the database trigger, not the notifier). Editing their own earlier comment on such a work goes through and shadows the edited comment. A block from a creator whose name is shown behaves as above.
  - **Gifts no longer say who blocked you:** a gift to someone with a block against a creator answers 201 like any gift, is stored as shadowed, and is shown only to the work's creators (not the recipient, not the public gift list), and notifies nobody. Creating gifts is limited to 30 per account per hour (429 `rate_limit_exceeded` with `Retry-After`, the same for everyone).
  - **Comment actions check visibility first:** editing, deleting, freezing or moderating a comment you can't see is 404, before any 403 about ownership or the block.
  - Open: a blocked reader still sees a work's comment count in search results and lists on works whose creator is named (the work page and stats show 0).
  - ~~Chapter comments couldn't be posted~~ **Fixed** (migration 043, 2026-10-31): the comments table's `comment_target` check refused a row with both `work_id` and `chapter_id`, which is what `CreateComment` writes. AO3 (`comment.rb`) comments on a chapter and treats the work as the comment's ultimate parent, so the rule is now: every comment has a `work_id`, and may name a `chapter_id` that must be a chapter of that work (a composite foreign key). Counts, notifications, visibility and blocks already read `work_id`. Posting also checks the chapter is visible to the commenter (a draft chapter takes comments only from its creators). Tested in `comments_test.go` and `migrations/chapter_comments_test.go`.
- **Privacy settings wizard, GDPR consent, data rights** (migration 016)
  - Status: Unverified.

## Moderation

- **Hidden works, chapters, series, bookmarks and comments; spam; suspensions and bans; an audit log of every admin action**
  - Intent: follow AO3 (cited in migration 033 and `internal/moderation`), with the same visibility rules in every place works are listed.
  - Status: Works (verified 2026-10-30 by tests against a real database and on a local server).
    - A hidden work shows up in no listing, search, tag page, series, collection, bookmark list or count for strangers and logged-out readers. Its creators see it with a "hidden" flag, and admins and moderators see it. Marking a work as spam hides it too; unmarking unhides it. Nobody can comment on, download or get notified about a hidden work.
    - Comments: with moderated commenting on, a comment waits for the work's creator (or an admin) to approve it and only they can see it meanwhile. Comments marked as spam are shown to admins only, with their replies. A hidden comment keeps its place in the thread but its text is withheld. A frozen thread takes no replies.
    - A banned or suspended account can log in and read, but can't create or change works, chapters, comments, bookmarks, series, collections, pseuds, kudos or their profile. A suspended account also can't delete or orphan works; a banned one can, as on AO3. A suspension ends at its end time with no job needed. Banning or suspending logs the account out.
    - Admin endpoints (`/api/v1/admin/...`) and `nuclear hide|unhide|suspend|unsuspend|ban|unban|roles|audit` do the same things. Moderators can do everything except read the audit log and change roles, and only an admin can act on an admin or moderator. Nobody can act on their own account. Each action writes one row to `admin_actions` in the same transaction; the table refuses UPDATE, DELETE and TRUNCATE.
  - Not done: the creator is not emailed when their work is hidden (AO3 does); AO3's spam checking (Akismet) and the spam review queue; hiding external works (the rebuild has none); the older admin endpoints that change a work's or comment's `status` are audited but still check the legacy `users.role` column.
  - Differences from AO3: AO3's chapter `hidden_by_admin` is stored but never used, here a hidden chapter is readable only by the work's creators and admins. A work with `spam` set but not hidden is treated as hidden. An admin-hidden comment's text is not sent to the browser (AO3 sends it and hides it with a style).

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
- ~~Postgres (with its default password), Redis (since removed), mail and monitoring ports were published on every network interface~~, which on a server puts them on the internet (Docker bypasses the `ufw` firewall). **Fixed:** everything but Caddy is bound to `127.0.0.1`.
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
- ~~**Backups stayed on the server**~~, so losing the server lost them. **Fixed 2026-10-30:** each daily backup is encrypted and copied to any S3-compatible or B2 storage with restic (14 daily, 8 weekly and 12 monthly copies kept, a weekly read-back check). A failed copy turns the run red. The repository password is generated once and held by the admin. See docs/BACKUPS.md. Not yet run against a real repository.
- ~~Redis was still required.~~ The app wouldn't start without it, though it only held rate limits, OAuth consent prompts and two caches. **Fixed 2026-10-30:** Redis is gone; rate limits are counted in memory (per copy of the app), OAuth state is in Postgres, and the caches were removed. See docs/DECISIONS.md.
- ~~Most services' logs had no size limit~~, which on a small disk eventually stops the database. **Fixed 2026-10-30:** every service keeps at most 5 files of 20 MB.
- **Older bugs found while adding moderation (2026-10-30):**
  - ~~A creator's public works list (`GetUserWorks`) filters on `status = 'posted'`~~, which the table never allows (it uses `published`), so other people always saw an empty list. **Fixed:** the list, the profile and dashboard counts and the site statistics now use `published` (the API still accepts `"posted"` and stores `published`). An author viewing their own list also got a 500 error; fixed.
  - ~~Editing a comment (`PUT /comments/:id`) read the wrong URL parameter~~ and always failed. **Fixed:** only the author can edit; the text is cleaned; frozen comments, comments with replies (AO3: "Comments with replies cannot be edited") and comments hidden by an admin are refused; banned and suspended accounts are refused.
  - ~~The works list's page totals counted only the current page.~~ **Fixed:** the total is counted with the same visibility rule and filters as the list.
  - ~~Deleting a work that had tags failed on a `work_tag_summaries` foreign key.~~ **Fixed** in migration 035: the tag summary trigger re-created a summary for a work that was being deleted. Deleting also drops the tags' use counts. The delete handler also no longer tries to clear two tables that don't exist.
  - ~~A work's creator could clear a spam mark an administrator had set~~ (`PUT /comments/:id/moderate`). **Fixed** in migration 035: `comments.spam_by_admin` records who marked it, and only an administrator can clear it. Creators can still approve and reject comments, as in AO3; the route now refuses banned and suspended accounts.
- ~~**The works service logs SQL details and other debug output** at startup and on every request.~~ **Fixed 2026-10-30:** the schema check at startup and the per-request "=== HANDLER CALLED ===", `DEBUG:`, query and arguments prints are gone from the works package (the tags package never had any). Real errors are logged with the id they concern and no SQL. Server errors no longer put the database's error text in the response (`"details"`). A test fails if an ordinary request logs anything.
- ~~The login endpoint has no stricter limit~~ than general browsing (100 requests a minute per IP), which is loose for password guessing. **Fixed:** an account is locked for 5 minutes after 50 wrong passwords (AO3's Devise numbers; kept in the database, answering like a wrong password, with one email to the owner), and one IP gets 20 requests per 5 minutes across login, password reset and activation. Both are counted in `nuclear_login_failures_total` and `nuclear_login_lockouts_total`, with the alert "LoginFailuresSpike".
- **`X-Test-User-ID` logs anyone in when `GIN_MODE=test`.** It is off by default, but it shouldn't exist in production code.
- If a Hetzner deployment from the old `deploy-*` workflows is live, it is still running the old, vulnerable code until it's redeployed.

## Setup and maintenance problems (found 2026-10-29)

- ~~Old test suites connected to a built-in local database address (`ao3_nuclear` on localhost) and deleted rows whose names start with "test"~~, so running the full test suite on a developer's machine could delete data from their own database. The app itself also fell back to that address when `DATABASE_URL` was unset, and a server started that way would apply migrations to it. **Fixed 2026-10-30:** tests only use `TEST_DATABASE_URL` and skip when it isn't set, and the app refuses to start without `DATABASE_URL`.

- ~~Migrations didn't apply to an empty database~~, and Postgres's first-start scripts stopped at the first error, so a new install couldn't create its database. **Fixed:** every migration applies to an empty database, and the app applies them itself on startup (`nuclear migrate`), one transaction per file, recording each in `schema_migrations`. CI checks this on every change. A database built by the old method is refused with a pointer to [RUNBOOK.md](RUNBOOK.md).
- ~~The code used seven tables that no migration created~~ (notifications, subscriptions, notification and profile preferences, collection membership, hit history). **Fixed** in migration 022. A test now fails if the code queries a table that doesn't exist.
- ~~Database triggers broke posting~~: a retention trigger read columns that don't exist, the chapter counter rejected every new draft, the tag summary joined on a missing column, and the user-stats trigger failed for guests. **Fixed** in migrations 016 and 023–026. The retention trigger was removed rather than fixed: it would have expired works after seven years, which AO3 doesn't do.
- **Reading a single chapter always returns 400.** The route is `/chapters/:chapter_id`, but the handler reads `chapter_number`.
- **Block, mute and report crashed on every request** (500) because of a type mismatch; fixed 2026-10-29. Block and mute were then rewritten by pseud (2026-10-30).
- **118 of 422 frontend unit tests fail.** They all failed before the Vite move too (compared test by test). Two suites that tested components that don't exist (`LoginForm`) were deleted, and two Next.js config checks were replaced by a check that every page has a route.
- ~~The dashboard, subscriptions page, work page and subscribe/gift buttons never knew who was logged in~~: they used a stub `useAuth` that always returned nobody, so the dashboard showed "Loading" forever. **Fixed 2026-10-30:** they use the real one (`src/utils/auth.ts`). It now returns the same objects between renders, because the subscriptions page re-fetched in a loop until rate-limited.
- ~~Search never returned anything~~: works were never sent to Elasticsearch (the code that did it had no callers), and the website's search client crashed on its own address handling. A search with no matches also showed every work instead. **Fixed 2026-10-30** by the move to Postgres search.
- **The works and auth test suites use the developer's own Postgres** (`localhost:5432/ao3_nuclear`) instead of a fresh database, so results depend on that machine. They should use `internal/testdb`, which builds a migrated database per test, as the search tests do.
- **Work gifts need a login to read** (`GET /works/:work_id/gifts` is behind `RequireUser`). On AO3, gifts show to everyone.
- **API endpoints the website calls that fail** (seen 2026-10-30 with a fresh database): `GET /my/bookmarks` returns 500 ("Failed to count bookmarks"); the profile and pseud endpoints return 404; work gifts return 401 for logged-out readers, so the work page shows "Error loading gifts".

## Found while making `go test ./...` pass (2026-10-31)

Fixed, each with a test:
- Deleting a chapter always failed (the renumbering query had an unused parameter, and renumbering broke the unique `(work_id, chapter_number)` check).
- The older `/admin/works`, `/admin/comments` and `/admin/reports` endpoints refused every administrator: they read `users.role`, which nothing sets. They now read `user_roles`, like the route guard. `GET /admin/statistics` did the same and also queried a column (`users.last_seen_at`) that does not exist.
- `PUT /admin/comments/:id/status` answered 500 for statuses the table does not allow; it now says 400 and accepts only `published` and `hidden`.
- OAuth "client credentials" tokens could not be stored (`oauth_access_tokens.user_id` was NOT NULL), and any request with no usable client address failed on the `inet` column. Userinfo and introspection failed for every account with no bio, location or website (NULL scanned into a string).
- The OAuth consent screen approved itself whenever Gin was in test mode. Removed; the server already refused to start in that mode.
- `POST /works` accepted a work with no title, language, fandom or text (the `validate` tags are never read) and answered 500 for a bad rating. It now answers 400.
- An unknown AO3 number in `/works/:id` answered 400 instead of 404; an empty gift list was `null`; empty tag lists were `null`; the notification list failed (500) when any notification had no link or actor name.

Still open:
- Tag prominence (migration 010) is only a schema: its trigger joins `work_tags.tag_name`, which does not exist, and no Go code uses it. Its test is skipped with that reason.
- `internal/auth/oauth_performance_test.go` is an opt-in load test (`RUN_PERF_TESTS=1`); it does not finish within a few minutes and has not been brought up to date.
- Handlers with no route (dead code): `GetWorkWithTags`, `UploadWorkAttachment`, `SearchWorksAdvanced` in `internal/works/enhanced_handlers.go`.
- **The cookie notice covers the consent page's Allow and Deny buttons** on a laptop-sized screen (seen 2026-11-02).
