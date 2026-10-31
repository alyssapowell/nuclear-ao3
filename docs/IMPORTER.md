# AO3 data migration: gap analysis and importer plan

Scope: what it takes to load a real otwarchive (Rails/MySQL) database into the Nuclear AO3 Postgres schema without losing data or changing who can see what.

Sources read for this document:

- otwarchive `db/schema.rb` (89 tables, schema version 2025_07_04_060231) and `app/models`, `lib`, `config`.
- Nuclear AO3 `backend/migrations/001` to `030` (there is no `005`), and `backend/internal/*` to see which tables the code actually uses.
- `docs/FEATURES.md`, `docs/DECISIONS.md`, `tools/ao3-importer/README.md`.

No database was contacted. Everything below is from reading code and schema. Line references are `path:line` relative to the otwarchive root (for AO3) or `nuclear-ao3/backend/migrations/` (for the rebuild, written as `NNN_name.sql:line`).

Status words used in the table section:

- `maps`: a target exists and every column that matters has a home.
- `partial`: a target table exists but some columns, values or rows have no home, or meaning differs.
- `missing`: no target table. The data would be dropped today.
- `skip (why)`: deliberately not imported.

---

## 1. Summary

### 1.1 Counts

Counts are of the 89 `create_table` calls in `db/schema.rb` (generated from the table in section 2; see the check at the end of section 2).

| Status | Tables |
|---|---|
| maps | 3 |
| partial | 26 |
| missing | 52 |
| skip | 8 |
| Total | 89 |

Only three tables move over without a schema change: `blocks`, `mutes`, `serial_works`. Even those carry behaviour notes (section 2).

Not counted as AO3 tables but also needed: the rebuild's own seeded rows (the `orphan_account` user and pseud in `009_core_ao3_features.sql:79-96`, five "Universe Family" tags in `020_universe_family_relationships.sql:6-12`, and two prominence rules in `010_tag_prominence_system.sql`) must be reconciled with imported rows rather than duplicated.

### 1.2 Top 10 gaps that block a faithful import

Ranked by how much real data or user-visible behaviour is affected, with privacy and safety ranked first because a wrong import there is not recoverable by re-running.

