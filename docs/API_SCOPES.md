# What an app can do: API scopes

This is the table behind "Apps get only what the person approved"
(docs/DECISIONS.md, 2026-11-02). The code is `backend/internal/platform/appaccess.go`;
this page is the same thing in words. A test (`backend/cmd/nuclear/appaccess_test.go`)
walks every route that is really mounted and fails if the two disagree.

## How it works

- A **login token** (the person at their own browser) can do everything the person can do. Nothing here changes it.
- An **app token** (an approved OAuth app acting for a person) is checked on every request, before any handler runs:
  1. It must be live: not expired or revoked, the app still `approved`, and the person active and in good standing (not banned or suspended). Staff accounts (moderator, tag wrangler, admin) cannot use app tokens at all: visibility is decided by what the person may see, and an app must never see more than an ordinary reader.
  2. The per-app rate limit is applied (below).
  3. **Default deny.** The route must be in the table below with a scope, and the token must hold that scope. Anything not listed is refused with `403 not_available_to_apps`, so a new route is safe until someone deliberately adds it here.
  4. A route that is listed but whose scope the person did not approve gets `403 insufficient_scope` and `WWW-Authenticate: Bearer error="insufficient_scope", scope="<needed scope>"` (RFC 6750).
- A **hard denylist** is checked before the table (see below), so a scope added to one of those routes by mistake still does nothing.
- Visibility still comes from the database (`can_user_view_*`), but in **reader mode** (migration 050, `as_reader`): the person counts as a logged-in member, so members-only works show, and their own blocks and mutes apply; every creator, owner and staff exception is off. Every handler an app can reach gets its viewer from one helper, `platform.ReaderViewer`, and hands nobody to anything only a creator or owner may see. Writes happen as the person, under one pseud (see "What apps see").

## Allowed routes

| Route group | Routes | Scope | Notes |
|---|---|---|---|
| Works and chapters | `GET /api/v1/works`, `/works/:id`, `/works/:id/chapters`, `/works/:id/chapters/:id`, and the `/work/:id...` aliases | `read` | Includes members-only works the person can see. |
| Comments | `GET /works/:id/comments`, `/work/:id/comments` | `read` | |
| Series | `GET /series`, `/series/:id`, `/series/:id/works` | `read` | |
| Collections | `GET /collections`, `/collections/:id` | `read` | `/collections/:id/works` is not open to apps: an organiser sees unapproved items there, so the answer would show which collections the person runs. |
| Pseud pages | `GET /pseuds/:id/works`, `/series`, `/bookmarks` | `read` | An app sees the person's own pseud page as any other logged-in reader would (no drafts, no private bookmarks, no works that hide their creator). |
| Search | `GET /search/works`, `/search/suggestions` | `read` | |
| Tags | `GET /tags`, `/tags/search`, `/tags/autocomplete`, `/tags/:id`, `/tags/:id/related`, `/tags/:id/works`, fandoms, characters, relationships, hierarchy, and `/stats/popular`, `/stats/trending`, `/stats/usage` | `read` | |
| What the person follows | `GET /subscriptions`, `/subscription-status`, `/user/tags/followed` | `read` | Lists targets (other pseuds, works, tags), never the person's own pseuds. |
| Post a comment | `POST /works/:id/comments`, `/work/:id/comments` | `comments:write` | Posted under the pseud shared with the app (below). |
| Bookmark a work | `POST /works/:id/bookmark` | `bookmarks:manage` | Create only for now. Same pseud rule. |
| Follow and unfollow | `POST /subscriptions`, `PUT` and `DELETE /subscriptions/:id`, `POST /user/tags/follow`, `DELETE /user/tags/follow/:tag_id` | `subscriptions:manage` | New scope (migration 049). |

`openid`, `profile` and `email` are for sign-in (`/auth/userinfo` and the ID token), not for these routes. `/auth/userinfo` checks its own token and scopes.

## Never reachable by an app, whatever scope it holds

