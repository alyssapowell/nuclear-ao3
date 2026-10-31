#!/usr/bin/env bash
# Looks at everything that usually goes wrong and says what it found. Changes
# nothing. Exits 1 if it found a problem.
set -euo pipefail
# shellcheck source=lib.sh
. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

echo "# Diagnose"

# --- containers
h2 "Containers"
states=""
for svc in postgres redis app caddy smtp; do
	id=$(compose ps -q "$svc" 2>/dev/null | head -n 1 || true)
	if [ -z "$id" ]; then
		state="not created"
	else
		state=$(docker inspect -f '{{.State.Status}}{{if .State.Health}} ({{.State.Health.Status}}){{end}}' "$id" 2>/dev/null || echo unknown)
	fi
	states="$states$svc: $state"$'\n'
	case "$state" in
	running | "running (healthy)") ;;
	*) problem "Container \`$svc\` is $state." "the-site-isnt-responding" ;;
	esac
done
printf '%s' "$states" | fence

# --- health
h2 "App health"
if body=$(curl -fsS -m 10 "$APP_URL/health" 2>&1); then
	printf '%s\n' "$body" | fence
else
	printf 'No answer from %s/health\n%s\n' "$APP_URL" "$body" | fence
	problem "The app does not answer on /health." "the-site-isnt-responding"
fi

# --- migrations
h2 "Database migrations"
if out=$(run_nuclear migrate status 2>&1); then
	printf '%s\n' "$out" | fence
	case "$out" in
	*"up to date"*) ;;
	*) problem "Some migrations have not been applied." "the-app-wont-start-migration-nnn-failed-nothing-from-it-was-applied" ;;
	esac
else
	printf '%s\n' "$out" | fence
	problem "Could not read the migration status (database down, or a migration problem)." "the-app-wont-start-migration-nnn-failed-nothing-from-it-was-applied"
fi

# --- jobs
h2 "Background jobs (emails)"
if out=$(run_nuclear jobs 2>&1); then
	printf '%s\n' "$out" | fence
	read -r waiting failed <<<"$(printf '%s\n' "$out" | awk '$2 ~ /^[0-9]+$/ { w += $2; f += $4 } END { print w + 0, f + 0 }')"
	if [ "$failed" -gt 0 ]; then problem "$failed background job(s) gave up (usually emails)." "emails-arent-arriving"; fi
	if [ "$waiting" -gt 100 ]; then problem "$waiting background jobs are waiting." "background-jobs-are-piling-up"; fi
else
	printf '%s\n' "$out" | fence
	problem "Could not read the background jobs." "background-jobs-are-piling-up"
fi

# --- disk
h2 "Disk space"
disk_paths="$OPS_DIR"
if [ -d /var/lib/docker ]; then disk_paths="$disk_paths /var/lib/docker"; fi
for p in $disk_paths; do
	line=$(df -Pk "$p" | awk 'NR == 2 { print $5, $4, $6 }')
	used=${line%% *}
	pct=${used%\%}
	free_gb=$(echo "$line" | awk '{ printf "%.1f", $2 / 1048576 }')
	bullet "\`$p\`: $used used, $free_gb GB free"
	if [ "$pct" -ge 90 ]; then
		problem "The disk holding $p is $used full." "the-disk-is-nearly-full"
	fi
done
echo

# --- backup
h2 "Last backup"
newest=$(ls -1t "$BACKUPS_DIR"/nuclear-*.sql.gz 2>/dev/null | head -n 1 || true)
if [ -z "$newest" ]; then
	para "No backup found in \`backups/\`."
	problem "There is no backup yet." "back-up-the-database"
else
	age_h=$(( ($(date +%s) - $(file_mtime "$newest")) / 3600 ))
	para "\`$(basename "$newest")\`, $age_h hour(s) old."
	if [ "$age_h" -gt 36 ]; then
		problem "The newest backup is $age_h hours old." "back-up-the-database"
	fi
fi

# --- signing key
h2 "Login signing key"
if out=$(run_nuclear_nodb keys show 2>&1); then
	printf '%s\n' "$out" | fence
else
	printf '%s\n' "$out" | fence
	problem "The login signing key does not load." "set-up-the-signing-key"
fi

report_problems
