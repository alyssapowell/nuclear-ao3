# Backups

This page explains what is backed up, where the copies go, how to set up off-server backups, and how to get everything back.

## What is backed up

Every day the server saves the database (all works, users, comments, tags and so on) to a file in `backups/`, named like `nuclear-20261030-024712.sql.gz`. The newest 14 are kept on the server.

If off-server backups are set up, each new file is also copied to a storage location run by another company. Restic, the backup tool, encrypts the data before it leaves the server. The storage company only ever sees scrambled data. Old copies there are thinned out automatically: the last 14 days, 8 weeks and 12 months are kept.

## What is not backed up

- `secrets/` is never copied off the server by these scripts. It holds the login signing key, the off-server backup password and your storage login. Backups that include their own key would be readable by anyone who finds the backup. Keep your own copies of these files somewhere safe, separate from the server (a password manager works well).
- `.env` (your settings) is not backed up. Keep a copy too.
- Anything else on the server's disk, such as logs.

If the server's backup runs but off-server backup is not set up, the daily summary says `WARNING` in plain words: your backups exist only on this server. If the server or its disk is lost, they are lost too.

## Choosing where to store the copies

Pick a company that is different from the one hosting your server. If both are at the same company, one billing problem or account suspension can take away both.

Options that work:

- Any S3-compatible bucket (Amazon S3, Wasabi, Cloudflare R2, DigitalOcean Spaces, and many others).
- Backblaze B2.
- A restic REST server.
- An SFTP box. This needs an SSH key and known-hosts file inside the backup container, which these scripts do not set up. Prefer S3 or B2 if you are unsure.

Buckets are cheap. A database backup is usually small, and about 35 copies are kept.

## Setting it up

You need a shell on the server, in the folder with `docker-compose.yml`.

1. Create a bucket (or equivalent) at your chosen provider, and create an access key that can read and write only that bucket.
2. Put the access key in a file inside `secrets/`, for example `secrets/offsite.env`, one `KEY=VALUE` per line with no quotes. For S3:

   ```text
   AWS_ACCESS_KEY_ID=your-key-id
   AWS_SECRET_ACCESS_KEY=your-secret
   ```

   For Backblaze B2 use `B2_ACCOUNT_ID` and `B2_ACCOUNT_KEY`. Then run `chmod 600 secrets/offsite.env`.
3. Add these lines to `.env`:

   ```text
   OFFSITE_REPOSITORY=s3:https://s3.example.com/my-bucket/nuclear
   OFFSITE_ENV_FILE=secrets/offsite.env
   ```

   Other address forms: `b2:my-bucket:nuclear`, `rest:https://host:8000/nuclear`, `sftp:user@host:/path`. `OFFSITE_PASSWORD_FILE` is optional and defaults to `secrets/offsite-password`. All three paths must be inside `secrets/`. The address itself must not contain a login (`https://user:password@...`): it would show up in logs and run summaries, so the scripts refuse it. Logins always go in the `OFFSITE_ENV_FILE` file.
4. Run `make offsite-init`. This makes a random password in `secrets/offsite-password` (readable only by its owner) and creates the encrypted repository. It will not overwrite an existing password file.
5. Save the password now. Copy the contents of `secrets/offsite-password` into your password manager. It is not stored anywhere else. Without it nobody can read the off-server backups, including you, and the storage company cannot help.
6. Run `make ops-backup`. The summary should show "Off-server copy" with a snapshot id. Then run `make offsite-list` to see it.

From then on every backup, including the daily scheduled one, is copied off-server. Once a week the backup also reads back a small random 1% sample of the stored data to check it is intact.

## When the off-server copy fails

The local backup is still saved, but the backup run is marked as failed (red in the GitHub Actions tab) so you notice. The summary says why. Common causes are a wrong or expired access key, a full or deleted bucket, or a network problem. Fix the cause and run the backup again.

## Listing what is stored off-server

`make offsite-list` on the server, or run the "ops-offsite-list" workflow from the Actions tab. Fetching and setup are done by hand on the server only.

## Restoring onto a fresh server from off-server alone

This is what you do if the old server is gone.

1. Set up the new server as described in the hosting guide and start it.
2. Recreate `.env` from your own copy. Recreate the secrets files from your own copies, including the signing key, `secrets/offsite.env` and `secrets/offsite-password` (the password you saved in step 5 above). Check the files are readable only by their owner (`chmod 600`).
3. Run `make offsite-list` to see the available backups.
4. Run `make offsite-fetch ID=latest`, or give a snapshot id from the list. The file appears in `backups/` under its original name, after being checked.
5. Run `make ops-restore FILE=nuclear-YYYYmmdd-HHMMSS.sql.gz` with that file name. It asks you to type a confirmation, then replaces the database with the backup. Everything saved after that backup was made is lost.
6. Check the site loads and recent works are there.

Without the signing key from your own copy, everyone has to log in again but no data is lost.

## Monthly restore drill

A backup you have never restored is only a hope. Once a month:

1. Run `make offsite-list` and confirm the newest snapshot is less than a day old.
2. Run `make offsite-fetch ID=latest` on the server (move away today's local file with the same name first, since fetch will not replace an existing file), or do it on a spare computer or test server.
3. Restore it into a test copy of the site, never the live one, with `make ops-restore`.
4. Open the site and check recent works and users are present.
5. Write down the date and the result.

If any step fails, that is the thing to fix this week.
