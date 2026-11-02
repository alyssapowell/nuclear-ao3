# Operations from GitHub

This lets a person press **Run workflow** on the GitHub Actions tab to check the archive, back it up, upgrade it, restore it, or log people out, and read a short summary. Nobody has to log in to the server and type commands.

You set it up once, with the steps below. After that, day-to-day use is in [RUNBOOK.md](RUNBOOK.md).

## What you get

| Workflow | What it does | Environment |
| --- | --- | --- |
| ops-diagnose | Checks containers, health, migrations, background jobs, disk space, last backup and the signing key. Changes nothing. Runs every day. | production |
| ops-emails | Shows why emails aren't arriving. Tick **retry** to send failed ones again (only if the mail server can be reached). | production |
| ops-backup | Saves the database to `backups/`. Keeps the newest 14. Runs every day. | production |
| ops-upgrade | Backs up, moves to a version you name, rebuilds, checks health, runs the smoke test. Goes back to the old version if it fails. | production |
| ops-restore | Replaces the whole database with a backup. | production-approval |
| ops-logout-everyone | Ends every login. | production-approval |
| ops-logout-user | Ends one account's logins. | production-approval |

Every workflow ends with a summary on its run page. A red run means a problem or a failed step; the summary says which.

The same operations work on the server without GitHub: `make ops-diagnose`, `make ops-emails`, `make ops-backup`, `make ops-upgrade REF=v1.0.0`, `make ops-restore FILE=nuclear-...sql.gz`, `make ops-logout-everyone`, `make ops-logout-user USER=name`. The workflows and the make commands run the same scripts in `scripts/ops/`.

## How it is kept safe

GitHub connects to the server with its own SSH key. On the server, that key is tied to one program, `scripts/ops/dispatch.sh`. Whatever the key's owner asks for, the server runs that program and nothing else. It accepts only these requests, written exactly like this:

```
diagnose
emails [--retry]
backup
upgrade <version>
restore <backup file name> --i-understand-this-replaces-the-database
logout-everyone
logout-user <email or account id>
```

Each value is checked against a strict pattern: a version may contain only letters, digits, `.`, `_`, `/` and `-`, may not start with `-` and may not contain `..`; an account is an email address or an account id (letters, digits and `. _ + - @` only); a backup file must be named like `nuclear-20260101-120000.sql.gz` and is only looked for in `backups/`. Anything else is refused with "not allowed". The text is never run through a shell.

**What the key can do:** run those seven operations, including replacing the database and upgrading to any version in the repository. Treat it like an admin password.

**What the key cannot do:** open a shell, run any other command, copy files, forward ports, or read `.env` or `secrets/`.

Restore and the two logouts use a separate GitHub environment, `production-approval`, where you can require another person to approve each run. The key is the same in both environments, so the approval is a safeguard against mistakes, not against someone who gets hold of the key itself.

## One-time setup

You need: admin access to the GitHub repository, and a login with `sudo` on the server. The server already runs the archive with Docker Compose, with the repository checked out and a `.env` file in it.

### 1. Create the `ops` user on the server

These commands assume the repository is at `/opt/nuclear-ao3`. If yours is elsewhere, use that path everywhere below.

```bash
sudo adduser --disabled-password --gecos "" ops
sudo usermod -aG docker ops
sudo chown -R ops:ops /opt/nuclear-ao3
```

Being in the `docker` group is what lets `ops` run the archive, and it is powerful: anyone who controls the `ops` account can control the whole server. That is why its SSH key is limited to the program above.

Check that git can fetch as `ops`, which the upgrade needs:

```bash
sudo -u ops git -C /opt/nuclear-ao3 fetch --tags origin
```

If the repository is private, give `ops` a read-only deploy key for it (GitHub: repository Settings -> Deploy keys) or use a public clone address.

The scripts need `docker`, `git`, `curl`, `python3` (for the smoke test) and `gzip` on the server.

### 2. Make the SSH key

On your own computer (not the server):

```bash
ssh-keygen -t ed25519 -f nuclear-ops-key -C "github-ops" -N ""
```

This makes two files: `nuclear-ops-key` (private, goes to GitHub) and `nuclear-ops-key.pub` (public, goes to the server). Do not put the private file anywhere but GitHub's secret box and, if you want, a password manager.

### 3. Let the key in, tied to the program

On the server, as `ops`:

```bash
sudo -u ops mkdir -p /home/ops/.ssh
sudo -u ops chmod 700 /home/ops/.ssh
sudo -u ops nano /home/ops/.ssh/authorized_keys
```

Add one line. Copy the contents of `nuclear-ops-key.pub` and put this in front of it:

```
command="/opt/nuclear-ao3/scripts/ops/dispatch.sh",restrict ssh-ed25519 AAAA...rest-of-the-public-key... github-ops
```

Then:

```bash
sudo -u ops chmod 600 /home/ops/.ssh/authorized_keys
sudo chmod +x /opt/nuclear-ao3/scripts/ops/*.sh
```

`command=` makes the server run only the program. `restrict` turns off everything else (terminals, forwarding, and so on).

Test it from your computer. Both of these should work as described:

```bash
ssh -i nuclear-ops-key ops@YOUR-SERVER diagnose        # prints a summary
ssh -i nuclear-ops-key ops@YOUR-SERVER "ls /"          # prints: not allowed
```

