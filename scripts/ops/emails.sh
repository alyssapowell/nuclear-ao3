#!/usr/bin/env bash
# usage: emails.sh [--retry]
# Shows why emails might not be arriving. With --retry, and only if the mail
# server can be reached, sends the failed ones again.
set -euo pipefail
# shellcheck source=lib.sh
. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

retry=0
case "${1:-}" in
"") ;;
--retry) retry=1 ;;
*) die "usage: emails.sh [--retry]" ;;
esac

echo "# Emails"

h2 "Background jobs"
if ! out=$(run_nuclear jobs 2>&1); then
	printf '%s\n' "$out" | fence
	problem "Could not read the background jobs." "emails-arent-arriving"
	report_problems || exit 1
fi
printf '%s\n' "$out" | fence
read -r waiting failed <<<"$(jobs_totals)"

h2 "Mail server"
host=$(env_get SMTP_HOST smtp)
port=$(env_get SMTP_PORT 1025)
# Shown as text only; these two values come from .env, so check their shape
# before they go anywhere near a command.
if ! [[ "$host" =~ ^[A-Za-z0-9._-]+$ ]] || ! [[ "$port" =~ ^[0-9]{1,5}$ ]]; then
	para "SMTP_HOST or SMTP_PORT in \`.env\` has an odd value, so it was not tested."
	problem "SMTP_HOST / SMTP_PORT in .env are not a host name and a port number." "emails-arent-arriving"
	report_problems || exit 1
fi
smtp_ok=0
if compose run --rm -T --no-deps --entrypoint sh app -c 'nc -w 5 "$0" "$1" </dev/null >/dev/null 2>&1' "$host" "$port" </dev/null; then
	smtp_ok=1
	para "The app container can reach \`$host:$port\`."
else
	para "The app container **cannot** reach \`$host:$port\`."
	problem "The mail server $host:$port cannot be reached from the app. Check SMTP_HOST and SMTP_PORT in .env, and that the mail server is running." "emails-arent-arriving"
fi

h2 "Retry"
if [ "$failed" -eq 0 ]; then
	para "No failed jobs, nothing to send again. $waiting job(s) are waiting and will go out on their own."
elif [ "$retry" -eq 0 ]; then
	para "$failed failed job(s). Run this again with **retry** ticked to send them again, once the cause is fixed."
elif [ "$smtp_ok" -eq 0 ]; then
	para "Not retrying: the mail server can't be reached, so they would only fail again. Fix the settings in \`.env\`, run \`docker compose up -d app\`, then retry."
else
	res=$(run_nuclear jobs retry 2>&1) || { printf '%s\n' "$res" | fence; problem "Retrying failed." "emails-arent-arriving"; report_problems || exit 1; }
	printf '%s\n' "$res" | fence
	log "waiting 20 seconds for the retried jobs to run"
	sleep 20
	read -r waiting2 failed2 <<<"$(jobs_totals)"
	para "After 20 seconds: $waiting2 waiting, $failed2 failed."
	if [ "$failed2" -gt 0 ]; then
		problem "$failed2 job(s) failed again. Read the reasons above." "emails-arent-arriving"
	fi
fi

report_problems
