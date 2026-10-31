#!/usr/bin/env bash
# Usage: offsite-push.sh <backup-file>
# Copies one backup file to the off-server restic repository (encrypted by
# restic), thins out old snapshots, and once a week reads back a small random
# sample of the repository to prove it is intact. Prints a Markdown summary;
# exits non-zero if anything failed. backup.sh calls this when
# OFFSITE_REPOSITORY is set.
set -euo pipefail
# shellcheck source=offsite-lib.sh
. "$(dirname "${BASH_SOURCE[0]}")/offsite-lib.sh"

file="${1:-}"
[ -n "$file" ] || die "usage: offsite-push.sh <backup-file>"
[ -f "$file" ] || die "no such file: $file"
base=$(basename "$file")
valid_backup_name "$base" || die "not a backup file name: $base"
case "$(cd "$(dirname "$file")" && pwd)" in
"$BACKUPS_DIR") ;;
*) die "the file must be inside backups/" ;;
esac

h2 "Off-server copy"

fail() { # what-failed, output
	bullet "**FAILED:** $1"
	if [ -n "${2:-}" ]; then printf '%s\n' "$2" | scrub | fence; fi
	para "The local backup \`$base\` is saved on this server, but it is **not** safely stored off-server."
	exit 1
}

rc=0
offsite_settings || rc=$?
[ "$rc" -eq 0 ] || fail "${OFFSITE_ERR:-OFFSITE_REPOSITORY is not set.}" ""

log "pushing $base off-server"
rc=0
out=$(restic backup --tag "$OFFSITE_TAG" --host "$OFFSITE_HOST" "/backups/$base" 2>&1) || rc=$?
[ "$rc" -eq 0 ] || fail "restic could not copy \`$base\` (exit code $rc)." "$out"
snap=$(printf '%s\n' "$out" | sed -n -E 's/^snapshot ([0-9a-f]+) saved.*/\1/p' | tail -n 1)
[ -n "$snap" ] || snap="(id not shown)"
bullet "Pushed \`$base\` with tag \`$OFFSITE_TAG\`. Snapshot id: \`$snap\`."

log "removing old off-server snapshots"
rc=0
out=$(restic forget --tag "$OFFSITE_TAG" --host "$OFFSITE_HOST" --keep-daily 14 --keep-weekly 8 --keep-monthly 12 --prune 2>&1) || rc=$?
[ "$rc" -eq 0 ] || fail "the new copy was saved, but clearing out old snapshots failed (exit code $rc)." "$out"
bullet "Old snapshots thinned: keeping 14 daily, 8 weekly and 12 monthly."

now=$(date +%s)
last=0
[ -f "$OFFSITE_CHECK_STATE" ] && last=$(tr -dc '0-9' <"$OFFSITE_CHECK_STATE" || true)
[ -n "$last" ] || last=0
if [ $((now - last)) -ge "$OFFSITE_CHECK_EVERY" ]; then
	log "weekly integrity check of 1% of the off-server data"
	rc=0
	out=$(restic check --read-data-subset=1% 2>&1) || rc=$?
	[ "$rc" -eq 0 ] || fail "the weekly integrity check found a problem (exit code $rc). The repository may be damaged." "$out"
	printf '%s\n' "$now" >"$OFFSITE_CHECK_STATE"
	bullet "Weekly integrity check passed (read back a 1% sample)."
else
	bullet "Weekly integrity check not due yet."
fi
echo
