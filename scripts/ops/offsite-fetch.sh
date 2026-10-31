#!/usr/bin/env bash
# Usage: offsite-fetch.sh <snapshot-id|latest>
# Brings one backup back from the off-server repository into backups/, under
# its original name, ready for restore.sh. Run it by hand on the server. It
# never replaces an existing file and never touches the database.
set -euo pipefail
# shellcheck source=offsite-lib.sh
. "$(dirname "${BASH_SOURCE[0]}")/offsite-lib.sh"

id="${1:-}"
re='^([A-Za-z0-9]{4,64}|latest)$'
[[ "$id" =~ $re ]] || die "usage: offsite-fetch.sh <snapshot-id|latest>  (see 'make offsite-list')"

rc=0
offsite_settings || rc=$?
[ "$rc" -ne 2 ] || die "off-server backup is not set up (OFFSITE_REPOSITORY is empty in .env)"
[ "$rc" -eq 0 ] || die "$OFFSITE_ERR"

mkdir -p "$BACKUPS_DIR"
chmod 700 "$BACKUPS_DIR"
echo "# Fetch from off-server"

filters=()
if [ "$id" = "latest" ]; then filters=(--tag "$OFFSITE_TAG" --host "$OFFSITE_HOST"); fi

log "looking inside snapshot $id"
rc=0
if [ "${#filters[@]}" -gt 0 ]; then
	listing=$(restic ls "$id" "${filters[@]}" 2>&1) || rc=$?
else
	listing=$(restic ls "$id" 2>&1) || rc=$?
fi
if [ "$rc" -ne 0 ]; then
	h2 "Result: PROBLEMS"
	bullet "restic could not read snapshot \`$id\` (exit code $rc)."
	printf '%s\n' "$listing" | scrub | fence
	exit 1
fi
path=$(printf '%s\n' "$listing" | grep -E '^/backups/nuclear-[0-9]{8}-[0-9]{6}\.sql\.gz$' | head -n 1 || true)
[ -n "$path" ] || die "snapshot $id does not contain a nuclear-*.sql.gz backup file"
name=$(basename "$path")
valid_backup_name "$name" || die "unexpected file name in snapshot: $name"
[ ! -e "$BACKUPS_DIR/$name" ] || die "backups/$name already exists here; move it away first if you really want to replace it"

tmp="$BACKUPS_DIR/.$name.fetching"
trap 'rm -f "$tmp"' EXIT
log "downloading $name"
rc=0
umask 077
if [ "${#filters[@]}" -gt 0 ]; then
	restic dump "$id" "$path" "${filters[@]}" >"$tmp" 2>"$tmp.err" || rc=$?
else
	restic dump "$id" "$path" >"$tmp" 2>"$tmp.err" || rc=$?
fi
errtext=$(tail -n 20 "$tmp.err" 2>/dev/null || true)
rm -f "$tmp.err"
if [ "$rc" -ne 0 ]; then
	h2 "Result: PROBLEMS"
	bullet "restic could not download \`$name\` (exit code $rc)."
	printf '%s\n' "$errtext" | scrub | fence
	exit 1
fi
[ -s "$tmp" ] || die "the downloaded file is empty"
gunzip -t "$tmp" || die "the downloaded file is not a valid gzip file"
gunzip -c "$tmp" | tail -n 5 | grep -q "PostgreSQL database dump complete" || die "the downloaded backup looks cut off"
chmod 600 "$tmp"
mv "$tmp" "$BACKUPS_DIR/$name"
trap - EXIT

h2 "Result: OK"
bullet "Restored \`backups/$name\` from snapshot \`$id\`. Checked: not empty, readable, complete."
para "Load it into the database with the restore operation (docs/BACKUPS.md, \"Restoring onto a fresh server\")."
