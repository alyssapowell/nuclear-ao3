# Decisions

Why the project is shaped the way it is. Newest first. Each entry says what was decided, why, and what it costs.

## The guiding constraint

AO3 is run by volunteers. Its outages over the last two years have come from upgrades, moving parts between systems, and recovery that needed outside vendors. Every choice here should make the system easier for a **beginner volunteer** to run, upgrade, restore and extend. When two options are close, pick the one with fewer moving parts.

---

## 2026-10-30: Blocks are account-wide in both directions, and the blocked person can still read

**Decision:**
- **A block is between two accounts and works both ways, across all pseuds.** The blocker picks a pseud (`POST /api/v1/blocks {pseud_id}`), which blocks the account behind it. That account, under any pseud, is blocked from the blocker's account, under any pseud, and the other way round.
- **Works stay readable.** The blocked person still finds the blocker's works in search and lists and reads them and their chapters. Only the comments close: the work page carries `comments_disabled_for_you: true` (the page shows "Comments are disabled"), the comment list for them is empty, and posting, replying or editing a comment, or giving kudos, is a 403 (`comments_disabled`, `kudos_disabled`). (Not on a work that hides its creator: there the block is silent; see the entry below.) A creator is never blocked from their own work; a co-creator's block closes the work for the blocked person even if the other creator hasn't blocked them. Guests are unaffected. Admins and moderators are not subject to blocks.
- **The blocker does not see comments from the blocked account**, under any pseud, anywhere (on their own works, on replies to their comments, and on other people's works). The blocked person still sees the blocker's comments on other people's works, as in AO3, but cannot reply to them (`replies_disabled`).
- **Notifications:** nobody is notified about anything done by an account they blocked, whichever pseud it used.
- **Other interactions follow AO3:** a work can't be gifted to someone with a block against one of its creators (the gift is answered like any other but kept aside, never shown or announced; see "On works that hide their creator, a block is applied silently"); bookmarks and subscriptions are not affected (AO3 has no rule for them).
- **Mutes are per pseud** (`POST /api/v1/mutes {pseud_id}`). A mute hides that pseud's works and comments from the muter only and never covers the account's other pseuds. It does not apply to a work or comment that hides its creator, since hiding it would tell the muter who wrote it.
- **Lists show what the person picked**, as `{id, name}`. They never carry an account id, a username or another pseud. Blocking a second pseud of an account already blocked adds a second entry and answers like any other block, so it can't be used to learn that two pseuds belong together; removing a block matches only the pseud that was picked, for the same reason.
- **AO3's @mentions don't exist**, so the rebuild's version (which resolved `@name` to an account) is gone. Comments keep their text as written. **Collections** name their organiser as a pseud `{id, name, role}`, chosen at creation (the default pseud if none), as AO3 records collection participants by pseud.

**Why:** The owner wants a block to be a real safety tool: the other person can't comment on, reply to or leave kudos on the blocker's work under any name, and the blocker doesn't have to read them. Hiding the works from the blocked person would have been simpler, but then a work vanishing for one reader would show exactly which pseuds are linked to the blocked account. So the works stay readable and only comments close.

**Accepted cost:** this can, thinly, hint that two pseuds are linked: a person who is blocked sees "Comments are disabled" on every work by the blocker's account, whatever pseud it is under, and if they are blocked by a stranger's account they can compare which works are closed to them. The owner accepted this because it is needed for safety, and kept works readable to keep the hint as small as possible. Mutes are per pseud so muting can't link pseuds at all.

**Differences from AO3:**
- AO3's block is one way (the blocker can still comment on the blocked person's works) and is by user. Here it is both ways, by account.
- AO3 hides nothing from the blocked person; here they see comments as disabled on the blocker's works.
- AO3 shows the blocker the blocked person's comments; here they're hidden.
- AO3's mute is by user; here it is by pseud.
- AO3 has one kind of block; the old "comments" and "works" types became full blocks (migration 041).

**What it costs:**
- A pseud can only be blocked or muted if you can find it; there is no way to block "the account" without naming a pseud, on purpose.
- If someone deletes the pseud you blocked, the block stays and your list keeps the name you picked.
- Existing blocks and mutes were moved to the account's default pseud (migration 041). The owner of a comment hidden by a mute or block loses sight of the replies under it too, as with spam.

---

## 2026-10-30: On works that hide their creator, a block is applied silently