1. **Hidden, spam and ban state has no home. Addressed 2026-10-30:** migration `033_moderation_state.sql` adds `hidden_by_admin` (works, chapters, series, bookmarks, comments), `works.spam` and `spam_checked_at`, `comments.unreviewed` and `comments.iced`, and `users.banned`, `suspended` and `suspended_until`; AO3's `comments.approved = false` maps to the existing `comments.is_spam`. `can_user_view_work` and the new chapter, series, bookmark and comment functions read them, and `034_admin_actions.sql` adds the audit log. `external_works.hidden_by_admin` still has no table. The original gap, for the record: AO3 has `hidden_by_admin` on `works`, `chapters`, `series`, `bookmarks`, `comments` and `external_works` (schema.rb:1278, 290, 1034, 219, 394, 481), `works.spam`, `comments.spam/approved/unreviewed/iced`, and `users.banned/suspended/suspended_until`. The rebuild has none of these on works, chapters, bookmarks or series, and `users` only has `is_active`. `can_user_view_work` (`027_visibility_checks_restricted.sql:6-83`) never looks at them. Importing today would make every admin-hidden work public. Posted-but-hidden is a distinct state from draft: the owner and admins can still see it (`app/models/work.rb:562-570`).
2. **Anonymous and unrevealed collections are only half represented.** `works.in_anon_collection` and `works.in_unrevealed_collection` exist in the rebuild (`009_core_ao3_features.sql:105-106`), but the per-collection settings that drive them (`collection_preferences.anonymous/unrevealed`, `collection_items.anonymous/unrevealed`, schema.rb:312, 340) have nowhere to live, and `can_user_view_work` ignores `in_unrevealed_collection`. AO3 refuses comments, downloads, and hits on unrevealed works (`app/controllers/comments_controller.rb:80-83`, `downloads_controller.rb:57`, `app/jobs/hit_count_update_job.rb`). Without the collection-side rows, revealing a collection later cannot un-hide its works.
3. **Only `works` has a `legacy_id`.** `011_add_legacy_id_for_migration.sql` adds it to `works` alone. AO3 URLs also use numeric ids for chapters (`/works/1/chapters/2`), series, comments, bookmarks, and admin posts, and the importer has no stable way to say "this row is that row" on a re-run for any table other than works.
4. **Pseuds are split across two tables, and the constraints reject real AO3 names.** `pseuds` (`009_core_ao3_features.sql:9-24`) is what `creatorships`, `gifts` and the visibility function use. `user_pseudonyms` (`012_enhanced_comments_and_profiles.sql:34-44`) is what `comments.pseudonym_id` points at, and the check `comment_identity_check` (`012:85-87`) requires every logged-in comment to have a `pseudonym_id`. `pseuds.name` is limited to `^[A-Za-z0-9_.-]+$` (`009:18`) but AO3 allows spaces and any Unicode word character (`app/models/pseud.rb:63-69`). Pseud icons, icon alt text, and co-creator approval history have no columns.
5. **Tags: types, synonyms, parents and metatags are lossy.** AO3 has nine tag types plus unsorted (`app/models/tag.rb:15`: Rating, ArchiveWarning, Category, Media, Fandom, Relationship, Character, Freeform, Banned; type NULL is `UnsortedTag`). The rebuild allows eight different names and has no `media`, `banned` or unsorted. Synonyms are `tags.merger_id` in AO3; the rebuild points `canonical_name` at a name string. Metatag `direct` flag, `unwrangleable`, `adult`, `sortable_name`, wrangler assignments, and every tag-set / nomination table have no home. `taggings` only maps for works; AO3 also tags bookmarks, collections and external works. `filter_taggings` (what makes a search for a canonical tag also find its synonyms and sub-tags) is not modelled, and the rebuild searches the raw `works.fandoms` text arrays.
6. **Work and chapter fields.** No home for `revised_at` (AO3's "updated" date, schema.rb:1278), `backdate`, `major_version/minor_version` (used by "new chapters since you read it"), `work_skin_id`, `imported_from_url` (rebuild `imported_from` is `VARCHAR(100)`), `ip_address`, per-chapter `hidden_by_admin`, per-chapter creatorships, or the `*_sanitizer_version` columns. `works.category` and `works.warnings` are declared as scalar text, but the write path stores array literals and one read path treats them as strings (details in section 2.2). A work can have several categories and several warnings in AO3. Chapter `published_at` is a `date` in AO3; the rebuild stores a `timestamptz`.
7. **Comments lose threading, targets and moderation state.** AO3 comments attach to a chapter, an admin post, a tag, or another comment (`commentable_type`), not to a work. The rebuild has `work_id XOR chapter_id` (`002_create_content_tables.sql:199-216`). No home for `approved`, `unreviewed`, `iced`, `hidden_by_admin`, `user_agent`, `thread`, `depth`, `parent_type`. Comment counts mean something different in each system (section 5.5).
8. **Kudos, bookmarks and counters.** Guest kudos are identified by IP in AO3 (`kudos.ip_address`, unique with work, schema.rb:646). The rebuild requires `user_id` or `guest_session` (`002:223-240`); AO3 also has kudos with neither (orphaned, `app/models/user.rb:176`), which the check constraint rejects. Bookmarks can be of a Work, a Series or an ExternalWork (`app/models/bookmark.rb:44-60`); the rebuild bookmarks only works and keeps tags as a text array. AO3 bookmark tags are real tag rows. Triggers also rewrite counters, `updated_at` and send notifications during load (section 5.3).
9. **Collections and everything under them.** Only a flat collection and a work-membership table exist. Missing: participants and roles, moderation and closed settings, intro/FAQ/rules, sub-collections (`parent_id`), icons, and the whole challenge system (gift exchanges, prompt memes, signups, prompts, assignments, claims, potential matches, tag sets, nominations): 17 tables, 9 for challenges and 8 for tag sets.
10. **Account-level data.** Preferences (about 30 flags, schema.rb:790), reading history and "mark for later" (`readings`), subscriptions (rebuild has two incompatible tables), invitations and the invite queue, fannish next of kin, favourite tags, skins and work skins, `log_items` (suspension/warning history), and all admin-authored content (posts, banner, FAQ, known issues). None of these has a table, or the nearest table has a different shape.

Also notable, and not counted above: the rebuild's chapter trigger copies every chapter's text into two more columns (section 6, migration 032), and the front end renders stored HTML without a sanitizer that I could find (section 3.6).

---

## 2. Table-by-table mapping

Legend for notes: "UUID" means the AO3 integer id needs a new UUID key and a stored `legacy_id`; see section 4. Every AO3 timestamp is UTC (`config/application.rb:50`) and every AO3 boolean is a MySQL `tinyint(1)`.

### 2.1 Accounts and identity

| AO3 table | Purpose | Rebuild target | Status |
|---|---|---|---|
| `users` (schema.rb:1229) | Login, email, Devise state, ban and suspension flags | `users` (`001:13-30`, plus `012`, `015`, `016`, `021` columns) | partial |
| `profiles` (825) | About-me text, location, birth date, profile title | `users.bio/location/birth_date` | partial |
| `preferences` (790) | About 30 per-user switches | `users.preferences` JSONB, `user_preferences`, `user_notification_preferences`, `user_privacy_settings` | partial |
| `pseuds` (933) | Pen names; owner of authorship | `pseuds` (`009:9`) and `user_pseudonyms` (`012:34`) | partial |
| `roles` (1004) | Named roles, optionally scoped to an object | `user_roles` (`001:41`) | partial |
| `roles_users` (1015) | User to role join | `user_roles` | partial |
| `admins` (139) | Separate staff login table with its own passwords | `users` + `user_roles` | partial |
| `blocks` (209) | User A blocks user B | `user_blocks` (`006:50`) | maps |
| `mutes` (719) | User A mutes user B | `user_mutes` (`008:9`) | maps |
| `fannish_next_of_kins` (493) | Designated next of kin for a user | none | missing |
| `favorite_tags` (500) | Tags pinned to a user's sidebar | none | missing |
| `invitations` (598) | Invite codes, who made them, who used them | none | missing |
| `invite_requests` (620) | Invite queue (email, IP, simplified email) | none | missing |
| `user_invite_requests` (1218) | Users asking for more invites | none | missing |
| `admin_blacklisted_emails` (74) | Emails barred from signing up | none | missing |
| `log_items` (684) | Per-user moderation history: warnings, suspensions, bans, with end date | none (`moderation_logs` targets content, not account history) | missing |
| `external_authors` (457) | Off-site authors for archivist-imported works | none | missing |
| `external_author_names` (449) | Names an external author goes by | none | missing |
| `external_creatorships` (469) | Links works to external author names and the archivist | none | missing |
| `work_original_creators` (1268) | Remembers who owned a work before it was orphaned | none | missing |
| `audits` (186) | Generic change history (gem `audited`, `Gemfile:98`) | none | skip (history only; includes IPs and email changes; see 3.5) |
| `last_wrangling_activities` (663) | When a wrangler last acted | none | skip (derived; rebuilt by use) |
| `api_keys` (155) | Keys for archivists' import API | none | skip (operator credentials; re-issue) |

Notes on the partial ones:

**`users`.** Columns and where they go:

- `id` -> UUID, `legacy_id` (new).
- `login` -> `username`. AO3 allows `\A[A-Za-z0-9]\w*[A-Za-z0-9]\Z`, 3 to 40 characters (`app/models/user.rb:195-200`, `config/config.yml:119-120`). Ruby `\w` is ASCII, so this fits the rebuild's `^[A-Za-z0-9_-]+$` and 3 to 50 check. Dry run must still report any older row that does not.
- `email` -> `email`. The rebuild adds a regex check (`001:27`) that AO3 does not apply to stored rows. Report violations; relax the check if any exist.
- `encrypted_password` -> `password_hash`. Hash scheme in section 3.4. `password_salt` has no column (needed for a fraction of old accounts).
- `confirmed_at` -> `is_verified` (boolean only; the timestamp is lost). `confirmation_token`, `confirmation_sent_at`, `unconfirmed_email`: no home. Rebuild uses `email_verification_tokens` with hashed tokens; AO3 tokens are not worth carrying, but `unconfirmed_email` (pending email change) is.
- `banned`, `suspended`, `suspended_until` -> no home. `is_active=false` can express "banned" but not suspension with an end date.
- `created_at`, `updated_at` -> same names. `current_sign_in_at/last_sign_in_at` -> `last_login_at` (one of the two). `sign_in_count`, `current/last_sign_in_ip`, `failed_attempts`, `locked_at`, `unlock_token`, `remember_created_at`, `reset_password_token/sent_at`, `resets_requested`: no home; drop or park in a sidecar table. IP columns are personal data and should be dropped unless a reason exists.
- `accepted_tos_version` -> `user_consent`/`gdpr_consent_date` (016) is close but not the same thing; add a column.
- `renamed_at`, `admin_renamed_at` -> none. They drive AO3's rule limiting how often a login can change (`app/models/user.rb:206`, `username_is_not_recently_changed`). Dropping them lets a user rename immediately after import.
- `invitation_id`, `out_of_invites` -> none (invitations missing).

**`profiles`.** `about_me` (HTML, with `about_me_sanitizer_version`) -> `users.bio` (string of unknown format; `users.preferred_content_format` defaults to html). `location` -> `users.location` (100 chars). `date_of_birth` -> `users.birth_date`. `title` -> none. `date_of_birth_visible` lives in `preferences`.

**`preferences`.** Columns with no home at all, by group:

- Display: `work_title_format`, `hide_warnings`, `hide_freeform`, `view_full_works`, `disable_work_skins`, `skin_id`, `minimize_search_engines`, `disable_share_links`, `banner_seen`, `first_login`, `preferred_locale` (integer id into `locales`), `time_zone` (rebuild has `users.timezone` and `user_notification_preferences.timezone`).
- Privacy: `email_visible`, `date_of_birth_visible`, `history_enabled`, `adult` (the "I am an adult" flag), `allow_cocreator`, `allow_gifts`, `allow_collection_invitation`, `guest_replies_off`.
- Email: `comment_emails_off`, `comment_inbox_off`, `comment_copy_to_self_off`, `kudos_emails_off`, `collection_emails_off`, `collection_inbox_off`, `recipient_emails_off`.

Partial homes: `guest_replies_off` has no match; `users.comment_permissions` ('all', 'registered_users', 'friends', 'none', `012:15`) is not an AO3 concept. `history_enabled` matters because it controls whether reading history is kept at all.

**`pseuds`.** `name` (1 to 40 chars, letters, digits, spaces, underscore, dash, any Unicode word character; unique per user; `app/models/pseud.rb:59-69`), `description` (HTML, `description_sanitizer_version`), `is_default`, `icon_*` (Active Storage; see `active_storage_*` below), `icon_alt_text`, `icon_comment_text`, `delta` (search flag; skip). Rebuild `pseuds` has name/description/is_default only; `user_pseudonyms` has name/description/is_default/`icon_url`. Both tables must collapse into one (section 6, migration 031). AO3 uniqueness is case-insensitive (MySQL collation `utf8mb4_unicode_ci`); `pseuds.name` is case-sensitive `VARCHAR`, `user_pseudonyms.name` is `CITEXT`.

**`roles` / `roles_users`.** AO3 roles seen in code: `tag_wrangler`, `archivist`, `official`, `protected_user`, `no_resets` (`app/models/user.rb:364-394`). The rebuild restricts `user_roles.role` to `user`, `tag_wrangler`, `moderator`, `admin` (`001:44`). `roles.authorizable_type/id` (scoping a role to an object) has no home. The rebuild `user_roles.revoked_at/granted_by` columns have no AO3 source and stay NULL.

**`admins`.** Staff are a separate model with their own login, password and `roles` (a YAML array stored in `admins.roles`). Valid roles: `superadmin board board_assistants_team communications development_and_membership docs elections legal translation tag_wrangling support policy_and_abuse open_doors` (`app/models/admin.rb:2-4`). Everything pointing at an `admin_id` (admin posts, FAQs, known issues, wrangling guidelines, admin activities, log items) needs a target. Mapping staff onto `users` requires a staff-only user per admin (admins have `login` and `email` but are not rows in `users`) and a larger role list.

**`blocks`, `mutes`.** The columns map one to one (`blocker_id/blocked_id`, `muter_id/muted_id`; the rebuild's `reason` stays NULL). `user_blocks.block_type` should be `'full'`. Behaviour differs: the rebuild's `can_user_view_work` refuses to show a blocked or muted viewer an author's works (`027:51-77`), while AO3's `Block` and `Mute` models are separate relations whose effects are enforced in controllers and helpers. I did not read AO3's block and mute behaviour in full (`app/models/block.rb`, `mute.rb`, `app/helpers/mute_helper.rb`); confirm it before relying on the function (section 7, item 8).

### 2.2 Works, chapters, series

| AO3 table | Purpose | Rebuild target | Status |
|---|---|---|---|
| `works` (1278) | One work: metadata, flags, versions | `works` (`002:10` plus columns from 004, 006, 009, 011, 015, 021, 028) | partial |
| `chapters` (290) | Chapter text and chapter-level notes | `chapters` (`002:79`, plus 015) | partial |
| `creatorships` (438) | Which pseuds created a work, chapter or series, and whether they accepted | `creatorships` (`009:33`) | partial |
| `series` (1034) | A named series | `series` (`002:166`) | partial |
| `serial_works` (1024) | Work to series join with position | `series_works` (`002:185`) | maps |
| `gifts` (574) | Work gifted to a pseud or a free-text name | `gifts` (`009:56`) | partial |
| `related_works` (992) | Inspired-by, remix, translation links | none | missing |
| `external_works` (481) | Off-site works that can be bookmarked | none | missing |
| `moderated_works` (710) | Works held for admin review (approved/reviewed) | none | missing |
| `stat_counters` (1107) | Hits, downloads, comments, kudos, bookmarks per work | `works.hit_count/kudos_count/comment_count/bookmark_count`, `work_statistics` (`004:27`) | partial |
| `readings` (978) | Reading history and "mark for later" | none | missing |
| `languages` (652) | Language list with support flags | `works.language` (`VARCHAR(10)`) | partial |
| `locales` (670) | Interface locales | none | skip (interface config; rebuild has its own i18n) |

**`works`.** Column by column:

- `id` -> UUID + `legacy_id` (exists).
- `title` (limit 255 by default) -> `title`. `title_to_sort_on` -> none (derive).
- `summary`, `notes`, `endnotes` (HTML; each with a `*_sanitizer_version`) -> `summary`, `notes`, `end_notes`. Rebuild adds `summary_format` and `notes_format` defaulting to `html` (`015:19-33`); no `end_notes_format`. Sanitizer versions: no home (section 3.6).
- `posted` -> inverse of `is_draft`, and also `status='draft'`. `can_user_view_work` tests `status`, not `is_draft` (`027:36-38`), so the importer must write both consistently. AO3 `posted=false` is a draft.
- `restricted` -> `restricted`. The rebuild also has `restricted_to_users` and `restricted_to_adults`; `027` made the function check both of the first two. `restricted_to_adults` has no AO3 source (AO3's adult check is a per-visitor click-through, stored in the session or in `preferences.adult`, not on the work). Leave false.
- `hidden_by_admin` -> none. **Blocking.**
- `spam`, `spam_checked_at` -> none.
- `in_anon_collection`, `in_unrevealed_collection` -> same-named columns. `works.is_anonymous` (rebuild only; `009:107`) has no AO3 equivalent; `is_work_anonymous()` ORs it with `in_anon_collection` (`009:139-155`). Leave false on import.
- `complete` -> `is_complete`. `expected_number_of_chapters` (NULL means unknown) -> `expected_chapters` and also `max_chapters` (`004:15`); two columns for one idea. The check `chapter_counts_valid` (`002:56`) requires `expected_chapters >= chapter_count`; AO3 has no such rule, so rows where fewer chapters are expected than exist would fail to insert.
- `word_count` -> `word_count`. In AO3 it is the sum over posted chapters. Rebuild trigger recomputes it (`024`).
- `language_id` -> `language` (code such as `en`, via `languages.short`, up to 4 chars). Code with no match must be reported.
- `created_at` -> `created_at`. `updated_at` -> `updated_at`. `revised_at` -> **none**. AO3 shows `revised_at` as the work's "Updated" date and sorts by it (`app/models/work.rb:1007-1011`). It is not the same as `updated_at` (which changes on any save). Importing into `updated_at` is wrong because rebuild triggers overwrite that column (section 5.3).
- `backdate` -> none. With `backdate=false` the chapter-one `published_at` is forced to the day of posting; true lets the author choose (`app/models/work.rb:650-666`). Needed to keep edit behaviour the same.
- `major_version`, `minor_version` -> none. Compared with `readings.major_version_read/minor_version_read` for "updated since you read it".
- `imported_from_url` -> `imported_from` is `VARCHAR(100)` (a site name, per its comment) and `external_id` is a different idea. A URL needs `TEXT`.
- `work_skin_id` -> none (skins missing).
- `ip_address` -> none. Personal data; do not import unless the team needs it for spam review.
- `delta` -> skip (search-index flag).
- `moderated_commenting_enabled` -> `moderate_comments`.
- `comment_permissions` (integer enum: `enable_all`=0, `disable_anon`=1, `disable_all`=2, `app/models/work.rb:219-224`, schema default 0 but model default 1) -> `comment_policy` text ('open', 'users_only', 'disabled', `006:19`) and `disable_comments`. Mapping: 0 to 'open', 1 to 'users_only', 2 to 'disabled'. `disable_comments` and `comment_policy='disabled'` both exist; set both for 2. Note schema default (0) vs model default (1): copy the stored integer, do not reapply a default.
- `user_id` does not exist in AO3 `works`; authorship is only via `creatorships`. The rebuild's `works.user_id NOT NULL REFERENCES users` is the primary owner. Fill with the owner of the earliest approved creatorship (ties by creatorship id); for orphaned works use the orphan user. Co-creators come from `creatorships`.
- Rating, fandoms, relationships, characters, freeforms, category and warnings are **not columns** on AO3 `works`; they are `taggings` (section 2.3). The rebuild stores them twice: in `work_tags` and in denormalised columns (`rating`, `category`, `warnings`, `fandoms[]`, `characters[]`, `relationships[]`, `freeform_tags[]`). Nothing keeps the two in sync (no trigger on `work_tags` updates the arrays; `028_postgres_search.sql:23-24` builds the search vector from the arrays only). The importer must write both.
- `category` and `warnings`: `002:22` and `004:10` make them scalar (`VARCHAR(100)` and `TEXT`). `enhanced_handlers.go:82-85` writes `pq.Array(...)` into them, `handlers.go:146-149` reads them as strings into one-element slices, and `handlers.go:941-942` reads them as `pq.StringArray`. So the shape is currently inconsistent in the code. AO3 allows several of each. They should be `TEXT[]`.
- Rating values: AO3 stores rating as a tag with the names in `config/config.yml:241-246` ('Not Rated', 'General Audiences', 'Teen And Up Audiences', 'Mature', 'Explicit'). The rebuild uses snake_case (`019_fix_rating_values.sql`): `not_rated, general, teen, mature, explicit`. A work with two rating tags (possible in old data) cannot be stored; report.

**`chapters`.**

- `content` (`LONGTEXT`; the table is declared `utf8mb3` with the content column overridden to `utf8mb4`, schema.rb:290) -> `content`. Copy bytes as stored; strip NUL characters (`\0`), which Postgres text rejects; replace invalid UTF-8 sequences and report them.
- `title`, `summary`, `notes`, `endnotes` -> `title`, `summary`, `notes`, `end_notes`.
- `position` -> `chapter_number`, unique per work in the rebuild (`002:97`). AO3 does not declare a unique index on `(work_id, position)` (schema.rb:308-309 only index `work_id`), so gaps or duplicates may exist. Renumber by `(position, id)` and report the work.
- `posted` -> inverse of `is_draft`.
- `published_at` (`date`) -> `published_at` (`timestamptz`). Store as the date at 00:00 UTC and render dates in UTC, or add a `DATE` column. See section 7.
- `word_count` -> `word_count`.
- `hidden_by_admin` -> none. **Blocking** (a hidden chapter inside a visible work).
- `*_sanitizer_version` -> none.
- Rebuild adds `content_format` ('html' default), `raw_content`, `processed_content`. A trigger fills the latter two from `content` on every insert (`015_content_formats.sql:156-197`), tripling storage. Grep finds no reader of either column in `backend/internal` or `frontend/src`.
- Chapter creatorships (`creatorships.creation_type='Chapter'`): rebuild allows the type (`009:41`) but nothing reads it.

**`creatorships`.** `creation_type` is 'Work', 'Chapter' or 'Series' (rebuild CHECK allows exactly those, `009:44`). `creation_id` -> UUID of the right table via the type. `approved` -> `approved` (rebuild default `true`, AO3 default false; copy the stored value). Unique `(creation, pseud)` matches. The visibility function only counts approved creatorships (`027:33`), same as AO3 treating unapproved co-creators as invited but not authors. `app/models/work.rb:466-471`: invited and chapter co-creators can see an unposted work.

**`series`.** `title`, `summary` (-> `description`), `series_notes` (-> `notes`), `complete` (-> `is_complete`), `restricted` (AO3 default `true`; rebuild `false`; copy), `hidden_by_admin` (none). `series.user_id NOT NULL` has no AO3 counterpart: authors are in `creatorships` with type 'Series'. Fill with first creator; also write creatorship rows. `work_count`/`word_count` are maintained by trigger (`002:373-393`) but count only `is_draft=false`; AO3 series counts visibility-dependent. Sanitizer versions: none.

**`serial_works`.** `series_id`, `work_id`, `position` map. `created_at` maps; `updated_at` is dropped. Rebuild enforces `UNIQUE(series_id, position)`; AO3 does not declare this, so check for duplicates with the same renumber rule as chapters.

**`gifts`.** `work_id`, `pseud_id` (nullable), `recipient_name`, `rejected` map. A gift is either a pseud or a free-text name, never both (`app/models/gift.rb:66-68`). Rebuild requires one of them (`009:67`) and caps `recipient_name` at 100 characters containing at least one letter or digit.

**`stat_counters`.** `hit_count` -> `works.hit_count`; `comments_count` -> `works.comment_count`; `kudos_count`; `bookmarks_count`. `download_count` -> none. Rebuild also has `work_statistics` (kept in sync by trigger `004:127`), `work_hits` (per-day history, `022`). AO3 keeps no per-day history, so `work_hits` stays empty after import.

**`related_works`.** `parent_id/parent_type` (Work or ExternalWork), `work_id`, `reciprocal`, `translation`. Needed for the "Inspired by", "Remixes" and "Translations" sections of a work page. `020_universe_family_relationships.sql` is unrelated to this.

**`readings`.** `user_id`, `work_id`, `major_version_read`, `minor_version_read`, `last_viewed`, `view_count`, `toread` (marked for later), `toskip` (hidden from history). Written to Redis first and flushed to MySQL (`app/models/reading.rb:15-21`), so a dump may lag recent activity (section 7). Rebuild has `user_activity_log` (generic log), not a history table. This is the "My History" and "Marked for Later" pages.

**`moderated_works`.** Pending-review flags on works (`approved`, `reviewed`). Used by the spam and hidden-works workflow.

**`languages`.** `short` -> `works.language`. `name`, `support_available`, `abuse_support_available`, `sortable_name`: no home. The importer needs this table to translate `language_id`.

### 2.3 Tags

| AO3 table | Purpose | Rebuild target | Status |
|---|---|---|---|
| `tags` (1194) | Every tag, one table, STI by `type` | `tags` (`002:109`, plus 004, 023) | partial |
| `taggings` (1183) | Tag applied to a work, bookmark, collection, external work | `work_tags` (`002:138`, plus 010) | partial |
| `common_taggings` (428) | Wrangled parent/child (character -> fandom, fandom -> media) | `tag_relationships` (`002:150`) | partial |
| `meta_taggings` (700) | Metatag to subtag (direct and inherited) | `tag_relationships` (type `parent_child`) | partial |
| `filter_taggings` (532) | Precomputed "this work is found under this canonical tag" | none | skip (derived; but the rebuild needs an equivalent, see below) |
| `filter_counts` (521) | Per-filter public and unhidden work counts | `tags.use_count` | skip (derived) |
| `wrangling_assignments` (1320) | Which wrangler covers which fandom | none | missing |
| `wrangling_guidelines` (1327) | Staff-written wrangling guidance | none | missing |
| `tag_sets` (1178) | Container for a set of tags | none | missing |
| `owned_tag_sets` (739) | Named tag set with owners and nomination limits | none | missing |
| `owned_set_taggings` (729) | Tag sets owned or reused by other things | none | missing |
| `set_taggings` (1047) | Tag to tag set | none | missing |
| `tag_set_associations` (1150) | Parent/child inside a tag set | none | missing |
| `tag_set_nominations` (1161) | A user's nominations to a tag set | none | missing |
| `tag_set_ownerships` (1170) | Owners and moderators of a tag set | none | missing |
| `tag_nominations` (1128) | Individual nominated tags | none | missing |

**`tags`.**

- `name` (max 150, unique, case- and accent-insensitive in MySQL) -> `name CITEXT UNIQUE`. CITEXT is case-insensitive but not accent-insensitive, so no collisions can arise on import. Lookups differ: AO3 resolves `Pokemon` to `Pokémon`; the rebuild would not.
- `type` -> `type`. Value map: `Rating`->`rating`, `ArchiveWarning`->`warning`, `Category`->`category`, `Fandom`->`fandom`, `Relationship`->`relationship`, `Character`->`character`, `Freeform`->`freeform`. `Media`, `Banned` and NULL (unsorted) have no allowed value (`002:121`). The rebuild uses `additional` for AO3's "Additional Tags" label, but AO3's type name is Freeform; `additional` should not receive imported rows.
- `canonical` -> `is_canonical` (and the duplicate `tags.canonical`, `004:55`).
- `merger_id` (synonym of) -> `canonical_name` (a name, not an id). Rename-safe only if names never change; map via the merger's name.
- `adult` -> none. `unwrangleable` -> none. `sortable_name` -> none (used for ordering lists). `taggings_count_cache` -> `use_count` (recompute). `last_wrangler_id/type` -> none. `delta` -> skip.
- `created_at/updated_at` -> same. `tags.created_by` (rebuild) has no AO3 source.
- Not-yet-wrangled tags are legal in AO3 and in the rebuild after `023_allow_unwrangled_tags.sql`.
- `tags.description` and `is_filterable` have no AO3 source.
- The rebuild seeds five "Universe Family" tags with `additional` type and parent/child links by `ILIKE` over names (`020_universe_family_relationships.sql`). These are not AO3 data and would give imported fandoms parents AO3 does not assign. Remove the seed before an import or tag its rows.

**`taggings`.** `tagger_id` is the tag (with `tagger_type`='Tag'); `taggable_type/id` is Work, Bookmark, Collection or ExternalWork. Only the Work subset has a table (`work_tags`). Others need a generic table or per-type joins. `work_tags` extras (`prominence`, `prominence_score`, `auto_assigned`, `migration_batch`, `010`) have no AO3 source; default `unassigned`. The trigger `update_work_tag_summary` (`010`, replaced in `025`) fires on every row; disable during load.

**`common_taggings`.** `common_tag_id` is the child, `filterable_id` the parent (`app/models/common_tagging.rb:6-7`). Maps to `tag_relationships(parent_tag_id=filterable, child_tag_id=common_tag, relationship_type='parent_child')`.

**`meta_taggings`.** `meta_tag_id`, `sub_tag_id`, `direct` (true when assigned by a wrangler, false when inherited). Map to `tag_relationships` with a new relationship type (e.g. `meta_sub`) or a separate table; the rebuild's `relationship_type` values are `parent_child`, `synonym`, `related` (`002:155`). Importing only `direct=true` rows and recomputing the rest avoids storing derived data.

**`filter_taggings` / `filter_counts`.** Derived by AO3 from taggings, synonyms and metatags. The rebuild needs the same behaviour to answer "works in Fandom X" including works tagged with a synonym or sub-fandom. Either recompute after import into a new table, or compute at query time. Not importing the AO3 rows is safe only if the rebuild computes them.

**Tag sets and nominations.** Used by challenges (section 2.5). Fandom, character, relationship, freeform nomination limits live on `owned_tag_sets`.

### 2.4 Social: comments, kudos, bookmarks, subscriptions

| AO3 table | Purpose | Rebuild target | Status |
|---|---|---|---|
| `comments` (394) | Threaded comments on chapters, admin posts, tags | `comments` (`002:199`, plus 006, 007, 012, 015) | partial |
| `inbox_comments` (586) | A user's comment inbox: read, replied-to | `notification_items` (`022:12`) | partial |
| `kudos` (639) | Kudos on works by users or guests | `kudos` (`002:223`) | partial |
| `bookmarks` (219) | Bookmark of a work, series or external work | `bookmarks` (`002:243`) | partial |
| `subscriptions` (1118) | Subscribe to a work, user or series | `subscriptions` (`013:83`) and `content_subscriptions` (`022:36`) | partial |

**`comments`.**

- `commentable_type/id`: `Chapter` (a comment on a work attaches to a chapter, `app/models/work.rb:940-945`), `AdminPost`, `Tag`, or `Comment` (reply). `parent_type/parent_id` is the ultimate parent. Map: Chapter -> `chapter_id` (and set `work_id` to NULL to satisfy `comment_target`, `002:211`), Comment -> `parent_comment_id`. AdminPost and Tag have no target.
- `pseud_id` -> `pseudonym_id` (FK to `user_pseudonyms`) and `user_id` (derive from the pseud's user). Guest comments: `name` -> `guest_name`, `email` -> `guest_email`. The check `comment_identity_check` (`012:85-87`) demands either (`user_id`, `pseudonym_id`, no guest name) or (no user, no pseud, a guest name). AO3 guest comments always have a name (`app/models/comment.rb:17`), but orphaned comments (pseud deleted) could have neither.
- `comment_content` -> `content` (HTML, `comment_content_sanitizer_version`; rebuild `content_format` defaults to html).
- `is_deleted` -> `is_deleted`. In AO3 a deleted comment with replies stays as a placeholder in the thread; one without replies is removed.
- `approved` (false means rejected as spam), `unreviewed` (held by moderated commenting), `iced` (frozen, replies disabled), `spam`, `hidden_by_admin` -> the rebuild has `status` ('published','pending','deleted','spam','hidden', `006:41`) and `is_spam`. No `iced`. `status` is a single value but AO3 combines several flags; define the mapping order (hidden_by_admin > spam/approved=false > unreviewed > deleted > published) and keep the raw flags.
- `edited_at` -> `edited_at`. `ip_address` -> `ip_address` (INET; drop on import unless wanted). `user_agent` -> none.
- `depth` -> `thread_level`. `thread` (id of the root comment) and `threaded_left/right` (legacy nested-set values, set by an older plugin; `app/models/comment.rb:393-399` only maintains `depth` and `thread`) -> none. Root can be rebuilt from `parent_comment_id`.
- `kudos_count`, `reply_count` (rebuild) have no AO3 source; recompute.
- Triggers on insert: `handle_comment_notifications_trigger` (`012:390`) would notify for every imported comment. Disable.

**`inbox_comments`.** One row per user per received comment: `feedback_comment_id`, `read`, `replied_to`. The "Inbox" page. The rebuild's `notification_items` has `source_id`, `is_read`, no `replied_to`. Migrate only recent unread rows, or all; but this is user state, not content, and is volatile.

**`kudos`.**

- `commentable_id/type`: type is always 'Work' (`app/models/kudo.rb:4`) -> `work_id`.
- `user_id` NULL means guest; `ip_address` identifies guests (unique with work). The rebuild needs `guest_session` and stores `ip_address INET`. Store the IP's string form in `guest_session` (for example `ip:203.0.113.5`) so the uniqueness matches, and keep `ip_address`.
- Kudos with neither user nor IP (a deleted user's kudos, `app/models/user.rb:176`) fail `kudos_identity` (`002:231`). Use `guest_session='ao3-orphaned:<id>'`.
- AO3 forbids kudos from the work's creator and from guests on restricted works; those are write-time rules, not stored. Nothing to check on import except that `user_id` is not an author.
- `updated_at` dropped.

**`bookmarks`.**

- `bookmarkable_type` ('Work', 'Series', 'ExternalWork', `limit: 15`) and `bookmarkable_id`. Rebuild `bookmarks.work_id NOT NULL` covers Work only.
- `pseud_id NOT NULL` and `user_id` -> rebuild has `user_id` only. Pseud is shown on the bookmark and needed to attribute it.
- `bookmarker_notes` (HTML, with sanitizer version) -> `notes`.
- `private` -> `is_private`. `rec` -> `is_rec`. `hidden_by_admin` -> none. **Blocking.**
- Bookmark tags: AO3 uses real tags through `taggings` with type 'Bookmark'; rebuild has `tags TEXT[]`. Convert tag names to the array and keep `taggings` links for filtering.
- `UNIQUE(work_id, user_id)` in the rebuild (`002:259`): `app/models/bookmark.rb:15-18` rejects a second bookmark of the same item by the same user, so this fits for works; the schema itself does not enforce it, so a dry run should confirm.
- `delta` skip.

**`subscriptions`.** `subscribable_type` is 'Work', 'User' or 'Series' (`app/models/subscription.rb:2`). The rebuild has two subscription tables with different shapes: `subscriptions` (`013:83`: `type`, `target_id`, `events[]`, `frequency`) and `content_subscriptions` (`022:36`: `target_type`, `target_id`, `events JSONB`). The Go code uses both (`subscriptions` 9 mentions, `content_subscriptions` 7). AO3 has no per-event or per-frequency settings; the import should set the defaults that mean "new chapter / new work notification". Pick one table and migrate the other.

### 2.5 Collections, gifts, challenges

| AO3 table | Purpose | Rebuild target | Status |
|---|---|---|---|
| `collections` (370) | A collection or challenge host; name is the URL slug | `collections` (`002:263`, plus 017) | partial |
| `collection_items` (312) | Work or bookmark in a collection with two-sided approval | `collection_items` (`022:85`) | partial |
| `collection_participants` (329) | Pseud with a role in a collection | none | missing |
| `collection_preferences` (340) | closed, moderated, unrevealed, anonymous, challenge flags | `collections.is_open/is_moderated/is_anonymous` | missing |
| `collection_profiles` (355) | Intro, FAQ, rules, notification texts | `collections.description` | missing |
| `gift_exchanges` (543) | Gift exchange challenge settings | none | missing |
| `prompt_memes` (837) | Prompt meme challenge settings | none | missing |
| `prompt_restrictions` (859) | Allowed tags and counts per prompt | none | missing |
| `potential_match_settings` (756) | Matching rules for exchanges | none | missing |
| `potential_matches` (776) | Candidate offer/request matches | none | missing |
| `challenge_signups` (279) | A pseud's signup | none | missing |
| `prompts` (904) | Requests and offers (STI by `type`) | none | missing |
| `challenge_assignments` (238) | Giver to receiver assignments, with fulfilled work | none | missing |
| `challenge_claims` (261) | Prompt-meme claims | none | missing |

**`collections`.**

- `name` (URL slug, unique) -> `name` (added by 017 with a `UNIQUE` and a title-derived default; imported rows must supply the real slug).
- `title` -> `title` (500). `description` (HTML) -> `description`. `email`, `header_image_url` -> none. `icon_*` -> none. `parent_id` -> none (sub-collections). `challenge_id/type` -> none. `multifandom`, `open_doors` -> none.
- `user_id NOT NULL` (single owner) has no AO3 source. Choose the pseud with role 'Owner' in `collection_participants`; AO3 can have several.
- `type` ('collection','challenge','exchange','prompt_meme', `002:277`): derive from `challenge_type` (GiftExchange, PromptMeme) and preferences.
- Rebuild `is_open`, `is_moderated`, `is_anonymous` come from `collection_preferences.closed/moderated/anonymous`; `unrevealed`, `gift_exchange`, `prompt_meme`, `show_random`, `email_notify` have no home.

**`collection_items`.** `collection_id`, `item_id`+`item_type` ('Work' default, or 'Bookmark'). Rebuild has `work_id` (works only) and a single `is_approved`. AO3 has two independent three-state statuses (`user_approval_status`, `collection_approval_status`: -1 rejected, 0 unreviewed, 1 approved, `app/models/collection_item.rb:16-26`) and per-item `anonymous` and `unrevealed`. Mapping `is_approved` = both approved loses invited, rejected and pending states, which decide whether a work is "in" the collection and whether anonymity applies. Add the two statuses and the two flags. `added_by`, `approved_at` have no AO3 source.

**Challenge tables.** Copy one to one with UUID keys; column lists are in schema.rb (the table reference above). `prompts.type` is STI ('Request' or 'Offer'). `challenge_assignments.creation_type/creation_id` point at the fulfilling work.

### 2.6 Admin, support and site content

| AO3 table | Purpose | Rebuild target | Status |
|---|---|---|---|
| `admin_posts` (96) | News posts, translations, with comments | none | missing |
| `admin_post_tags` (89) | Admin post tags | none | missing |
| `admin_post_taggings` (81) | Post to tag join | none | missing |
| `admin_banners` (67) | Site-wide banner | none | missing |
| `admin_activities` (56) | Staff action log (target, action, summary) | `moderation_logs` (`014:23`) | partial |
| `admin_settings` (111) | Site toggles: account creation, invite queue, downloads, guest comments, tag wrangling | `system_settings` (`016:284`) | skip (operator config, set fresh; see 7) |
| `abuse_reports` (14) | Public abuse form submissions (URL, email, text) | none (`reports` targets known objects) | missing |
| `feedbacks` (506) | Support form submissions | none | missing |
| `archive_faqs` (175) | FAQ categories | none | missing |
| `archive_faq_translations` (165) | Translated FAQ titles | none | missing |
| `questions` (966) | FAQ questions | none | missing |
| `question_translations` (952) | Translated questions | none | missing |
| `known_issues` (630) | Staff-posted known issues | none | missing |

`admin_activities` -> `moderation_logs`: the target is a polymorphic id and type; `moderation_logs.target_type` allows only work, comment, user, collection, and `moderator_id` must be a `users` row, but staff are `admins` (see accounts). `abuse_reports` and `feedbacks` hold email addresses and IPs of people who may not have accounts; keep only if the team wants history.

### 2.7 Other: skins and file storage

| AO3 table | Purpose | Rebuild target | Status |
|---|---|---|---|
| `skins` (1064) | Site skins and work skins (STI by `type`; NULL type is a site skin) | `users.skin_theme` (a name only) | missing |
| `skin_parents` (1056) | Skin inheritance order | none | missing |
| `active_storage_blobs` (38) | Uploaded file metadata (S3 in production) | none | missing |
| `active_storage_attachments` (28) | Which record owns which file (icons for pseuds, collections, skins) | none | missing |
| `active_storage_variant_records` (50) | Resized-image bookkeeping | none | skip (derived) |

`skins.css` is up to `LONGTEXT`, `public`, `official`, `rejected`, `unusable`, `cached`, `in_chooser`, `featured`, plus form-wizard fields (`margin`, `font`, `base_em`, colours). `works.work_skin_id` points at work skins (STI class `WorkSkin`, `app/models/work_skin.rb`). Icons: `has_one_attached :icon` on `Pseud`, `Collection`, `Skin` (`app/models/pseud.rb:7`, `collection.rb:6`, `skin.rb:51`); production uses S3 (`config/environments/production.rb:36`). The old `icon_file_name/content_type/file_size/updated_at` columns remain on the tables from the Paperclip era; whether they hold live data or only stale values needs checking on the real dump.

### 2.8 Check on the counts

The table lists above contain 89 distinct AO3 tables. Status totals are `maps` 3 (`blocks`, `mutes`, `serial_works`), `skip` 8 (`audits`, `last_wrangling_activities`, `api_keys`, `locales`, `filter_taggings`, `filter_counts`, `admin_settings`, `active_storage_variant_records`), `partial` 26, `missing` 52. Re-count with:

```
grep -E '^\| `[a-z_]+` \(' docs/IMPORTER.md | awk -F'|' '{s=$(NF-1); gsub(/^ +| +$/,"",s); sub(/ \(.*/,"",s); print s}' | sort | uniq -c
```

(This matches only the rows that name a table with its schema.rb line, and prints the status column. It returns 89 rows.)

---

## 3. Identity and privacy

### 3.1 Users, pseuds and authorship

Order and rules:

1. Load `users` first. Key on `legacy_id` (new). `orphan_account` is special (3.3).
2. Load `pseuds` for every user. Keep each pseud's `user_id`. Every user has exactly one `is_default` pseud (AO3 has `has_one :default_pseud`). The rebuild trigger `ensure_default_pseud` (`009:375`) acts only on delete and `assign_user_default_role` (`001:166`) adds a role row on user insert; disable both during load.
3. Authorship comes only from `creatorships`. `works.user_id` in the rebuild is a derived convenience (3.2). Never infer authorship from `works.user_id`.
4. Comments, bookmarks, gifts, collection participants, signups and prompts all reference a pseud, not a user. Keep the pseud id on each. The rebuild's `comments.user_id` and `bookmarks.user_id` are derivable from the pseud.
5. Pseud names are case-insensitively unique per user in AO3; keep names exactly as stored (no normalisation) and make the target column `CITEXT`.

Pseuds hold the public name. An account's login is not shown on a work page when a pseud is used: the byline is `pseud (login)` only when they differ (`app/helpers/application_helper.rb:95-130`). Responses from the rebuild must keep naming creators by pseud, which `docs/DECISIONS.md:76` already states.

### 3.2 Choosing `works.user_id`

The rebuild requires `works.user_id NOT NULL`. Rule: the user of the pseud on the lowest-id approved `creatorships` row for the work (type 'Work'). If the work has none (possible for data corruption, or an external-archivist import not yet claimed), use the orphan user and report. Series use the same rule against `creatorships` of type 'Series'.

### 3.3 Anonymous and unrevealed works

AO3 behaviour, from code:

- A work is anonymous when `works.in_anon_collection` is true; unrevealed when `works.in_unrevealed_collection` is true (`app/models/work.rb:572-582`). Both are denormalised from `collection_items` (`anonymous`, `unrevealed`) and the collection's preferences (`app/models/collection_item.rb:46-52, 61-79`).
- An anonymous work displays "Anonymous" as byline; the real creators are shown in brackets only to an admin or to the creators themselves (`app/helpers/application_helper.rb:71-78`).
- Unrevealed works show a hidden title page, cannot be commented on (`comments_controller.rb:80-83`), cannot be downloaded (`downloads_controller.rb:57`), receive no hits (`hit_count_update_job.rb`), are excluded from the search index of creator and tags (`app/models/search/work_indexer.rb:115`) and cannot be shared (`views/works/_work_header_navigation.html.erb:99`). Moderators can view them (`app/helpers/works_helper.rb:66-71`).
- Revealing a collection (`app/models/collection.rb:393-398`) updates every approved item's flag, and then each work's flag on save.

Import rules:

- Copy `in_anon_collection` and `in_unrevealed_collection` verbatim, and import `collection_items` and `collection_preferences` in the same run so that later reveal/unreveal works. Do not recompute flags from items on import; compare them in the dry run and report a mismatch.
- Do not import `works.is_anonymous` from anywhere: leave it false (it has no AO3 source).
- Anonymous works keep their `creatorships` rows. The rebuild function `get_work_authors` already replaces them with "Anonymous" for non-owners (`009:158-200`). Every Go query that lists a work's creators must go through that function; direct joins on `creatorships` would reveal them. I found five mentions of `get_work_authors` and 29 of `creatorships` in the Go code (counted by a text search over `backend/internal`); checking each is outside this document but must precede go-live.
- Unrevealed works are not handled by `can_user_view_work` at all. Before importing, extend it (or the callers) so unrevealed works are hidden from everyone except creators and collection maintainers.
- Anything derived from creators must be built after the flags are in place: author counts, user work counts (`user_statistics`, trigger `update_user_stats`), author pages (`/users/:login/works`) and search facets. AO3 excludes anonymous and unrevealed works from the creator's work list; the rebuild trigger counts all works with `user_id` (`012:245`, `026`).
- Comments left by a creator on their own anonymous work are shown as "Anonymous Creator" in AO3 (`app/models/comment.rb`, cache poke `poke_cached_creator_comments`, `work.rb:586-590`); any export of comments by user must exclude those.
- Notifications: do not send any notification, email or digest during the import (section 5.3).

### 3.4 Orphaned works

- AO3 orphaning moves creatorships from the author's pseuds to the `orphan_account` user. `Creatorship.orphan(pseuds, works, default)` (`app/models/creatorship.rb:249-256`) either uses orphan_account's default pseud or finds or creates a pseud of the same name under orphan_account. So orphan_account owns many pseuds, one for each name kept.
- The rebuild seeds a single `orphan_account` user (`00000000-0000-0000-0000-000000000000`, password hash `!`) and one pseud (`...0001`) (`009:79-96`). `orphan_work()` only uses that default pseud (`009:204-250`).
- Import: map the AO3 user with `login = 'orphan_account'` onto the fixed UUID (do not create a second user), map its default pseud onto the seeded pseud, and insert its other pseuds as extra rows under the same user. Pseud names are unique per user, so no collisions; check with dry run.
- `work_original_creators` records who owned the work before orphaning (`app/models/work.rb:47`). It is the only way to un-orphan. Import it into a new table (section 6, migration 033); it is private (staff and the former owner use it) and must never be exposed in an API.
- Orphaned kudos (user deleted, not orphan_account) exist, see kudos.
- The orphan account's `login` is required to exist (`app/models/user.rb:517-523`); `ORPHANING_ALLOWED: true` (`config/config.yml:616`). The user should have no usable password.

### 3.5 Emails and passwords

Facts:

- AO3 uses Devise with the `bcrypt` gem (`Gemfile:65-67`). `config/initializers/devise.rb:99` sets `stretches` to 14 outside tests. The pepper line is commented out (`devise.rb:102`), and no `config.encryptor` is set (`devise.rb:204-212` are comments), so Devise uses its default bcrypt scheme: `bcrypt(password)` with cost 14 and **no pepper**. The stored string in `users.encrypted_password` is a standard bcrypt hash like `$2a$14$...`.
- `User` also includes `BackwardsCompatiblePasswordDecryptor` (`app/models/user.rb:19-20`, `lib/backwards_compatible_password_decryptor.rb`). If Devise's normal check fails, it tries two older schemes:
  1. bcrypt over `password + password_salt` (Authlogic style, `lib/backwards_compatible_password_decryptor.rb:24`). So some accounts' hashes only verify when the stored `password_salt` is appended to the typed password.
  2. If `encrypted_password` is not a valid bcrypt string, a legacy SHA-512 scheme: `digest = password + password_salt`, then SHA-512 hex applied 20 times (`:36-38`). A code comment calls these "all 1 of them".
- Passwords on AO3 are 6 to 40 characters (`config/config.yml:121-122`); Go bcrypt accepts these (limit 72 bytes).

What the rebuild does: `bcrypt.CompareHashAndPassword` on `users.password_hash` (`internal/auth/auth_handlers.go:59`, `passwords.go:304`), the same Go library that parses `$2a$` hashes. Its password rule is also 6 to 40 (`passwords.go:36-37`).

Conclusion:

- **The rebuild can verify AO3 hashes at first login with no code change** for every account whose hash was made with plain Devise bcrypt. Copy `encrypted_password` into `password_hash` unchanged. Cost 14 makes each check about as slow as AO3's own; it is not a problem for correctness.
- Accounts that only verify with the salt (scheme 1) need an extra column `legacy_password_salt` and a second comparison against `password + salt` when the first fails. Both schemes produce a valid bcrypt string, so the dry run cannot tell which an account uses without the password. Copy `password_salt` for every user that has one, implement scheme 1 as a fallback, and rehash on success (set `password_hash` to a fresh bcrypt of the typed password, clear `legacy_password_salt`).
- Scheme 2 (non-bcrypt strings; a hash that does not start with `$2`) can be verified by implementing the 20-round SHA-512 or handled by forcing a password reset. With "1 of them" in the code comment the forced reset is the cheaper choice, but the count must be measured: `SELECT count(*) FROM users WHERE encrypted_password NOT LIKE '$2%'`.
- Rehash on first successful login to the rebuild's own cost setting if the team chooses a different one.
- Emails: copy `users.email` as stored. Normalise only for lookups (CITEXT does that). `invite_requests.simplified_email` and `admin_blacklisted_emails` are used to prevent re-signups and need the same table of banned emails. Do not copy `audits` (it records email changes) or any IP columns by default.
- Password reset tokens (`reset_password_token`) and confirmation tokens are hashed by Devise and expire; do not import them. Users reset through the rebuild's own flow.
- Pending states: users with `confirmed_at IS NULL` are unactivated accounts. The rebuild activation flow needs a token; the importer should either skip them (AO3 purges them after `days_to_purge_unactivated` days, `admin_settings`) or import them as unverified and issue a fresh activation link.
- Devise `lockable`: `locked_at`/`failed_attempts` can be reset on import.

### 3.6 HTML content and sanitising

Every HTML field in AO3 has a `*_sanitizer_version` integer. The current version is 4 (`config/config.yml:304`). `HtmlCleaner#sanitize_field` returns the stored text as-is only if the stored version is >= 4, otherwise it sanitises on read (`lib/html_cleaner.rb:10-16`). So a stored chapter with version < 4 may contain markup that AO3 would strip when displaying it.

The rebuild renders stored HTML using `dangerouslySetInnerHTML` (`frontend/src/components/ContentRenderer.tsx:102,110`, `SearchResults.tsx:423`, `MobileReadingInterface.tsx:455-483`). I found no sanitiser dependency in `frontend/package.json` and no sanitiser in `backend/internal` (a search for bluemonday and "sanitize" returned nothing). Therefore the importer must either sanitise rows whose version is below 4 using rules equivalent to AO3's (`lib/html_cleaner.rb`, `config/initializers/` sanitize config) or the rebuild must sanitise on write and read. Carry the version column across so that sanitised-already rows can be recognised (section 6, migration 032).

---

## 4. ID strategy

Requirements:

- `/works/12345` must keep working (and `/works/12345/chapters/67890`, `/series/N`, `/bookmarks/N`, `/comments/N`, `/users/login`, `/tags/Name`). User and tag URLs use names, not ids (`/users/:login`, `/tags/:name`); works, chapters, series, bookmarks, comments, collections (by name) and admin posts use integers or names.
- A re-run of the importer must not create duplicates.
- Foreign keys must resolve without per-row lookups on 10^8 rows.

What exists: `works.legacy_id INTEGER UNIQUE` with a partial index (`011_add_legacy_id_for_migration.sql`). The Go handler already accepts either a UUID or an integer in `/works/:id` and answers integers with a 301 redirect to the UUID route (`internal/works/handlers.go:28-45`, `cached_handlers.go:138`). Rails ids are 32-bit signed (`id: :integer` throughout schema.rb) except `bookmarks`, `kudos`, `blocks`, `mutes` and `work_original_creators` which use bigint ids with `bigint` foreign keys; `INTEGER` fits all legacy ids that URLs use.

Options:

- **A. `legacy_id` column per table + random UUID.** Importer keeps a map from legacy id to generated UUID, in memory or in a lookup table, to fill foreign keys. Re-runs need the map to survive.
- **B. Make the AO3 integer the primary key.** Would change every FK and the whole backend; not viable.
- **C. `legacy_id` column per table + deterministic UUID derived from (table name, legacy id).** UUIDv5 over a fixed namespace. The importer computes any foreign key without lookups, so batches can run in any order and in parallel, and a re-run writes the same ids.

**Recommendation: C.** A fixed namespace UUID for the import, `uuid_v5(namespace, "works:12345")`, plus `legacy_id INTEGER` with a unique partial index on each table whose rows have public URLs: `users`, `pseuds`, `tags`, `works` (exists), `chapters`, `series`, `comments`, `bookmarks`, `collections`. Reasons:

1. Idempotency without state. `INSERT ... ON CONFLICT (id) DO UPDATE` is enough. No map to keep or lose.
2. Foreign keys resolve by computation, so child tables can be loaded without waiting on parent lookups and shards run in parallel.
3. The redirect route stays what it is today: look up `legacy_id`, answer 301. It is already written for works; the same handler shape serves chapters and series.
4. UUIDs for new content are still random (`uuid_generate_v4()`), so new rows cannot collide with derived ones.
5. The AO3 ids are already public for works, chapters, series, comments and bookmarks, so deriving the UUID from them discloses nothing new. For users and pseuds the id is not public in AO3 (URLs use names), and the derived UUID reveals only the old integer. If that is unwanted, use random UUIDs for those two tables and keep `legacy_id` plus a persisted mapping for them only (they are the smallest tables).

Fixed rows: the orphan user and pseud keep their seeded UUIDs (3.4).

Redirect behaviour to add (chapters shown as an example):

- `GET /works/:legacy_work_id/chapters/:legacy_chapter_id` -> resolve both and redirect to the rebuild's chapter route.
- Respond 404 (not 301) for legacy ids of works that are drafts or hidden for the viewer, same as for UUIDs. Redirects must not confirm that a hidden or anonymous-collection work exists to someone who could not otherwise see it; do the visibility check before the redirect, or redirect only after `can_user_view_work`.
- Unrevealed works: same.
- Keep `legacy_id` forever. Never reuse an AO3 id for a new row.

---

## 5. Import mechanics proposal

### 5.1 Source

Read from a restored copy of a MySQL dump (or a physical backup restored into a scratch MySQL), not from production and not from a live replica.

Reasons:

- Consistency. A dump gives one point in time across all tables. A live replica keeps moving; a long read would need a repeatable-read transaction held for days or would see rows added mid-run. A replica is acceptable for the final delta pass (below).
- Repeatability. The same dump can be run through dry runs, partial runs and the real run.
- Safety. No risk to the live archive; no credentials to production in the importer.
- Matches the existing stub: `tools/ao3-importer/README.md` lists a direct MySQL mode, an API-scraping mode and a file mode. API scraping would lose hidden, restricted and anonymous data, because it only sees what a logged-out visitor sees; it is not a migration path for this purpose.

Cutover plan: bulk-load from the dump, then a delta pass (rows with `updated_at`/`id` greater than the dump's high-water marks) from a replica during a short freeze, then switch DNS. Tables with no `updated_at` (`kudos` has it, `taggings` has it, `readings` has `last_viewed`) need per-table delta rules; deletions in AO3 after the dump (comments, bookmarks, works deleted by authors) need a diff of id sets.

### 5.2 Order of loading (dependency graph)

Each step lists what it needs; steps in one level can run in parallel.

1. **Reference data.** `languages`; fixed seeds (orphan user and pseud).
2. **Accounts.** `users` -> `pseuds` (needs users) -> `profiles`, `preferences`, `roles`/`roles_users`, `blocks`, `mutes`, `fannish_next_of_kins`, `favorite_tags` (needs tags, so after step 3), `invitations` (needs users and external_authors), `invite_requests`, `user_invite_requests`, `admins`.
3. **Tags.** `tags` (no FK to others except `merger_id` to itself; load all rows first, then set synonym links) -> `common_taggings`, `meta_taggings`, `wrangling_assignments` (needs users) -> tag sets and nominations (need pseuds).
4. **Skins.** `skins` (needs users) -> `skin_parents`.
5. **Works.** `works` (no user FK) -> `chapters` -> `creatorships` (needs pseuds and works, chapters, series) -> `series` -> `serial_works` -> `gifts` (needs pseuds) -> `related_works` (needs works and external_works) -> `external_works`, `external_authors`, `external_author_names`, `external_creatorships` -> `moderated_works` -> `work_original_creators` -> `taggings` (needs tags and works, bookmarks, collections, external_works; so after those) -> `stat_counters`.
6. **Collections.** `collections` (parent self-reference: insert all, then set `parent_id`) -> `collection_preferences`, `collection_profiles`, `collection_participants` (needs pseuds) -> `collection_items` (needs works and bookmarks) -> challenge tables (`gift_exchanges`, `prompt_memes`, `prompt_restrictions`, `potential_match_settings`, `challenge_signups`, `prompts`, `potential_matches`, `challenge_assignments`, `challenge_claims`).
7. **Social.** `bookmarks` (needs pseuds, works, series, external_works) -> `kudos` -> `comments` (needs chapters, admin_posts, tags, pseuds; parents before children, or defer the self FK) -> `inbox_comments` -> `subscriptions` -> `readings`.
8. **Admin content.** `admin_posts`, `admin_post_tags`, `admin_post_taggings`, `admin_banners`, `archive_faqs`, `questions` and translations, `known_issues`, `wrangling_guidelines`, `log_items`, `admin_activities`, `abuse_reports`, `feedbacks`, `admin_blacklisted_emails`.
9. **Files.** Copy icons from S3 (or the attachment store) and write `active_storage_*` equivalents or a new file table; point `pseuds`, `collections`, `skins` at them.
10. **Derive.** Counters, `tags.use_count`, filter rows, search vectors, user statistics (5.5).

Because ids are deterministic (section 4), FKs do not have to wait for parents to be committed; but to keep database FK constraints on, load parents first or defer constraints per transaction (`SET CONSTRAINTS ALL DEFERRED` works for constraints declared `DEFERRABLE`; the existing ones are not, so either use the order above or add `DEFERRABLE INITIALLY DEFERRED` to the FKs used in bulk loads).

### 5.3 Batching, triggers, idempotency, resume

- Read by key range: `WHERE id > :last ORDER BY id LIMIT :n` (several thousand rows; `chapters` smaller because of text size). Never `OFFSET`.
- Write with `COPY` into an unlogged staging table, then `INSERT ... SELECT ... ON CONFLICT (id) DO UPDATE` into the target. One transaction per batch.
- Record progress in a table the importer owns: `import_progress(table_name, last_legacy_id, rows_read, rows_written, rows_rejected, status, updated_at)`. Resume starts from `last_legacy_id`.
- Rejects go into `import_rejects(table_name, legacy_id, reason, row JSONB)` so the run does not stop for a single bad row. The dry run counts the same reasons.
- Triggers must be off during the load, because they change data and send mail:
  - `update_work_stats_*` (`002:337-351`, `024`): overwrite `works.word_count`, `chapter_count`, counters and set `updated_at = NOW()`. Imported `updated_at` would be destroyed.
  - `sync_work_statistics_trigger` (`004:127`): creates `work_statistics` on works insert; backfill instead.
  - `update_tag_use_count_trigger` (`002:368`) and `trigger_update_work_tag_summary` (`010:143`, `025`): per-row recounts on `work_tags`.
  - `update_series_stats_trigger` (`002:395`).
  - `update_comment_stats_*`, `update_user_stats_*` (`012:236-316`, `026`).
  - `handle_comment_notifications_trigger` (`012:390`, `013`): would create a notification (and possibly a job-queue email) for each imported comment.
  - `create_user_notification_defaults_trigger` (`013:478`), `assign_user_default_role` (`001:166`), `trigger_update_processed_content` (`015:195`) which copies chapter text into two more columns.
  - `update_users_updated_at` (`001:151`) and other `updated_at` triggers: imported `updated_at` must survive.
  Use `ALTER TABLE ... DISABLE TRIGGER USER` per table, or `SET session_replication_role = replica` if the importer connects as a superuser (this also skips FK checks, so validate afterwards). Re-enable and run the derive step.
- No outbound email or job queue entries from import: the `jobs` table (`029_jobs.sql`) is written by the notifier; confirm it stays empty or flag all import-time rows. Ensure SMTP is not configured during the run.
- Parallelism: tables at the same level of 5.2 in parallel; within `chapters` and `comments`, shard by key range.
- Size: `tools/ao3-importer/README.md` quotes about 10 million works and 100 million comments. Chapter text dominates. `raw_content` and `processed_content` triple that (3.6/section 2.2); turn the trigger off and drop the two columns first.
- Text clean-up on every text column: remove NUL bytes; replace invalid UTF-8; reject `'0000-00-00'` dates (MySQL can hold them; Postgres cannot); convert `tinyint(1)` to boolean. MySQL `utf8mb3` tables (`chapters`, `roles_users`) are fine.

### 5.4 What the importer reads that is not in rows

None of the following is needed to rebuild content; see section 7 for what is lost or must be re-created:

- Search index: Elasticsearch documents are derived (`app/models/search/*_indexer.rb`). The rebuild searches Postgres (`028`), so nothing to import.
- Autocomplete data in Redis (`lib/tasks/load_autocomplete_data.rake`): derived.

### 5.5 Counters and derived data

After loading all rows with triggers disabled, recompute from imported rows, using AO3's definitions, and compare with AO3's stored values:

| Counter | AO3 rule | Source |
|---|---|---|
| `works.kudos_count` | count of `kudos` rows for the work (users and guests) | `app/models/work.rb:964-971` |
| `works.comment_count` | comments on the work's chapters that are not deleted, not hidden by an admin, not unreviewed, and `approved` | `lib/acts_as_commentable/commentable_entity.rb:33-40` |
| `works.bookmark_count` | bookmarks with `private = false` (`bookmarks.where(private: false)`) | `app/models/work.rb:964-971` |
| `works.hit_count` | **copy** `stat_counters.hit_count`; hits cannot be recomputed from rows | `app/jobs/hit_count_update_job.rb` |
| `works.word_count` | sum over posted chapters | AO3 `works.word_count` stored; recompute and compare |
| `works.chapter_count` | posted chapters | derived |
| `tags.use_count` | works (and bookmarks etc. in AO3) tagged, counted through synonyms in AO3's `filter_counts` | derived |
| `series.work_count`, `word_count` | from `serial_works` | derived |
| user statistics | works, bookmarks, comments, kudos | derived |

Differences from rebuild triggers that matter: the rebuild counts every non-deleted comment, including spam, hidden and unreviewed ones (`024`), and every bookmark including private ones. Each makes counters on a work page higher than AO3 shows. Replace the trigger definitions or compute with the AO3 rules and stop the trigger from overwriting them.

AO3's own `stat_counters` can lag: the code recomputes `kudos_count`, `comments_count` and `bookmarks_count` only on `update_stat_counter` (`work.rb:964`), so small drift between stored and recomputed values is expected. The dry run should report the distribution, and the importer should take the recomputed value for the rebuild's rule and keep the AO3 stored value in the report.

Hits are held in Redis until flushed (section 7); the last few hours of hits may be missing from the dump.

### 5.6 Verification

Automatic checks (the importer exits non-zero if any fails):

- Row counts per table: source rows read, rows written, rows rejected, rows skipped on purpose. `read = written + rejected + skipped`, per table, listed in the report.
- Sums that should be equal: total `word_count` (works and chapters), total kudos, total posted chapters, total creatorships by type, total taggings by tagger type, total comments by `commentable_type`.
- Distribution checks for flags: count of works by (`posted`, `restricted`, `hidden_by_admin`, `in_anon_collection`, `in_unrevealed_collection`), identical on both sides.
- Orphan check on every FK without a constraint: creatorship pseud exists; comment chapter exists; tag synonym exists.
- Uniqueness: no two pseuds per user that differ only in case; no tag names that differ only in case (MySQL could not have them, Postgres CITEXT cannot either).
- Visibility check, as a script: for a random sample of 10 000 works and a set of viewers (guest, random user, creator, admin), compare AO3's `Work#visible?` rule (`app/models/work.rb:562-570`) with `can_user_view_work`. Add rules for hidden, unrevealed and anonymous. Mismatch count must be zero.
- Byline check: for the sample, anonymous and unrevealed works show "Anonymous" (or the hidden title) to a guest and a non-creator, and reveal the creator to the creator.

Spot checks (human):

- Render the same 200 works on both sites (a fixed list covering: multi-chapter, one chapter, anonymous, unrevealed, orphaned, restricted, hidden, in a series, with gifts, with co-creators, with emoji, with RTL text, with a very long chapter, with a work skin) and compare title, byline, summary, notes, tags in order, word count, chapter list, comment counts, kudos list. Compare after the rebuild's own HTML treatment, not byte for byte.
- Tag pages: fandom page work counts for 50 tags including a synonym, a metatag and an unwrangled tag.
- Account pages: dashboard counts for 20 users (works, bookmarks, series, gifts).
- Logins: 50 real accounts with known passwords, covering each hash scheme found.

### 5.7 Dry run

`--dry-run` reads from the source and runs every transform and validation, then writes nothing to Postgres (or writes inside a transaction that is rolled back, as a second level `--dry-run=db`, which also catches constraint and trigger errors). The report, per table:

- rows by status (would write, would reject, would skip) with reason counts: pseud name fails check, work expects fewer chapters than exist, kudos with no user and no IP, chapter positions duplicated, tag type not representable, email fails regex, language code unknown, user has no default pseud, comment on unsupported commentable type, date `0000-00-00`;
- the id and derived-UUID collision check;
- predicted counters versus AO3 stored counters;
- the list of triggers that would fire on that table and the planned disable;
- total bytes and estimated time.

The dry run must produce the same reject list as the real run for the same dump.

---

## 6. Schema changes needed in the rebuild

This is an ordered checklist with sketched SQL, not code to apply. Numbers continue from `030`. Names are suggestions. Column types follow existing conventions (UUID keys, `TIMESTAMPTZ`, `CITEXT`).

### 031_identity_and_legacy_ids

Merge the pseud tables, add legacy ids, add account state.

```sql
-- one pseud table: keep `pseuds`, retire `user_pseudonyms`
ALTER TABLE pseuds
  ALTER COLUMN name TYPE CITEXT,
  DROP CONSTRAINT pseud_name_format,                       -- AO3 allows spaces and Unicode word chars
  ADD CONSTRAINT pseud_name_format CHECK (name ~ '^[[:alnum:]_ -]+$' AND name ~ '[[:alnum:]]'),
  ADD COLUMN icon_url TEXT,                                -- or a file reference, see 038
  ADD COLUMN icon_alt_text VARCHAR(250) DEFAULT '',
  ADD COLUMN icon_comment_text VARCHAR(250) DEFAULT '',
  ADD COLUMN legacy_id INTEGER;
-- copy user_pseudonyms rows into pseuds with the same UUIDs, repoint
-- comments.pseudonym_id to pseuds(id), then DROP TABLE user_pseudonyms
-- (backend/internal/auth and works use both tables today; code change needed)

ALTER TABLE users ADD COLUMN legacy_id INTEGER,
  ADD COLUMN legacy_password_salt TEXT,          -- Authlogic-style hashes (3.5)
  ADD COLUMN confirmed_at TIMESTAMPTZ,
  ADD COLUMN unconfirmed_email CITEXT,
  ADD COLUMN banned BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN suspended_until TIMESTAMPTZ,        -- NULL with suspended=true means indefinite
  ADD COLUMN suspended BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN renamed_at TIMESTAMPTZ,
  ADD COLUMN admin_renamed_at TIMESTAMPTZ,
  ADD COLUMN accepted_tos_version INTEGER,
  ADD COLUMN out_of_invites BOOLEAN NOT NULL DEFAULT true,
  ADD COLUMN invitation_id UUID;                 -- FK added in 039
ALTER TABLE users DROP CONSTRAINT email_format;  -- or loosen; report violations first
ALTER TABLE users ADD COLUMN profile_title VARCHAR(255);

CREATE UNIQUE INDEX users_legacy_id ON users(legacy_id)   WHERE legacy_id IS NOT NULL;
CREATE UNIQUE INDEX pseuds_legacy_id ON pseuds(legacy_id) WHERE legacy_id IS NOT NULL;

-- roles: AO3 role names, scoped roles, staff
ALTER TABLE user_roles DROP CONSTRAINT user_roles_role_check;   -- constraint name to confirm
-- allow: tag_wrangler, archivist, official, protected_user, no_resets, plus staff roles
ALTER TABLE user_roles ADD COLUMN authorizable_type VARCHAR(40), ADD COLUMN authorizable_id UUID;
```

Staff: add `users.is_staff BOOLEAN` or a `staff_accounts(id, user_id, legacy_id, login, email, roles TEXT[])` table; each AO3 `admins` row becomes one user plus its role list, with `admin_id` columns elsewhere pointing at the user.

### 032_work_and_chapter_fields

```sql
ALTER TABLE works
  ADD COLUMN hidden_by_admin BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN spam BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN spam_checked_at TIMESTAMPTZ,
  ADD COLUMN revised_at TIMESTAMPTZ,
  ADD COLUMN backdate BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN major_version INTEGER NOT NULL DEFAULT 1,
  ADD COLUMN minor_version INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN imported_from_url TEXT,
  ADD COLUMN work_skin_id UUID,                     -- FK added in 038
  ADD COLUMN summary_sanitizer_version SMALLINT NOT NULL DEFAULT 0,
  ADD COLUMN notes_sanitizer_version SMALLINT NOT NULL DEFAULT 0,
  ADD COLUMN endnotes_sanitizer_version SMALLINT NOT NULL DEFAULT 0,
  ADD COLUMN ip_address INET;                       -- optional; may be left out
ALTER TABLE works
  ALTER COLUMN category TYPE TEXT[] USING CASE WHEN category IS NULL THEN '{}' ELSE ARRAY[category] END,
  ALTER COLUMN warnings TYPE TEXT[] USING CASE WHEN warnings IS NULL THEN '{}' ELSE ARRAY[warnings] END;
ALTER TABLE works DROP CONSTRAINT chapter_counts_valid;   -- AO3 has no such rule
ALTER TABLE chapters
  ADD COLUMN hidden_by_admin BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN legacy_id INTEGER,
  ADD COLUMN content_sanitizer_version SMALLINT NOT NULL DEFAULT 0,
  ADD COLUMN notes_sanitizer_version SMALLINT NOT NULL DEFAULT 0,
  ADD COLUMN summary_sanitizer_version SMALLINT NOT NULL DEFAULT 0,
  ADD COLUMN endnotes_sanitizer_version SMALLINT NOT NULL DEFAULT 0;
-- storage: stop tripling chapter text
DROP TRIGGER trigger_update_processed_content ON chapters;
ALTER TABLE chapters DROP COLUMN raw_content, DROP COLUMN processed_content;
-- views published_works_with_stats / popular_works use w.*; recreate after column changes

CREATE UNIQUE INDEX chapters_legacy_id ON chapters(legacy_id) WHERE legacy_id IS NOT NULL;
CREATE INDEX works_revised_at ON works(revised_at DESC) WHERE NOT is_draft;

CREATE TABLE work_original_creators (               -- private; never exposed
  work_id UUID NOT NULL REFERENCES works(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (work_id, user_id)
);
CREATE TABLE moderated_works (
  work_id UUID PRIMARY KEY REFERENCES works(id) ON DELETE CASCADE,
  approved BOOLEAN NOT NULL DEFAULT false, reviewed BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL);
CREATE TABLE related_works (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  work_id UUID NOT NULL REFERENCES works(id) ON DELETE CASCADE,       -- the child work
  parent_type VARCHAR(20) NOT NULL CHECK (parent_type IN ('Work','ExternalWork')),
  parent_id UUID NOT NULL,
  reciprocal BOOLEAN NOT NULL DEFAULT false,
  translation BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ);
```

Update `can_user_view_work`, `can_user_comment_on_work`, `get_work_authors` and `update_work_stats` in the same migration (see "functions" at the end).

Dates: decide the type for chapter `published_at` first (section 7). If a `DATE` is wanted: `ALTER TABLE chapters ADD COLUMN published_on DATE`.

Series:

```sql
ALTER TABLE series ADD COLUMN legacy_id INTEGER, ADD COLUMN hidden_by_admin BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN summary_sanitizer_version SMALLINT NOT NULL DEFAULT 0,
  ADD COLUMN notes_sanitizer_version SMALLINT NOT NULL DEFAULT 0;
ALTER TABLE series ALTER COLUMN restricted SET DEFAULT true;     -- AO3 default
```

### 033_visibility_and_moderation_functions

Replace the SQL functions; no new tables.

- `can_user_view_work`: add `hidden_by_admin` (visible to creators and admins only, as `app/models/work.rb:562-570`), `in_unrevealed_collection` (creators and collection maintainers only), and keep drafts via `status`/`is_draft`. Admin viewers see everything posted. Decide what blocks and mutes should do, from `app/models/block.rb` and `mute.rb`.
- `get_work_authors`: no change in behaviour; document that every listing must call it.
- `can_user_comment_on_work`: check all co-creators for blocks, not only `works.user_id` (`006:214-262`).
- `update_work_stats`: use AO3's comment and bookmark rules (5.5) and do not touch `updated_at`.
- Admin-visible state on comments: see 035.

### 034_tags

```sql
ALTER TABLE tags DROP CONSTRAINT tag_type_values;
ALTER TABLE tags ADD CONSTRAINT tag_type_values CHECK (type IN
  ('rating','warning','category','media','fandom','relationship','character','freeform','banned','unsorted'));
-- 'additional' becomes a synonym for 'freeform' in code; retire it
ALTER TABLE tags
  ADD COLUMN legacy_id INTEGER,
  ADD COLUMN sortable_name VARCHAR(150) NOT NULL DEFAULT '',
  ADD COLUMN adult BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN unwrangleable BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN merger_id UUID REFERENCES tags(id),            -- real synonym link; keep canonical_name in step or drop it
  ADD COLUMN last_wrangler_id UUID REFERENCES users(id),
  ADD COLUMN last_wrangled_at TIMESTAMPTZ;
CREATE UNIQUE INDEX tags_legacy_id ON tags(legacy_id) WHERE legacy_id IS NOT NULL;

-- metatags: direct flag, same type on both sides, canonical only
CREATE TABLE meta_taggings (
  meta_tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  sub_tag_id  UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  direct BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ,
  PRIMARY KEY (meta_tag_id, sub_tag_id));

-- taggings on things other than works
CREATE TABLE taggings (
  tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  taggable_type VARCHAR(20) NOT NULL CHECK (taggable_type IN ('Bookmark','Collection','ExternalWork')),
  taggable_id UUID NOT NULL,
  created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ,
  PRIMARY KEY (tag_id, taggable_type, taggable_id));

-- what a search for a canonical tag should find (derived; recomputed)
CREATE TABLE filter_taggings (
  filter_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  work_id UUID NOT NULL REFERENCES works(id) ON DELETE CASCADE,
  inherited BOOLEAN NOT NULL DEFAULT false,
  PRIMARY KEY (filter_id, work_id));

CREATE TABLE wrangling_assignments (user_id UUID REFERENCES users(id), fandom_id UUID REFERENCES tags(id), PRIMARY KEY (user_id, fandom_id));
CREATE TABLE wrangling_guidelines (id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), admin_id UUID REFERENCES users(id),
  title TEXT, content TEXT, position INT, content_sanitizer_version SMALLINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ);
CREATE TABLE favorite_tags (user_id UUID REFERENCES users(id) ON DELETE CASCADE, tag_id UUID REFERENCES tags(id) ON DELETE CASCADE, PRIMARY KEY (user_id, tag_id));
```

Also: remove the seeded "Universe Family" tags and links from `020` (not AO3 data) or mark them with a flag so they do not appear in imported data.

Keep `work_tags` as the work-to-tag join; add nothing else. Keep the arrays on `works` in step with `work_tags` by writing both in the importer, or replace them with a view.

### 035_comments_kudos_bookmarks

```sql
ALTER TABLE comments
  ADD COLUMN legacy_id INTEGER,
  ADD COLUMN commentable_type VARCHAR(20),   -- 'Chapter','AdminPost','Tag' for top-level comments
  ADD COLUMN admin_post_id UUID,             -- FK in 040
  ADD COLUMN tag_id UUID REFERENCES tags(id),
  ADD COLUMN hidden_by_admin BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN approved BOOLEAN NOT NULL DEFAULT true,
  ADD COLUMN unreviewed BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN iced BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN user_agent VARCHAR(500),
  ADD COLUMN content_sanitizer_version SMALLINT NOT NULL DEFAULT 0;
ALTER TABLE comments DROP CONSTRAINT comment_target;
ALTER TABLE comments ADD CONSTRAINT comment_target CHECK (
  parent_comment_id IS NOT NULL OR num_nonnulls(work_id, chapter_id, admin_post_id, tag_id) = 1);
-- identity check: keep, but allow user comments whose pseud was deleted?  Decide after dry run counts.
CREATE UNIQUE INDEX comments_legacy_id ON comments(legacy_id) WHERE legacy_id IS NOT NULL;

ALTER TABLE kudos DROP CONSTRAINT kudos_identity;
ALTER TABLE kudos ADD CONSTRAINT kudos_identity CHECK (user_id IS NOT NULL OR guest_session IS NOT NULL OR ip_address IS NOT NULL OR legacy_id IS NOT NULL);
ALTER TABLE kudos ADD COLUMN legacy_id INTEGER;     -- or use the 'ao3-orphaned' guest_session value instead (section 2.4)
CREATE UNIQUE INDEX kudos_unique_ip ON kudos(work_id, ip_address) WHERE user_id IS NULL AND ip_address IS NOT NULL;

ALTER TABLE bookmarks
  ADD COLUMN legacy_id INTEGER,
  ADD COLUMN pseud_id UUID REFERENCES pseuds(id),
  ADD COLUMN bookmarkable_type VARCHAR(15) NOT NULL DEFAULT 'Work' CHECK (bookmarkable_type IN ('Work','Series','ExternalWork')),
  ADD COLUMN series_id UUID REFERENCES series(id) ON DELETE CASCADE,
  ADD COLUMN external_work_id UUID,           -- FK in 036
  ADD COLUMN hidden_by_admin BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN notes_sanitizer_version SMALLINT NOT NULL DEFAULT 0;
ALTER TABLE bookmarks ALTER COLUMN work_id DROP NOT NULL;
ALTER TABLE bookmarks DROP CONSTRAINT bookmarks_work_id_user_id_key;   -- replace with per-type unique indexes
```

Also add `iced`/`approved` handling to the Go comment code; and an `inbox` table if the inbox is to be kept:

```sql
CREATE TABLE inbox_comments (user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  comment_id UUID REFERENCES comments(id) ON DELETE CASCADE,
  read BOOLEAN NOT NULL DEFAULT false, replied_to BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ, PRIMARY KEY (user_id, comment_id));
```

### 036_external_works_and_archivists

```sql
CREATE TABLE external_works (id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), legacy_id INTEGER,
  url TEXT NOT NULL, author VARCHAR(255) NOT NULL, title VARCHAR(255) NOT NULL, summary TEXT,
  language VARCHAR(10), hidden_by_admin BOOLEAN NOT NULL DEFAULT false,
  summary_sanitizer_version SMALLINT NOT NULL DEFAULT 0, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ);
CREATE TABLE external_authors (id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), email CITEXT,
  is_claimed BOOLEAN NOT NULL DEFAULT false, user_id UUID REFERENCES users(id),
  do_not_email BOOLEAN NOT NULL DEFAULT false, do_not_import BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ);
CREATE TABLE external_author_names (id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  external_author_id UUID NOT NULL REFERENCES external_authors(id) ON DELETE CASCADE, name VARCHAR(255));
CREATE TABLE external_creatorships (id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  creation_type VARCHAR(20) NOT NULL, creation_id UUID NOT NULL,
  archivist_id UUID REFERENCES users(id), external_author_name_id UUID REFERENCES external_author_names(id));
```

External author emails are personal data of people who never signed up; carry them only to honour `do_not_email` / `do_not_import`, or drop the email and keep the flags.

### 037_collections_and_challenges

```sql
ALTER TABLE collections
  ADD COLUMN legacy_id INTEGER, ADD COLUMN parent_id UUID REFERENCES collections(id),
  ADD COLUMN email VARCHAR(255), ADD COLUMN header_image_url TEXT,
  ADD COLUMN multifandom BOOLEAN, ADD COLUMN open_doors BOOLEAN,
  ADD COLUMN closed BOOLEAN NOT NULL DEFAULT false, ADD COLUMN moderated BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN unrevealed BOOLEAN NOT NULL DEFAULT false, ADD COLUMN anonymous BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN gift_exchange BOOLEAN NOT NULL DEFAULT false, ADD COLUMN prompt_meme BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN show_random BOOLEAN NOT NULL DEFAULT false, ADD COLUMN email_notify BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN description_sanitizer_version SMALLINT NOT NULL DEFAULT 0,
  ADD COLUMN intro TEXT, ADD COLUMN faq TEXT, ADD COLUMN rules TEXT,
  ADD COLUMN gift_notification TEXT, ADD COLUMN assignment_notification TEXT;
-- is_open / is_moderated / is_anonymous are replaced by closed / moderated / anonymous (is_open = NOT closed)

ALTER TABLE collection_items
  ADD COLUMN item_type VARCHAR(10) NOT NULL DEFAULT 'Work' CHECK (item_type IN ('Work','Bookmark')),
  ADD COLUMN bookmark_id UUID REFERENCES bookmarks(id) ON DELETE CASCADE,
  ADD COLUMN user_approval_status SMALLINT NOT NULL DEFAULT 0 CHECK (user_approval_status BETWEEN -1 AND 1),
  ADD COLUMN collection_approval_status SMALLINT NOT NULL DEFAULT 0 CHECK (collection_approval_status BETWEEN -1 AND 1),
  ADD COLUMN anonymous BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN unrevealed BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE collection_items ALTER COLUMN work_id DROP NOT NULL;
-- is_approved is replaced by user_approval_status = 1 AND collection_approval_status = 1

CREATE TABLE collection_participants (
  collection_id UUID NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
  pseud_id UUID NOT NULL REFERENCES pseuds(id) ON DELETE CASCADE,
  participant_role VARCHAR(10) NOT NULL DEFAULT 'None' CHECK (participant_role IN ('None','Owner','Moderator','Member','Invited')),
  created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ, PRIMARY KEY (collection_id, pseud_id));
```

Challenge tables, copied 1:1 from schema.rb with UUID keys and the same column names: `gift_exchanges`, `prompt_memes`, `prompt_restrictions`, `potential_match_settings`, `potential_matches`, `challenge_signups`, `prompts` (with `type` 'Request' or 'Offer'), `challenge_assignments`, `challenge_claims`, `tag_sets`, `owned_tag_sets`, `owned_set_taggings`, `set_taggings`, `tag_set_associations`, `tag_set_nominations`, `tag_set_ownerships`, `tag_nominations`. Sanitizer-version columns for each HTML field (`signup_instructions_*`, `description`).

### 038_skins_files_and_account_extras

```sql
CREATE TABLE skins (id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), legacy_id INTEGER,
  type VARCHAR(10) CHECK (type IN ('site','work')),            -- AO3: NULL = site skin, 'WorkSkin'
  title VARCHAR(255), author_id UUID REFERENCES users(id), css TEXT,
  public BOOLEAN, official BOOLEAN, rejected BOOLEAN, admin_note VARCHAR(255), description TEXT,
  margin INT, paragraph_gap INT, font VARCHAR(255), base_em INT, paragraph_margin REAL,
  background_color VARCHAR(255), foreground_color VARCHAR(255), headercolor VARCHAR(255), accent_color VARCHAR(255),
  role VARCHAR(255), media TEXT[], ie_condition VARCHAR(255), filename VARCHAR(255),
  do_not_upgrade BOOLEAN, cached BOOLEAN, unusable BOOLEAN, featured BOOLEAN, in_chooser BOOLEAN,
  icon_alt_text VARCHAR(255), created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ);
CREATE TABLE skin_parents (child_skin_id UUID REFERENCES skins(id) ON DELETE CASCADE,
  parent_skin_id UUID REFERENCES skins(id), position INT, PRIMARY KEY (child_skin_id, position));
ALTER TABLE works ADD CONSTRAINT works_work_skin_fk FOREIGN KEY (work_skin_id) REFERENCES skins(id);
ALTER TABLE pseuds ADD COLUMN icon_file_id UUID;   -- after choosing a file store (section 7)

-- reading history
CREATE TABLE readings (user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  work_id UUID REFERENCES works(id) ON DELETE CASCADE,
  major_version_read INT, minor_version_read INT, last_viewed TIMESTAMPTZ, view_count INT NOT NULL DEFAULT 0,
  toread BOOLEAN NOT NULL DEFAULT false, toskip BOOLEAN NOT NULL DEFAULT false, created_at TIMESTAMPTZ,
  PRIMARY KEY (user_id, work_id));

-- AO3 preferences with no column today (alternatively extend users.preferences JSONB; typed columns are easier to validate)
CREATE TABLE ao3_preferences (user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  history_enabled BOOLEAN DEFAULT true, email_visible BOOLEAN DEFAULT false, date_of_birth_visible BOOLEAN DEFAULT false,
  adult BOOLEAN DEFAULT false, hide_warnings BOOLEAN, hide_freeform BOOLEAN, view_full_works BOOLEAN,
  disable_work_skins BOOLEAN, skin_id UUID REFERENCES skins(id), minimize_search_engines BOOLEAN, disable_share_links BOOLEAN,
  banner_seen BOOLEAN, first_login BOOLEAN DEFAULT true, preferred_locale TEXT, time_zone VARCHAR(64),
  work_title_format VARCHAR(255) DEFAULT 'TITLE - AUTHOR - FANDOM',
  allow_cocreator BOOLEAN DEFAULT false, allow_gifts BOOLEAN DEFAULT true, allow_collection_invitation BOOLEAN DEFAULT true,
  guest_replies_off BOOLEAN, comment_emails_off BOOLEAN, comment_inbox_off BOOLEAN,
  comment_copy_to_self_off BOOLEAN DEFAULT true, kudos_emails_off BOOLEAN, collection_emails_off BOOLEAN,
  collection_inbox_off BOOLEAN, recipient_emails_off BOOLEAN);
```

The default values shown for `allow_gifts` and `allow_collection_invitation` must be taken from schema.rb:790-823 (not guessed) when this is written for real.

Subscriptions: keep `subscriptions` (`013`), migrate `content_subscriptions` rows into it (or the reverse), and constrain `type` to 'work','series','user' for imported rows. Drop one of the two tables.

### 039_invitations

```sql
CREATE TABLE invitations (id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  creator_id UUID, creator_type VARCHAR(20), invitee_email CITEXT, token TEXT UNIQUE,
  sent_at TIMESTAMPTZ, redeemed_at TIMESTAMPTZ, resent_at TIMESTAMPTZ, used BOOLEAN NOT NULL DEFAULT false,
  invitee_id UUID, invitee_type VARCHAR(20), from_queue BOOLEAN NOT NULL DEFAULT false,
  external_author_id UUID REFERENCES external_authors(id), created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ);
CREATE TABLE invite_requests (id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), email CITEXT,
  simplified_email CITEXT NOT NULL DEFAULT '', ip_address INET, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ);
CREATE TABLE user_invite_requests (id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), user_id UUID REFERENCES users(id),
  quantity INT, reason TEXT, granted BOOLEAN NOT NULL DEFAULT false, handled BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ);
CREATE TABLE banned_emails (email CITEXT PRIMARY KEY, created_at TIMESTAMPTZ);        -- admin_blacklisted_emails
CREATE TABLE fannish_next_of_kins (user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  kin_id UUID REFERENCES users(id), kin_email CITEXT);
CREATE TABLE account_log_items (id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), user_id UUID NOT NULL REFERENCES users(id),
  admin_id UUID REFERENCES users(id), role_id UUID, action SMALLINT, note TEXT, enddate TIMESTAMPTZ,
  fnok_user_id UUID REFERENCES users(id), note_sanitizer_version SMALLINT NOT NULL DEFAULT 0, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ);
```

`invitations.token` is a live credential for unredeemed invites: import only unused rows if the team keeps invitations at all, and treat tokens like passwords. These tables are only needed if the rebuilt archive keeps invite-only signup; AO3 does.

### 040_admin_content

```sql
CREATE TABLE admin_posts (id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), legacy_id INTEGER, admin_id UUID REFERENCES users(id),
  title VARCHAR(255), content TEXT, content_sanitizer_version SMALLINT NOT NULL DEFAULT 0,
  translated_post_id UUID REFERENCES admin_posts(id), language VARCHAR(10),
  comment_permissions SMALLINT NOT NULL DEFAULT 1 CHECK (comment_permissions BETWEEN 0 AND 2),
  moderated_commenting_enabled BOOLEAN NOT NULL DEFAULT false, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ);
CREATE TABLE admin_post_tags (id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), name VARCHAR(255), language VARCHAR(10));
CREATE TABLE admin_post_taggings (admin_post_id UUID REFERENCES admin_posts(id) ON DELETE CASCADE,
  admin_post_tag_id UUID REFERENCES admin_post_tags(id) ON DELETE CASCADE, PRIMARY KEY (admin_post_id, admin_post_tag_id));
CREATE TABLE admin_banners (id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), content TEXT, banner_type VARCHAR(50),
  active BOOLEAN NOT NULL DEFAULT false, content_sanitizer_version SMALLINT NOT NULL DEFAULT 0);
CREATE TABLE known_issues (id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), admin_id UUID, title TEXT, content TEXT,
  content_sanitizer_version SMALLINT NOT NULL DEFAULT 0, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ);
CREATE TABLE archive_faqs (id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), admin_id UUID, title TEXT, slug TEXT NOT NULL DEFAULT '', position INT DEFAULT 1);
CREATE TABLE archive_faq_translations (archive_faq_id UUID, locale VARCHAR(10) NOT NULL, title TEXT);
CREATE TABLE questions (id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), archive_faq_id UUID, question TEXT, content TEXT,
  anchor TEXT, screencast TEXT, position INT DEFAULT 1, content_sanitizer_version SMALLINT NOT NULL DEFAULT 0);
CREATE TABLE question_translations (question_id UUID, locale VARCHAR(10) NOT NULL, question TEXT, content TEXT,
  is_translated TEXT, content_sanitizer_version SMALLINT NOT NULL DEFAULT 0);
CREATE TABLE abuse_reports (id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), email TEXT, url VARCHAR(2080) NOT NULL,
  comment TEXT NOT NULL, summary TEXT, language VARCHAR(10), username TEXT, ip_address INET,
  comment_sanitizer_version SMALLINT NOT NULL DEFAULT 0, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ);
CREATE TABLE feedbacks (id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), email TEXT, summary TEXT, comment TEXT NOT NULL,
  category TEXT, username TEXT, language TEXT, rollout TEXT, user_agent VARCHAR(500),
  comment_sanitizer_version SMALLINT NOT NULL DEFAULT 0, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ);
ALTER TABLE comments ADD CONSTRAINT comments_admin_post_fk FOREIGN KEY (admin_post_id) REFERENCES admin_posts(id) ON DELETE CASCADE;
```

`admin_activities` can load into `moderation_logs` once `moderator_id` allows staff users and `target_type` allows the AO3 target types.

### Import support tables (not AO3 data)

```sql
CREATE TABLE import_progress (table_name TEXT PRIMARY KEY, last_legacy_id BIGINT, rows_read BIGINT, rows_written BIGINT,
  rows_rejected BIGINT, rows_skipped BIGINT, status TEXT, started_at TIMESTAMPTZ, updated_at TIMESTAMPTZ);
CREATE TABLE import_rejects (id BIGSERIAL PRIMARY KEY, table_name TEXT, legacy_id BIGINT, reason TEXT, row JSONB, created_at TIMESTAMPTZ DEFAULT now());
```

### Not part of the schema work, but required by the import

- Constraint relaxations listed above (`pseud_name_format`, `email_format`, `chapter_counts_valid`, `kudos_identity`, `comment_target`, `comment_identity_check`).
- `DEFERRABLE INITIALLY DEFERRED` on FKs used during bulk load, or load order from 5.2.
- Pre-import disable list for triggers (5.3).
- Remove the `020` seed tags and the two `010` prominence rules, or leave them flagged.
- The Go code uses `user_pseudonyms` (15 queries) and `pseuds` (35); merging needs matching changes.

---

## 7. Open questions

Things the schema alone cannot answer, with what I found in otwarchive for each.

1. **Is work text stored outside MySQL?** No. `chapters.content` is a `LONGTEXT` column in MySQL (schema.rb:290-291). Work summary, notes and end notes are columns on `works` and `chapters`. There is no external text store. Downloads (HTML, EPUB, PDF, MOBI) are generated on demand from the database (`app/controllers/downloads_controller.rb`) and cached on disk; those are derived and not imported. Uncertain: the size of the largest chapters and whether any row has been truncated by the `LONGTEXT` limit (4 GB), which is not a practical concern.
2. **S3 attachments.** Icons for pseuds, collections and skins use Active Storage with S3 in production and Staging (`config/environments/production.rb:36`, `staging.rb:73`). Rows are in `active_storage_blobs` (key, filename, content type, byte size, checksum, service name) and `active_storage_attachments` (record type and id). The files themselves are in the bucket and must be copied; the importer needs the bucket name and credentials, which are not in the repository (`config/storage.yml:12-16` is blank). The legacy Paperclip columns `icon_file_name`, `icon_content_type`, `icon_file_size`, `icon_updated_at` remain on `pseuds`, `collections`, `skins` (schema.rb:933, 370, 1064); whether they hold live values or only old ones needs checking against a dump. The rebuild has no file store at all (only `user_pseudonyms.icon_url`).
3. **Elasticsearch-only data.** The search indexes (works, bookmarks, pseuds, tags, stat counters; `app/models/search/*_indexer.rb`) are built from MySQL. I found no data that exists only in Elasticsearch. Nothing to import; the rebuild's search is in Postgres.
4. **Redis-only or Redis-first data.** From code:
   - Hits: `RedisHitCounter` holds a per-day set of (work, IP) pairs and a hash `recent_counts`, flushed to `stat_counters.hit_count` by `HitCountUpdateJob` (`app/models/redis_hit_counter.rb:1-24`, `app/jobs/hit_count_update_job.rb`). The durable total is in MySQL; hits since the last flush are only in Redis. Decide whether to flush before the dump.
   - Reading history: `Reading.update_or_create` pushes new readings onto the Redis set `Reading:new` and a job writes them later (`app/models/reading.rb:15-21`). Unflushed history is only in Redis.
   - Kudos notification queue and subscription notification queue (`app/models/redis_mail_queue.rb`): pending mail; drop.
   - Tag counts (`write_taggings_to_redis`, `app/models/tag.rb:52-59`): derived; recompute.
   - Resque queues: transient jobs; drop. Autocomplete sets: derived.
   Nothing else in Redis is a source of truth that I found; I did not read every file under `lib/` and `app/jobs/`, so a final grep for `REDIS_` on the production code is needed.
5. **Memcached.** Cache only (`config/environments/production.rb:40`). Nothing to import.
6. **Per-day hit history.** AO3 keeps none. The rebuild's `work_hits` (`022`) will be empty for imported works; any "hits over time" view starts at import day.
7. **Date semantics of chapter `published_at`.** AO3 stores a `date`. Choose one: store as `timestamptz` at 00:00 UTC and always render in UTC, or add a `DATE` column. If the front end formats timestamps in the visitor's time zone, a date stored at midnight UTC shows the previous day to visitors west of UTC.
8. **Block and mute semantics.** Read `app/models/block.rb`, `mute.rb` and `app/helpers/mute_helper.rb` for exact effects, then compare with the rebuild's use of both in `can_user_view_work`. The rebuild's function denies viewing; whether AO3 does the same for blocks is not established here.
9. **Which hash schemes exist in the real dump.** Count: `encrypted_password` not starting with `$2`; users with non-empty `password_salt`; bcrypt cost values present. Needs a database read; I have not run it.
10. **Unactivated and deleted accounts.** Whether to import users with `confirmed_at IS NULL`, and what to do with comments, kudos and bookmarks whose user no longer exists (pseud NULL, `user_id` NULL). AO3's own purge policy is in `admin_settings.days_to_purge_unactivated`, which is data, not code.
11. **Staff accounts.** Whether to import `admins` at all or have the new operators create their own. `admins.encrypted_password` is a bcrypt hash too, with a minimum length of 10 (`config/config.yml:123`).
12. **Audit and log history.** `audits`, `log_items`, `admin_activities`. Whether the people running the archive need the history of past moderation. `audits` holds old email addresses and IPs; the default here is to skip it.
13. **Invitations.** Whether the rebuilt archive keeps invite-only sign-up; `admin_settings.creation_requires_invite` controls it in AO3.
14. **Admin settings.** Values in `admin_settings` (account creation, invite queue timing, downloads, hide spam, guest comments, tag wrangling) are operational choices; the rebuild has `system_settings` (key/value). Skipped here; copy any the operators want to keep.
15. **Sanitizer parity.** Whether the rebuild will sanitise on write (new posts) and on import. The AO3 allow-list is in `lib/html_cleaner.rb` and `config/initializers` and the rebuild has no sanitiser that I found.
16. **Tag wrangling model.** Whether the rebuild commits to AO3's canonical, synonym, metatag and parent model (needed for faithful filtering), or something simpler. The import cannot decide this; 034 assumes AO3's model.
17. **Large-object sizes.** `tools/ao3-importer/README.md` claims about 4 million accounts, 10 million works, 50 million chapters, 100 million comments and 5 to 10 TB. These figures come from that README, not from this analysis, and should be replaced by counts from the real dump.
18. **`works.updated_at` for imported rows.** The rebuild triggers touch it. Whether the front end sorts by `updated_at` or by `revised_at` after migration 032 decides which date the "Updated" column should show.
19. **Existing rebuild data.** If the rebuild's database already has users, works or tags, the importer's UUIDs must not collide with those, and usernames or tag names that clash need a rule (AO3's side wins, or the rebuild's does). The plan assumes an empty database except for the seeded rows.