### 4. Pin the server's host key

This makes GitHub refuse to talk to anything that is not your server.

From your computer:

```bash
ssh-keyscan -t ed25519 YOUR-SERVER > nuclear-known-hosts
ssh-keygen -lf nuclear-known-hosts
```

The second command prints a fingerprint such as `SHA256:abc...`. On the server, print its real one and compare them:

```bash
ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub
```

They must be identical. If they are not, stop: something is between you and the server. Use the contents of `nuclear-known-hosts` for the `OPS_KNOWN_HOSTS` secret.

### 5. Add the four secrets in GitHub

Repository -> Settings -> Secrets and variables -> Actions -> **New repository secret**:

| Name | Value |
| --- | --- |
| `OPS_SSH_KEY` | The whole contents of `nuclear-ops-key`, including the BEGIN and END lines |
| `OPS_HOST` | The server's address (name or IP) |
| `OPS_USER` | `ops` |
| `OPS_KNOWN_HOSTS` | The contents of `nuclear-known-hosts` from step 4 |

Once the secrets are saved, delete the files from your computer (`rm nuclear-ops-key nuclear-ops-key.pub nuclear-known-hosts`). Keep a copy of the private key in a password manager only if you want to be able to re-add it later; otherwise you can simply make a new key.

### 6. Create the two environments

Repository -> Settings -> Environments -> **New environment**:

1. `production`. No other settings are needed.
2. `production-approval`. Tick **Required reviewers** and add the people who may approve a restore or a logout. If you are the only admin, add yourself; you will still have to click Approve, which gives you a moment to read what you are about to do.

Only run the workflows from the default branch. In each environment you can also set **Deployment branches** to the default branch only.

### 7. Try it

Actions tab -> **ops-diagnose** -> **Run workflow**. After a minute, open the run and read the summary. If it says OK, the setup works. The first run of **ops-backup** creates the `backups/` folder.

If a run fails before any summary appears, open the step's log. The usual causes are a wrong secret (the log says `Permission denied` or `Host key verification failed`) or `ops` missing from the `docker` group.

## Revoking the key

To cut off GitHub's access immediately, delete its line from `/home/ops/.ssh/authorized_keys` on the server. Nothing else needs to change. To start again, repeat steps 2, 3 and 5 with a new key.

Do this if the private key may have leaked, or when someone with access to the repository's secrets leaves.

## Daily runs

`ops-diagnose` runs at 03:17 UTC and `ops-backup` at 02:47 UTC. To change the times, edit the `cron:` line in the workflow file. When a scheduled run fails, GitHub emails the person who last changed the schedule; check Settings -> Notifications there is turned on. Scheduled workflows pause after 60 days without any repository activity.

Backups are kept on the same server. Copy `backups/` somewhere else now and then, along with `.env` and `secrets/`, which the backups do not contain.

## Things to know

- Only one operation runs at a time. If you start a second while one is running, it waits. GitHub keeps only the newest waiting run.
- **ops-upgrade** refuses to run if someone changed files in the repository folder on the server. Clean them up (`git status` shows them), then run it again.
- **ops-upgrade** never restores the database by itself. If a failed upgrade had already applied database migrations, the summary says so, names the backup, and leaves the choice of restoring (with **ops-restore**) to a person.
- **ops-upgrade** runs the smoke test in read-only mode (`SMOKE_READ_ONLY=1`): it checks the site, the works list, search, logins and the login keys, and changes nothing on the live archive. The full smoke test, which creates accounts and works and reads emails from MailHog, is for development machines and test servers.
- Each run's summary can be read by anyone who can see the repository's Actions tab. It contains counts, file names and error lines, not passwords.

## What the end-to-end CI job proves

Every pull request runs the `end-to-end` job in `.github/workflows/ci.yml`. It is the first place where the whole Docker Compose stack and the ops scripts run for real, on a fresh machine, with nothing carried over. It fails if any step fails; nothing in it ignores an error.

It starts the stack with `docker compose up -d --build`, waits (with a deadline) until the app, Caddy, Prometheus, Grafana and MailHog answer, and then checks:

- The full smoke test against the app, and the read-only smoke test through Caddy.
- Grafana has the `prometheus` datasource and the "Archive overview" dashboard, Prometheus scrapes the app and node-exporter successfully, every alert in `monitoring/alerts.yml` is loaded, and `http_requests_total` has data. The app's metrics port is not reachable from outside Docker.
- `make ops-backup` makes a readable backup. `make ops-diagnose` exits 1 with exactly one problem, the missing off-server backup (CI has none, so that is the expected result; any other problem fails the job).
- A restore round trip: an account is created through the API, backed up, deleted from the database, brought back with `make ops-restore`, and the app is healthy afterwards.
- `make jobs` prints the jobs table.

For CI only, the job sets `ENVIRONMENT=development`, `DOMAIN=:80` (so Caddy serves plain http) and creates a throwaway signing key and Grafana password. It does not prove anything about real HTTPS certificates, real email delivery, or off-server backups. If the job fails, its last step prints `docker compose ps`, the last 200 log lines of each service and the names of the files in `backups/` (never `.env` or `secrets/`).