**Decision:** this refines the block rule above. When a work hides its creator (`work_hides_creator()`: anonymous, in an anonymous collection, or in a collection that hasn't revealed its creators), a block from one of its creators is never announced:
- The blocked person sees what any other reader sees: the real comments, the real count, and no `comments_disabled_for_you`. Comments, replies and kudos from them get the normal answer, but are stored as **shadowed** (`comments.shadowed`, `kudos.shadowed`). Only they and moderators can see a shadowed comment; it is left out of the public comment, reply and kudos counts and the kudos list; it notifies nobody (the database trigger and the Go notifier both skip it). The giver's own view still shows their kudos as given, and a second kudos is the usual 409.
- If they edit their own earlier comment there, the edit is allowed and the edited comment becomes shadowed.
- A block from a creator whose name is shown (a work that doesn't hide its creators) keeps today's behaviour: comments disabled, 403.
- **Gifts:** a gift to someone with a block against a creator used to be refused (`gift_blocked`), which let anyone try pseuds one at a time to find who had blocked them. It now answers like any gift but is stored as shadowed (`gifts.shadowed`): the recipient doesn't see it, nobody is notified, and only the work's creators see it in the work's gift list. Creating gifts is also limited to 30 per account per hour (429).
- **Counts include your own shadowed items (migration 044).** Without this, someone who left kudos and saw the number stay the same would know they were blocked. So every kudos or comment count shown to a viewer (work page, works lists, series, collection, pseud and tag pages, stats, search results, the kudos total, a comment's reply count) is the public count plus that viewer's own shadowed items on the work (`own_shadowed_kudos`, `own_shadowed_comments`); their own shadowed kudos also appear in their view of the kudos list. Everyone else, the creator included, sees the public count. Sorting and filtering by kudos or comments (`kudos_min`, `sort=kudos`) still use the public count, a trade-off: a filter's cut-off is a blunt tool, not a precise way to detect a block.
- **Replies to a blocker's comment:** on a work that hides its creator, a reply to a comment by someone who blocked you is always accepted and shadowed, whether or not the work's creator blocked you. On a signed work it is still refused (`replies_disabled`).
- **Best-effort:** silent blocks are made as hard to notice as possible, but they can't be made undetectable. Someone who compares what they see with a second account, or with a friend's view, could still notice a difference (their comment is missing from the other account's list, or the counts differ by one). Treat the silence as best-effort, not a guarantee.
- **Order of checks:** editing, deleting, freezing or moderating a comment now checks that the comment exists, is visible to the caller and is theirs (or they may act on it) before any block is looked at. A stranger asking about a comment they can't see gets 404, not 403.

**Why:** the whole point of hiding the creator is that readers can't tell who wrote the work. "Comments are disabled for you" on an anonymous work tells the blocked person that its author is someone who blocked them, and combined with other blocks or gifts it can narrow down who it is. Saying nothing, and quietly keeping what they write out of sight, gives them nothing to learn from.

**What it costs:** a blocked person's comment on such a work looks posted to them but nobody else sees it, so a block there can't be told apart from being ignored. The blocker never sees the comment either, which is what a block is for. If the collection later reveals its creators, the block then closes comments as it does on any signed work, and the comments written in the meantime stay shadowed.

---

## 2026-11-02: Keep the OAuth / OpenID Connect provider, and publish it only once it's safe

**Decision:** Keep "Sign in with the archive" and API access for other apps. It's one of the community's biggest asks, and AO3 has no way to offer it today. It stays off the public front door until it has been reviewed and hardened.

**Rules it has to meet before it's published:**
- **No way to link people across apps.** An app gets a different, stable identifier for each person (a "pairwise" subject), so two apps can't compare notes to find out they have the same user. Pseud details go to an app only when the person chooses to share them.
- **The pseud rules apply.** Apps see pseuds, never account ids, usernames or (without explicit consent) email addresses. An app must not be able to work out that two pseuds belong to the same person.
- **Apps get only what the person approved.** Each API route checks the token's scopes. Admin and moderation scopes can never be granted to an app.
- **Apps are vetted.** Anyone can register an app, but it stays inactive until an admin approves it. Self-registered apps can't use the client-credentials grant, and public apps must use PKCE.
- **Everything an app does can be undone:** people can see and revoke every app they've approved, and a password change or reset revokes all of them (done).

## 2026-10-30: The public API names people by pseud only, and drops the account pages

**Decision:**
- **No account-keyed pages.** `GET /users/:user_id/works`, `/series` and `/bookmarks` are gone. Their place is `GET /pseuds/:pseud_id/works`, `/series` and `/bookmarks`, as in AO3's `/users/:login/pseuds/:pseud/...` pages. I did not keep AO3's `/users/:login/...` pages keyed by username.
- **Series** show their creators as pseuds (`authors: [{id, name}]`), taken from the creatorships of the works in them, as AO3's `Series#allpseuds` does. A work that hides its creator adds nobody, except to its own creator. A series with no works yet is credited to its owner's default pseud.
- **Kudos** are listed as the giver's default pseud (AO3 lists the giver's login, which is also the name of the default pseud at sign-up). Guests are only counted (`guest_count`).
- **Author subscriptions** follow one pseud. `target_id` is a pseud id and `target_name` is the pseud's current name. AO3 subscribes to a user.
- **Bookmarks** belong to a pseud, as in AO3 (`bookmarks.pseud_id`, migration 040). The account id stays in the table for privacy checks and is never shown.
- **"Anonymous Creator" is reserved** as a pseud name (ignoring case and spaces; a username that would give that default pseud is refused at sign-up). The label marks a creator of a work that hides its creators, and the marker is now decided by the database query, never by comparing the name.

**Why:** The requirement is that the API reveals an author's pseud and nothing about the account behind it. AO3 itself publishes the username in page addresses and next to each pseud ("Pen Name (login)"), so a username-keyed page is the AO3 answer. It does not meet this requirement: one list of everything a person wrote under all their pseuds, reachable from a name, ties their pseuds together. The only front-end callers of the account pages were the signed-in user's own dashboard and series list, which already have `/my/...`. A page for a single pseud is also what the AO3 pseud pages are.
- Subscriptions are per pseud for the same reason. A subscription to an account would tell its subscriber that a second pseud belongs to the same person, as would a "subscribed?" check that was true for both. Deleting a pseud ends subscriptions to it instead of moving them to the default pseud, for the same reason.
- Existing rows are moved by migration 040: bookmarks to their owner's default pseud, author subscriptions to the author's default pseud.

**Confirmed as a rule (2026-10-30):** pseuds must not be connectable to each other, or to the account behind them, by anything public. Only the person's own view (and admins, where moderation needs it) may show that two pseuds belong to the same person. You subscribe to a named author, and a named author is a pseud. This is deliberately stricter than AO3, and it wins wherever the two disagree.

**What it costs:**
- There is no "everything by this person" page, and there won't be one: it would connect a person's pseuds.
- A subscriber follows what is posted under that pseud only. Someone who writes as two pseuds needs two subscriptions.
- A pseud page for a person with several pseuds shows only that pseud's works. The "your works" page (`/my/works`) is still all of yours.
- Collections, the block and mute routes and `@name` mentions were done in the entry above (migration 041).

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

## 2026-10-30: One pseud table, the way AO3 has it

**Decision:** `pseuds` is the only pseud table. Comments, authorship, gifts and descriptions all use it, and `user_pseudonyms` is gone (migration `037_one_pseud_table.sql`). Names follow AO3's `Pseud` model: 1 to 40 characters, letters, numbers, spaces, underscores and dashes in any script, unique per user ignoring case. Every user has exactly one default pseud, enforced by a partial unique index, and the database creates it (named after the username) in the same transaction as the account.

**Why:** Authorship and comments pointed at different tables, so a pseud existed twice and the two copies could disagree, and the old name rule rejected real AO3 names. The AO3 importer needs one place to put `pseuds`, with the ids its comments, gifts and works refer to.

**How:**
- **Merging:** rows of `user_pseudonyms` that match a pseud (same user, same name ignoring case) become that pseud; the others become new pseuds and keep their ids. Comments and comment kudos follow. If a user's two tables disagreed about the default, the pseud authorship used wins.
- **Name characters in the app, not the database:** Postgres's idea of a letter depends on how the database was created, so the database checks length and blanks, and `ValidatePseudName` checks the characters. The importer must call it, or accept names AO3 would have refused.
- **The database makes the default pseud:** a trigger on `users` does it, so every way of creating an account gets one and a failed registration leaves nothing behind. Imports that bring their own pseuds must disable `create_default_pseud_trigger`.
- **Deleting a pseud** moves its works, comments and gifts to the default pseud, and the default (and the pseud named after the username) can't be deleted, as in AO3.
- **Creators stay hidden:** the API names a creator or commenter as `{id, name}` only. A work that is anonymous, in an anonymous collection, or in an unrevealed collection names no one, is left out of the creator's page, series lists and profile totals, and the creator's comments on it read "Anonymous Creator". (Migration `038`.)

**Cost:** Two views that nothing used (`comments_with_details`, `user_profiles`) were dropped. The comment column is still called `pseudonym_id` (AO3: `pseud_id`). Icon files can't be uploaded yet; only their address is stored. Responses that used `username`, `user_id` and `pseud_name` for authors changed to `{id, name}`, and the frontend comment and gift components follow.

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
- Visibility goes through the same `can_user_view_work` function as the work page, so search can't show a draft or a work restricted to logged-in users. (A block no longer hides works: a blocked reader still finds them; see the entry on blocks.) Searching by creator never matches anonymous works.
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
