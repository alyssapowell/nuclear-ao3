#!/usr/bin/env bash
# Saves the database to backups/nuclear-YYYYmmdd-HHMMSS.sql.gz and keeps the
# newest 14. The database login is read by the postgres container from its
# own settings (which come from .env); nothing is written in this script.
#
# The backup holds the DATABASE ONLY. It does NOT contain secrets/ (the login
# signing key) or .env. Keep your own copy of those somewhere safe.
set -euo pipefail
# shellcheck source=lib.sh
. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

mkdir -p "$BACKUPS_DIR"
chmod 700 "$BACKUPS_DIR"
name="nuclear-$(date +%Y%m%d-%H%M%S).sql.gz"
tmp="$BACKUPS_DIR/.$name.partial"
final="$BACKUPS_DIR/$name"
trap 'rm -f "$tmp"' EXIT

echo "# Backup"
log "dumping the database"
# sh -c runs inside the postgres container, where these variables are set.
compose exec -T postgres sh -c 'PGPASSWORD="$POSTGRES_PASSWORD" pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" --no-owner' \
	</dev/null | gzip -c >"$tmp"

[ -s "$tmp" ] || die "the backup file is empty"
gunzip -t "$tmp" || die "the backup file is not a valid gzip file"
# A real dump ends with this line; a cut-off one doesn't.
gunzip -c "$tmp" | tail -n 5 | grep -q "PostgreSQL database dump complete" || die "the backup looks cut off"
chmod 600 "$tmp"
mv "$tmp" "$final"
trap - EXIT

# Keep the newest $KEEP_BACKUPS.
removed=0
for old in $(ls -1t "$BACKUPS_DIR"/nuclear-*.sql.gz | tail -n +$((KEEP_BACKUPS + 1))); do
	rm -f "$old"
	removed=$((removed + 1))
done
total=$(ls -1 "$BACKUPS_DIR"/nuclear-*.sql.gz | wc -l | tr -d ' ')
size=$(du -h "$final" | cut -f 1)

h2 "Result: OK"
bullet "Saved \`backups/$name\` ($size). Checked: not empty, readable, complete."
bullet "$total backup(s) kept (newest $KEEP_BACKUPS); $removed old one(s) removed."
h2 "Not included"
para "The backup has the database only. It does **not** contain \`secrets/\` (the login signing key) or \`.env\`. Keep separate copies of those; see the signing key section of the runbook. Backups also live on this same server, so copy them somewhere else now and then."
