# Runbook

What to do when something goes wrong, or when you need to do a routine job. Each section is written so you can follow it without knowing the code.

Most problems below can be handled by pressing **Run workflow** on the GitHub Actions tab and reading the summary; the one-time setup is in [OPS_PIPELINES.md](OPS_PIPELINES.md). Each entry also gives the `make ops-...` command to run on the server instead, and then the manual steps.

Commands assume you're in the repo folder on the server, using Docker Compose. `nuclear` is the app's program; inside Compose you run it as `docker-compose run --rm app nuclear ...`, or use the `make` shortcut shown.

---

## Routine jobs

### Set up a brand-new archive

1. Set up the signing key (next section). The app won't start without it.
2. `docker-compose up -d`. The app creates the database tables itself the first time it starts.
3. Register your own account on the site like any user.
4. Make yourself an admin: `make grant USER=yourname ROLE=admin`.

There are no built-in accounts or default passwords.

### Set up the signing key

To check afterwards that the key loads, run the **ops-diagnose** workflow (Actions tab -> ops-diagnose -> Run workflow), or `make ops-diagnose` on the server. It reports a key that can't be read. The key itself has to be made on the server, as below.

The signing key is what proves a login is real. You make it once, and you keep it. It lives in a file in `secrets/` and is never put in the database or the backups.

1. `make keys`. It creates `secrets/signing-key-DATE.pem` and prints a key ID.
2. **Save a copy of that file outside this server**: a password manager or offline storage. If the server is lost and you have no copy, the only fix is a new key, and everyone has to log in again.
3. In `.env`, set `SIGNING_KEY_FILE=/secrets/signing-key-DATE.pem` (use the file name from step 1). On Linux also set `APP_UID` to the number printed by `id -u`.
4. `docker-compose up -d app`.
5. Check it: `make keys-show` prints the key ID. It should match the one from step 1.

If the app says "SIGNING_KEY_FILE is not set" or can't read the key, one of these steps is missing. There is no automatic fallback.

### Change the signing key (planned)

Do this now and then, or when someone who held a copy of the key leaves. People stay logged in.

1. `make keys`. Save a copy of the new file outside this server.
2. In `.env`, change `SIGNING_KEY_FILE` to the new file, and put the old file in `SIGNING_KEY_PREVIOUS_FILES`, for example `SIGNING_KEY_PREVIOUS_FILES=/secrets/signing-key-OLDDATE.pem`.
3. `docker-compose up -d app`.
4. `make keys-show` should list the new key as current and the old one as previous.
5. **After 14 days**, take the old file out of `SIGNING_KEY_PREVIOUS_FILES` and run `docker-compose up -d app` again. By then every login made with it has expired.

### Change the signing key right now (it may have leaked)

Anyone with the key can make logins for any account, so don't wait. The key has to be replaced on the server by hand (below). Afterwards run the **ops-diagnose** workflow (Actions tab -> ops-diagnose -> Run workflow; `make ops-diagnose` on the server) to confirm the new key loads, and the **ops-logout-everyone** workflow if you also want OAuth tokens and unused authorization codes ended.

1. `make keys`. Save a copy of the new file outside this server.
2. In `.env`, change `SIGNING_KEY_FILE` to the new file, and make `SIGNING_KEY_PREVIOUS_FILES` empty. Don't list the old key.
3. `docker-compose up -d app`.
4. Every login ends at once and everyone has to log in again. Apps connected through OAuth fetch the new public key from `/auth/jwks` by themselves; if one keeps failing, ask its owner to refresh it.
5. Destroy the old key file and every copy of it, including old backups of the `secrets/` folder and the copy in your password manager.

### Back up the database

Run the **ops-backup** workflow (Actions tab -> ops-backup -> Run workflow). If GitHub isn't set up or reachable, run `make ops-backup` on the server. Either way you get a short summary of what was found and done. The manual steps follow.

It saves `backups/nuclear-DATE-TIME.sql.gz`, checks that the file is complete, and keeps the newest 14. A backup also runs by itself every day if you kept the schedule in the workflow. The backup holds the **database only**: it does not contain `secrets/` (the signing key) or `.env`. Copy those separately.

Manual steps: `make backup-db`. Copy `backups/` to somewhere that isn't this server now and then.

### Restore the database from a backup

This **replaces the whole database**. Everything saved after the backup was made is lost.