| Routes | Why |
|---|---|
| Everything under `/auth/` (the pages and the OAuth endpoints, including `register-client`) and `/api/v1/auth/` (login, logout, register, password reset, change-password, verify-email, sessions, `me`, the admin user routes) | Sign-in and account security. `/auth/userinfo` is the one exception: it is where an app presents its own token. |
| `/api/v1/oauth/`, `/api/v1/apps` | Consent, connected apps, managing apps. An app must not approve itself or edit its own review. |
| `/api/v1/admin/`, `/api/v1/wrangling/` | Moderation and wrangling. The admin-only scopes can't be granted at all. |
| `/api/v1/support` | The Contact and Feedback forms. A person writes to the admins; an app does not do that for them. |
| `/api/v1/blocks`, `/api/v1/mutes` | They name the pseuds a person wants to avoid. |
| `POST /pseuds`, `PUT` and `DELETE /pseuds/:id` | Pseud management. |
| `/api/v1/my/` (works, series, collections, comments, pseuds, stats, exports) and `GET /api/v1/bookmarks` | See "The /my decision". |
| `/api/v1/export...` | Personal downloads. |
| `/api/v1/notification...` and `/ws` | Notifications, preferences, rules and live updates. They announce activity on all of a person's pseuds at once. |
| `GET /works/:id/kudos`, `/work/:id/kudos`, `/works/:id/stats`, `/work/:id/stats`, `/works/:id/authors`, `/works/:id/gifts`, `/works/:id/bookmark-status` | The answer depends on whether the caller is the creator or has already acted ("have I given kudos", owner analytics). That difference would tell an app which works and pseuds belong to the person. |
| Creating, editing and deleting works, chapters, series, collections; co-authors, gifts, orphaning; editing or deleting a comment or bookmark; kudos; reports; tag creation and merging | Not opened yet. Succeeding or failing on an id the app already knows would show whether the person made it. The `write`, `works:manage` and `collections:manage` scopes can still be requested, but open no route in this release. |

Anything else that is not in the first table is refused the same way (default deny).

## The /my decision

`/my/works`, `/my/series`, `/my/collections`, `/my/comments`, `/my/pseuds`, `/my/stats` and `GET /bookmarks` list a person's content **across all their pseuds**. Handing that to an app would link pseuds that the person shared only one of (docs/DECISIONS.md, pseud rules), so apps are **denied** these routes for now. The alternative, limiting them to the pseud shared at consent, is not built: it needs every one of those queries to filter by pseud. An app that wants "this person's works" can read `GET /pseuds/:id/works` for the pseud the person shared with it (`profile` scope; the id is in the ID token and `/auth/userinfo`). The person's own browser keeps all of them.

## What apps see

- Answers name people by pseud only (`{id, name}`); account ids and emails are not in any answer an app can reach (accounts have no username at all). `TestAppAnswersHaveNoAccountDetails` fetches 21 of these routes and the answers to the writes, and fails if the person's or an author's account id or email appears.
- **Posting pseud.** An app posts under the one pseud the person chose for it at consent. If the person chose none (no `profile` scope), it posts under their default pseud, and the person agreed to this by approving a write scope. If the app names a different `pseudonym_id` or `pseud_id` it gets one "that pseud isn't yours" answer whether the pseud is the person's or not, so asking cannot reveal which pseuds are theirs.

## Rate limits

App requests are counted separately from logged-in browsing and from the per-address limits on sign-in:

| Limit | Default | Counted by |
|---|---|---|
| App API requests | **120 per minute** | per (app, person). Two tokens of one app for one person share the count; other people of the same app, and other apps of the same person, have their own. Refused requests count too, so an app can't hammer routes it may not use. |

Past the limit the answer is `429` with `Retry-After`. Change it with `APP_RATE_LIMIT_PER_MINUTE`. Like the other limits, the counts live in the app's memory (see "Redis removed" in docs/DECISIONS.md).

## Audit

Each request updates the token's `last_used`, at most once a minute (the same rule as sessions). The person sees it on the Connected apps page.

## Known limits

- Apps always get the reader's view; creator and staff views are only for the site itself. A test (`backend/cmd/nuclear/reader_mode_test.go`) asks every open read route as a person's app and as an ordinary reader, for a person who owns anonymous, unrevealed, draft and moderator-hidden works and a series made only of them, and fails if the answers differ.
- **Posting is still checked as the person where a block is concerned.** If a hidden creator has silently blocked the person, a comment the app posts is stored as "shadowed" so it never shows to anyone else. The app gets the normal answer, but as a reader it never sees that comment in later lists, which is the one way left for it to notice the block.
- Staff accounts can't use apps (above). A separate ordinary account can.
- The `write`, `works:manage` and `collections:manage` scopes are grantable but open no route yet; their consent text still describes the intended future.
