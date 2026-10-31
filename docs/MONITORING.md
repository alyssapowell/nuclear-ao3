# Monitoring

Monitoring is already set up when you start the archive with Docker Compose. Three small programs do the work:

- **The app** counts what it does (visitors, errors, how long pages take, background jobs, database connections) and shows the counts on a private port, 9091. That port is never reachable from the internet.
- **Prometheus** reads those counts every 15 seconds, keeps 30 days of them, and checks the alert rules in `monitoring/alerts.yml`.
- **Grafana** draws the counts as charts. The charts, the connection to Prometheus and the alert list all come from files in `monitoring/grafana/`, so there is nothing to set up by hand.

A fourth program, **node-exporter**, reports how full the server's disk is.

## Before the first start

Set a Grafana password in `.env`. Docker Compose will refuse to start until you do.

```
openssl rand -base64 24
```

Paste the result after `GRAFANA_ADMIN_PASSWORD=` in `.env`, and keep it in your password manager. The user name is `admin`.

Grafana stores the password the first time it starts. If you change `GRAFANA_ADMIN_PASSWORD` later, Grafana ignores it. Change it on the Grafana profile page, or run `docker compose exec grafana grafana cli admin reset-admin-password <new password>`.

## Open Grafana

Grafana listens only on the server itself (127.0.0.1, port 3002), so strangers can't reach it. To see it from your own computer, make a tunnel:

```
ssh -L 3002:localhost:3002 your-user@your-server
```

Leave that window open, then go to http://localhost:3002 in your browser and log in as `admin`. Open **Dashboards**, then the **Nuclear AO3** folder, then **Archive overview**.

If you run the archive on your own computer, skip the tunnel and go straight to http://localhost:3002.

## The Archive overview dashboard

Each section has a note at the top saying what bad looks like and what to do. Charts are read-only: changing them in the browser does nothing, because Grafana puts them back from the files. To change a chart, edit `monitoring/grafana/dashboards/archive-overview.json`.

**Is anything wrong right now?** Two boxes (App and Database) that should say UP in green, and a table of alerts that are firing. An empty table is good.

**Visitors and errors.** Requests per second, the share of requests that failed on the server's side, how long the slowest 5% of requests take, and which pages fail the most. Bad: errors above 5%, or the slow line above 1 second, for many minutes.

**Background jobs.** Emails and other work happen in the background. Jobs waiting, jobs that gave up, and how long the longest-waiting job has waited. Bad: "gave up" above zero, or a job waiting more than 15 minutes.

**Database and server.** Database connections in use, app memory, goroutines (small tasks inside the app) and free disk space. Bad: connections in use sit at the "allowed" line, memory or goroutines climb for days without ever dropping, or free disk falls under 10%.

## The alerts

Alerts are checked by Prometheus. When one fires, it shows in the "Alerts firing now" table and on Grafana's **Alerting** page. Each alert carries the name of the runbook entry to follow, in `docs/RUNBOOK.md`.

| Alert | What it means | What to do |
| --- | --- | --- |
| AppDown | Prometheus couldn't reach the app for 2 minutes. The site is probably down. | Run the **Diagnose** workflow. Runbook: "The site is down". |
| DatabaseDown | The app is running but can't reach the database. | Same as above. Runbook: "The site is down". |
| HighErrorRate | More than 5% of requests failed with a server error for 5 minutes. | Run **Diagnose**, look at "Pages with the most errors". Runbook: "The site shows errors or is slow". |
| SlowResponses | One request in 20 took over 1 second, for 10 minutes. | Same entry as above. |
| JobsFailing | A background job gave up in the last 15 minutes. | Runbook: "Background jobs are piling up". For emails, use the **Emails** workflow or see "Emails aren't arriving". |
| JobsBacklog | A background job has been waiting over 15 minutes. | Same as JobsFailing. |
| DiskAlmostFull | The server's disk has under 10% free. A full disk stops the database. | Runbook: "The disk is almost full". |

JobsFailing goes quiet 15 minutes after the last new failure, even if failed jobs are still listed. Check the "Jobs that gave up" chart, or run `make jobs`, until it reads zero.

## How alerts reach someone

Out of the box, they don't. Prometheus decides an alert is firing, and Grafana shows it, but nothing sends an email or a message. Someone has to look at the dashboard. Do this at least once a day, or set up email below.

Because of that, also use an outside check on the public site, such as a free uptime service that pings your archive's address. It will tell you about an outage even when the whole server is off, which no monitoring on the server itself can.

### Get alerts by email (optional)

You need a mail account that can send (the same kind of settings the archive uses for its own emails).

1. Copy the example file:
   `cp monitoring/grafana/provisioning/alerting/notifications.yml.example monitoring/grafana/provisioning/alerting/notifications.yml`
2. In `.env`, remove the `#` from the lines `ALERT_EMAIL_TO`, `GF_SMTP_ENABLED`, `GF_SMTP_HOST`, `GF_SMTP_USER`, `GF_SMTP_PASSWORD` and `GF_SMTP_FROM_ADDRESS`, and fill them in. `ALERT_EMAIL_TO` can hold several addresses separated by semicolons.
3. Restart Grafana: `docker compose up -d grafana`.
4. Test it: in Grafana go to **Alerting**, then **Contact points**, choose `archive-email`, and press **Test**.

The email names the alert. Find that name in the table above to see what to do. The email is sent at most once every four hours for the same alert while it keeps firing.

To turn it off, delete `notifications.yml` and restart Grafana.

## If a chart says "No data"

- Everything empty: Prometheus can't reach the app. Open http://localhost:9090/targets on the server (or through a tunnel to port 9090). Each target should say UP.
- Only job charts empty: no background jobs have ever run. That is normal on a brand-new archive.
- Disk chart empty: node-exporter isn't running. Check `docker compose ps`.