Run the **ops-restore** workflow (Actions tab -> ops-restore -> Run workflow). Type the backup file name (the backup workflow's summary shows it; the file is in `backups/`) and type `replace the database` to confirm. A person who is set as a reviewer must approve it. If GitHub isn't set up, run `make ops-restore FILE=nuclear-DATE-TIME.sql.gz` on the server; it asks you to type the same words. The manual steps follow.

It takes a safety backup of the current database first, stops the app, loads the backup (all of it or nothing), starts the app and checks health.

Manual steps: there is no safe hand-made version of this. Use `make ops-restore` on the server; it is the same script the workflow runs.

If the server is gone and the backup is only off-server, fetch it first: see "Restoring onto a fresh server from off-server alone" in docs/BACKUPS.md.

### Log everyone out

Use this after a password leak, or when you aren't sure who is logged in. Run the **ops-logout-everyone** workflow (Actions tab -> ops-logout-everyone -> Run workflow; `make ops-logout-everyone` on the server). Or the **ops-logout-user** workflow (`make ops-logout-user USER=name`) for one account. It ends all login sessions, OAuth tokens, unused authorization codes and app permissions, so people and connected apps have to log in again. The summary shows the counts. These need a reviewer's approval on GitHub.

Manual steps: `docker-compose run --rm app nuclear sessions revoke-all`, or `... nuclear sessions revoke <username>`.

### Give someone a role

`make grant USER=theirname ROLE=tag_wrangler` (or `moderator`, or `admin`). They need to have registered first. Running it again for a role that was revoked restores it.

### Upgrade to a new version

Run the **ops-upgrade** workflow (Actions tab -> ops-upgrade -> Run workflow) and type the version (a tag such as `v1.4.0`, or a branch). If GitHub isn't set up or reachable, run `make ops-upgrade REF=v1.4.0` on the server. Either way you get a short summary of what was found and done. The manual steps follow.

It backs up, switches versions, rebuilds, waits for the app to be healthy and runs the smoke test. If anything fails it goes back to the previous version by itself, and tells you whether database migrations had run. It never restores the database on its own.

The manual way:

1. **Back up first:** `make backup-db`.
2. Get the new version and rebuild: `git pull && docker-compose build app`.
3. `docker-compose up -d app`.
4. Check it works: `scripts/smoke-test.sh https://your.archive`. It creates two throwaway accounts and a work, and prints PASS or FAIL for posting, privacy, publishing, comments, kudos, search and notifications.

On startup the app applies any new database migrations, one at a time. If you run several copies of the app, only one migrates; the others wait. If a migration fails, the app doesn't start, nothing from that migration is applied, and the log says which file failed (see below).

To see what an upgrade will change before starting it: `make migrate-status`.

### Load sample data on your own computer

`make seed` adds four sample accounts (`testuser`, `author2`, `tagwrangler`, `admin`, all with password `password123`) and two works. **Never run this on a real archive.**

---

## When something is wrong

### The app won't start: "migration NNN failed, nothing from it was applied"

Run the **ops-diagnose** workflow (Actions tab -> ops-diagnose -> Run workflow) to see the error and the state of every part. If GitHub isn't set up or reachable, run `make ops-diagnose` on the server. Either way you get a short summary of what was found and done. The manual steps follow.

If this happened during an upgrade, the **ops-upgrade** workflow has already put the previous version back and says so in its summary.

The database is exactly as it was before that migration, so nothing is half-done.

1. Read the error after the colon. It names the problem, such as a duplicate value or a missing table.
2. If the problem is in your data (for example, two rows that a new rule says must be unique), fix the data, then start the app again.
3. If you can't tell, roll back to the previous version of the app (`git checkout <previous tag>`, rebuild, start). The older version runs fine because the failed migration was never applied. Then report the error.

### The app won't start: "Database was created before migrations were tracked"

Run the **ops-diagnose** workflow (Actions tab -> ops-diagnose -> Run workflow) to confirm what is wrong. If GitHub isn't set up or reachable, run `make ops-diagnose` on the server. Either way you get a short summary of what was found and done. The manual steps follow.

If the archive has real data, run the **ops-backup** workflow (`make ops-backup`) before anything else.

The full message says the database "already has tables but no schema_migrations record". It means this database was set up by an older version of Nuclear AO3, which loaded the SQL files through Postgres's first-start scripts. Those stopped at the first error, so there's no way to know which parts were applied. The app refuses to guess, because guessing could run a file twice or skip one.

If the archive has **no real data yet** (a test or a fresh install):

1. `docker-compose down`
2. `docker volume rm nuclear-ao3_postgres_data` (this deletes the database)
3. `docker-compose up -d`. The app builds a clean database.

If it **has real data**:

1. Back up: `make backup-db`.
2. Don't delete anything. Ask for help with the backup file in hand: the data has to be moved into a freshly built database, and that needs someone who can compare the two schemas.

### Notifications aren't arriving

Run the **ops-diagnose** workflow (Actions tab -> ops-diagnose -> Run workflow) and read the Containers, App health and Background jobs parts. If GitHub isn't set up or reachable, run `make ops-diagnose` on the server. Either way you get a short summary of what was found and done. The manual steps follow.

1. In-app notifications: check that the app log has no `PANIC while processing` lines.
2. Email: see "Emails aren't arriving" below.

### Emails aren't arriving

Run the **ops-emails** workflow (Actions tab -> ops-emails -> Run workflow) (tick **retry** to send the failed ones again once the mail server works). If GitHub isn't set up or reachable, run `make ops-emails (or make ops-emails RETRY=1)` on the server. Either way you get a short summary of what was found and done. The manual steps follow.

It shows the failed jobs and why, checks that the app can reach the mail server in `.env`, and only retries if it can.

Every email (welcome emails, notification emails) is a background job. If the mail server is down, the app keeps the email and tries again: after 30 seconds, then 1 minute, 2, 4, and so on, for about four hours. Nothing is lost while it waits.

1. See what's happening: `make jobs`. It shows how many emails are waiting and failed, and why the latest ones failed.
2. The reason is usually the mail server settings: `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD` in `.env`. Fix them and restart the app: `docker compose up -d app`. Waiting emails go out on their next try.
3. Emails that gave up (after about four hours) stay listed as failed. Once mail works again, send them all with `make jobs-retry`, or one with `make jobs-retry ID=<id>`.
4. If links in emails point to the wrong address, set `SITE_URL` in `.env` to the archive's public address, such as `https://archive.example.org`.

### Background jobs are piling up

Run the **ops-diagnose** workflow (Actions tab -> ops-diagnose -> Run workflow) to see the counts and the latest failures. If GitHub isn't set up or reachable, run `make ops-diagnose` on the server. Either way you get a short summary of what was found and done. The manual steps follow.

`make jobs` shows a large WAITING count and an old OLDEST WAITING time.

- If the failures show the same error over and over, fix that first (for email, see above). Retrying doesn't help until the cause is fixed.
- If jobs are succeeding but slowly, run more at once: set `JOB_WORKERS` in `.env` (default 2) and restart the app.
- A job that was running when the app stopped is picked up again after 10 minutes. You don't need to do anything.

### The site isn't responding

The alerts "AppDown" and "DatabaseDown" lead here.

Run the **ops-diagnose** workflow (Actions tab -> ops-diagnose -> Run workflow) to see which container is down or what the app answers on `/health`. If GitHub isn't set up or reachable, run `make ops-diagnose` on the server. Either way you get a short summary of what was found and done. The manual steps follow.

Manual steps:

1. `docker-compose ps` shows which containers are running.
2. `docker-compose logs --tail 100 app` shows why the app stopped.
3. `docker-compose up -d` starts anything that is down.
4. If it started failing right after an upgrade, see "The app won't start" above.

### The disk is nearly full

The alert "DiskAlmostFull" leads here. A full disk stops the database, and the site goes down with it.

Run the **ops-diagnose** workflow (Actions tab -> ops-diagnose -> Run workflow) to see how full it is. If GitHub isn't set up or reachable, run `make ops-diagnose` on the server. Either way you get a short summary of what was found and done. The manual steps follow.

Manual steps: `df -h` shows which disk. Old backups in `backups/` are removed automatically (the newest 14 are kept). `docker system prune -f` removes unused Docker leftovers. Copy backups elsewhere and delete the oldest if you need space.

### The site shows errors or is slow

The alerts "HighErrorRate" and "SlowResponses" lead here.

1. On the Archive overview dashboard (see docs/MONITORING.md), look at "Pages with the most errors" to see which part of the site is failing.
2. Run the **ops-diagnose** workflow and read the app's recent log lines for the same pages (`docker-compose logs --tail 200 app`).
3. Check "Database connections". If "in use" stays at "allowed", the database is the bottleneck: look for a long-running query, or restart the app with `docker-compose restart app`.
4. Check "App memory" and the disk. If the server is out of either, free some or move to a bigger server.
5. If it began right after an upgrade, run the **ops-upgrade** workflow with the previous version to go back.

### Off-server backup problems

The daily **ops-backup** run went red, or **ops-diagnose** reported a problem with the off-server backup. The backup on the server is still saved; the off-server copy didn't happen. Open the run's summary: it shows restic's own message.

- "access denied" or "unable to open repository": the storage login in `secrets/offsite.env` is wrong, expired or missing permissions, or `OFFSITE_REPOSITORY` is mistyped.
- "wrong password" or "no key could be found": `secrets/offsite-password` isn't the password the repository was created with. Put back your saved copy. **Never run `make offsite-init` again for a repository that already exists.**
- Network or timeout errors: usually temporary. Run **ops-backup** again from the Actions tab.
- "the weekly integrity check found a problem": the stored backups may be damaged. Set up a new repository (a new bucket or path, then `make offsite-init`), take a fresh backup, and keep the old repository until you're sure.
- "Off-server backup is not set up": follow docs/BACKUPS.md.

Fix the cause, then run **ops-backup** again. Until it's green, the newest backup exists only on this server.

If the off-server password is lost and the server is gone, nobody can read those backups, including you. Set up a new repository and start again. This is why the password has to be saved somewhere other than the server.

### Search doesn't find a work

Search reads the works table directly. There's no separate index that can fall behind, so there's nothing to rebuild. If a work doesn't show up, it's being left out on purpose. Check, in order:

1. **Is it posted?** Drafts never appear in search.
2. **Is it restricted to logged-in users?** Then it only appears for readers who are logged in.
3. **Has the reader blocked or muted the creator, or has the creator blocked the reader?** Then it's hidden from that reader.
4. **Is it anonymous?** Then searching by creator won't find it (on purpose), but searching its title will.
5. **Is it in a collection that hasn't revealed its works yet?** Then it stays out of search until the reveal.

To see exactly what search returns: `curl "https://your.archive/api/v1/search/works?q=some+words" | jq '.total, [.works[].title]'`.
