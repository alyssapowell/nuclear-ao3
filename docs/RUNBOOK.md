# Runbook

What to do when something goes wrong, or when you need to do a routine job. Each section is written so you can follow it without knowing the code.

Commands assume you're in the repo folder on the server, using Docker Compose. `nuclear` is the app's program; inside Compose you run it as `docker-compose run --rm app nuclear ...`, or use the `make` shortcut shown.

---

## Routine jobs

### Set up a brand-new archive

1. `docker-compose up -d`. The app creates the database tables itself the first time it starts.
2. Register your own account on the site like any user.
3. Make yourself an admin: `make grant USER=yourname ROLE=admin`.

There are no built-in accounts or default passwords.

### Give someone a role

`make grant USER=theirname ROLE=tag_wrangler` (or `moderator`, or `admin`). They need to have registered first. Running it again for a role that was revoked restores it.

### Upgrade to a new version

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

The database is exactly as it was before that migration, so nothing is half-done.

1. Read the error after the colon. It names the problem, such as a duplicate value or a missing table.
2. If the problem is in your data (for example, two rows that a new rule says must be unique), fix the data, then start the app again.
3. If you can't tell, roll back to the previous version of the app (`git checkout <previous tag>`, rebuild, start). The older version runs fine because the failed migration was never applied. Then report the error.

### The app won't start: "Database was created before migrations were tracked"

The full message says the database "already has tables but no schema_migrations record". It means this database was set up by an older version of Nuclear AO3, which loaded the SQL files through Postgres's first-start scripts. Those stopped at the first error, so there's no way to know which parts were applied. The app refuses to guess, because guessing could run a file twice or skip one.

If the archive has **no real data yet** (a test or a fresh install):

1. `docker-compose down`
2. `docker volume rm nuclear-ao3_postgres_data` (this deletes the database)
3. `docker-compose up -d`. The app builds a clean database.

If it **has real data**:

1. Back up: `make backup-db`.
2. Don't delete anything. Ask for help with the backup file in hand: the data has to be moved into a freshly built database, and that needs someone who can compare the two schemas.

### Notifications aren't arriving

1. In-app notifications: check that the app log has no `PANIC while processing` lines.
2. Email: the log says `Failed to send` with the reason. Usually the mail server settings (`SMTP_HOST`, `SMTP_PORT`) are wrong.

### Search doesn't find a work

Search reads the works table directly. There's no separate index that can fall behind, so there's nothing to rebuild. If a work doesn't show up, it's being left out on purpose. Check, in order:

1. **Is it posted?** Drafts never appear in search.
2. **Is it restricted to logged-in users?** Then it only appears for readers who are logged in.
3. **Has the reader blocked or muted the creator, or has the creator blocked the reader?** Then it's hidden from that reader.
4. **Is it anonymous?** Then searching by creator won't find it (on purpose), but searching its title will.
5. **Is it in a collection that hasn't revealed its works yet?** Then it stays out of search until the reveal.

To see exactly what search returns: `curl "https://your.archive/api/v1/search/works?q=some+words" | jq '.total, [.works[].title]'`.
