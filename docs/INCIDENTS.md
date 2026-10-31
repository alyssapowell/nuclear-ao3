# When something serious happens

This is for the big ones: the site is down for more than a few minutes, data may have been seen by someone who shouldn't have it, accounts are being taken over, or someone did something with admin powers that nobody can explain. For everyday problems, use docs/RUNBOOK.md.

You don't need to be an expert to follow this. Do the steps in order. Each one is written so that doing it wrong is better than not doing it.

## In the first hour

### 1. Pick one person to lead

One person, the **incident lead**, decides what happens next until the incident is over. Others help, but the lead decides. If the lead has to leave, they hand over by name ("Sam is lead now"), and the log records it.

### 2. Start the incident log

Copy `docs/incidents/TEMPLATE.md` to a private place everyone helping can edit, such as a private repository, a shared document or a private chat channel. Don't use a public issue tracker: the log may name accounts or attackers.

From now on, write down **everything, with the time**: what was seen, what was done, by whom, and what happened next. "14:02 Alex ran ops-logout-everyone, 4,213 sessions ended." Nobody remembers these details a week later, and they are what the review is built from.

### 3. Tell people

Post a short status update where readers look (the site banner, your status page or social account). Say what's happening in plain words, and when you'll update next. Then update when you said you would, even if nothing has changed.

> We're having an outage. The archive is unreachable. We're working on it and will post again by 15:00 UTC.

> We're investigating a possible security problem. As a precaution, everyone has been logged out. Your works are not affected. We'll post again by 18:00 UTC.

Call it what it is: an outage, or a security incident. Don't call an unplanned outage "maintenance".

### 4. Contain it

Pick the section below that fits. If more than one fits, do them all, starting with security.

## The site is down

1. Run the **ops-diagnose** workflow. It says which part is broken and links to its runbook entry.
2. Follow that entry. If it started right after an upgrade, run **ops-upgrade** with the previous version.
3. If you have to restore from a backup, write down the time of the backup you're restoring and roughly how much was lost. Readers will ask, and the review needs it.

## Data may have been exposed, or the server may be compromised

Assume the worst until you know otherwise. Do these in this order. **Preserve first, then contain**, because containing can destroy the evidence of what happened.

### Preserve

1. **Copy the logs now.** Each service keeps only about 100 MB of logs, so they are overwritten within days. On the server, run `docker-compose logs --timestamps > incident-logs-$(date +%Y%m%d-%H%M).txt`, then copy the file off the server.
2. **Take a backup** with **ops-backup**, and copy it off the server. It shows the state of the data at this moment.
3. If your hosting provider can **snapshot the server's disk**, do it now. It keeps everything, including things you don't yet know matter.
4. Write in the log what you have preserved, and where.

### Contain

Do every step, even ones that seem unrelated. You don't yet know how they got in.

1. **End every login:** run the **ops-logout-everyone** workflow. It also cuts off every OAuth app.
2. **Change the signing key at once:** follow "Change the signing key right now (it may have leaked)" in docs/RUNBOOK.md. Leave the previous-key list empty.
3. **Change the database password:** set a new `POSTGRES_PASSWORD` (generate one: `openssl rand -base64 24`). Change it in Postgres too: `docker-compose exec postgres psql -U ao3_user -d ao3_nuclear -c "ALTER USER ao3_user PASSWORD 'the-new-one'"`. Then restart the app: `docker-compose up -d app`.
4. **Change every other secret** in `.env`: the mail password, the Grafana password, and anything else listed there. Change it with the service it belongs to first (for example, at your mail provider), then in `.env`.
5. **Replace the ops SSH key:** make a new one (see docs/OPS_PIPELINES.md), put it in GitHub's `OPS_SSH_KEY` secret, and delete the old line from `~ops/.ssh/authorized_keys` on the server.
6. **Check who can get in.**
   - On the server, check `~/.ssh/authorized_keys` for every user, and that nobody new is in the docker group.
   - In GitHub, check who has access to the repository and to the `production-approval` environment.
7. **Check who holds admin powers:** run `docker-compose exec app nuclear roles`. It lists everyone who holds the admin, moderator or tag_wrangler role. Anyone you don't recognise is a finding: write it in the log, then have a trusted admin take the role away (`DELETE /api/v1/auth/admin/users/<id>/roles/<role>`). If you are unsure whom to trust, run `docker-compose exec app nuclear ban <username> --reason "..."` to stop the account from posting and log it out.
8. **Review what admins have done:** run `docker-compose exec app nuclear audit --since 2026-10-01 --limit 200` (use a date before the incident began). Every hide, spam mark, suspension, ban and role change is listed with who did it, when, and the reason they gave. Anything you can't explain is a finding. Actions taken with `nuclear` on the server show as `cli:<server user>`. The log can't be edited from the site, but someone with full database access could change it, so compare against your backups if you think that happened.
9. If the server itself may be compromised (unknown processes, unknown users, files you didn't change), **don't trust it.** Build a new server, restore the latest backup that predates the incident, and set it up with all the new secrets above.

### Who to tell

- Readers whose data may have been exposed: tell them what was exposed, when, and what they should do, once you know.
- Depending on where you and your users are, the law may set a deadline for telling a data-protection authority (in the EU and UK it's 72 hours from when you became aware). Ask your organisation's legal contact on day one, not day three.

## Something harmful was posted

1. Hide it at once: `docker-compose exec app nuclear hide work <id> --reason "..."` (or `chapter`, `comment`, `series`, `bookmark`). It's gone for everyone the app answers, immediately.
2. **If a CDN is in front, purge it there too.** A CDN may keep serving its copy to logged-out visitors for up to a minute, plus up to five more while it refreshes. For anything that must disappear now (illegal material, private information, content involving minors), use the CDN's "purge by URL" (in Cloudflare: Caching, Configuration, Purge Cache, Custom Purge). For a work with id W, purge `https://YOUR-DOMAIN/api/v1/work/W` and `/api/v1/works/W`, and their `/chapters` and `/comments` addresses. If a search or tag page showed its title or summary, purge that address too (the query string is part of the address). For a hidden comment, purge its work's `/comments` address. Logged-in readers never get the CDN's copy, so they see the hide at once.
3. If you couldn't purge, write that in the incident log: the content may have stayed visible to logged-out visitors for up to six minutes.

## Accounts are being taken over

1. For each affected account: run **ops-logout-user**, then ask the owner to use "Forgot password". The reset ends every session and disconnects OAuth apps.
2. If it's many accounts at once, treat it as "Data may have been exposed" above.

## Afterwards: the review

Within two weeks, while people still remember, the incident lead fills in the rest of the log template: what happened, why it was possible, and what changes so it can't happen again.

- **It's about the system, not about people.** "Why could one person's mistake take the site down?" is a useful question. "Who made the mistake?" is not. People only tell you what really happened if they won't be blamed for it.
- **Keep asking "why?"** until the answer is something you can change. "The disk filled up" isn't a root cause. "Nothing stopped the logs growing, and nothing warned us before the disk was full" is.
- **Prefer guardrails to instructions.** For each action, ask: can the software or the pipeline do this automatically, or refuse to run without it? An instruction depends on someone remembering it, possibly years from now. A guardrail doesn't.
- **Every action gets an owner and a date.** An action nobody owns doesn't happen.

Publish a version for readers when it's safe to: what happened, what it meant for them, and what's changing. Leave out details that would help someone do it again until they're fixed.
