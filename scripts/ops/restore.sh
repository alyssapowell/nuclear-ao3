#!/usr/bin/env bash
# usage: restore.sh <backup-file> --i-understand-this-replaces-the-database
# <backup-file> is a name like nuclear-20260101-120000.sql.gz in backups/.
# Replaces the whole database with the backup. Takes a safety backup first.
set -euo pipefail
# shellcheck source=lib.sh
. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

[ "${2:-}" = "$RESTORE_FLAG" ] || die "this replaces the whole database. Add $RESTORE_FLAG to confirm. usage: restore.sh <backup-file> $RESTORE_FLAG"
arg="${1:-}"
name=$(basename "$arg")
valid_backup_name "$name" || die "not a backup file name: $arg"
file="$BACKUPS_DIR/$name"
[ -f "$file" ] || die "no such backup: backups/$name"
[ -s "$file" ] || die "backup is empty: backups/$name"
gunzip -t "$file" || die "backup is damaged: backups/$name"

echo "# Restore"
log "safety backup first"
safety_out=$("$(dirname "${BASH_SOURCE[0]}")/backup.sh") || die "the safety backup failed; nothing was changed"
safety=$(printf '%s\n' "$safety_out" | grep -o 'nuclear-[0-9]\{8\}-[0-9]\{6\}\.sql\.gz' | head -n 1)

log "stopping the app"
compose stop app >&2
restart_app() { compose up -d app >&2 || true; }

# One transaction: either the whole backup loads or nothing changes.
log "restoring"
rc=0
{
	echo 'DROP SCHEMA public CASCADE; CREATE SCHEMA public;'
	gunzip -c "$file"
} | compose exec -T postgres sh -c 'PGPASSWORD="$POSTGRES_PASSWORD" psql -q -1 -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -o /dev/null' >&2 || rc=$?

log "starting the app"
restart_app
if [ "$rc" -ne 0 ]; then
	h2 "Result: PROBLEMS"
	bullet "Restoring \`$name\` **failed**. Nothing was changed: the database is as it was before. The app was started again."
	bullet "A safety backup was taken first: \`$safety\`."
	para "Read the job log for the Postgres error. A human decides what to do next."
	exit 1
fi

h2 "Checks"
ok=1
if wait_healthy 120; then bullet "App is healthy again."; else bullet "App is **not** healthy after the restore."; ok=0; fi
status=$(run_nuclear migrate status 2>&1 | tail -n 5 || true)
printf '%s\n' "$status" | fence
users=$(compose exec -T postgres sh -c 'PGPASSWORD="$POSTGRES_PASSWORD" psql -At -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT count(*) FROM users"' </dev/null 2>/dev/null || echo "?")
bullet "Accounts in the restored database: $users"

if [ "$ok" -eq 1 ]; then
	h2 "Result: OK"
	bullet "Restored \`$name\`. Safety backup of the earlier state: \`$safety\`."
	para "Everything saved after that backup was made is gone. Logins keep working only if \`secrets/\` still holds the same signing key."
else
	h2 "Result: PROBLEMS"
	bullet "Restored \`$name\`, but the app is not healthy. Run the diagnose workflow. Safety backup: \`$safety\`."
	exit 1
fi
