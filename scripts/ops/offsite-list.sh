#!/usr/bin/env bash
# Usage: offsite-list.sh [--newest-age-hours]
# Lists the snapshots in the off-server repository as a Markdown summary.
# With --newest-age-hours it prints only a whole number: how many hours old
# the newest snapshot is (used by diagnose). It exits 2 if off-server backup
# is not set up, 1 if the repository can't be read.
set -euo pipefail
# shellcheck source=offsite-lib.sh
. "$(dirname "${BASH_SOURCE[0]}")/offsite-lib.sh"

mode="${1:-}"
case "$mode" in "" | --newest-age-hours) ;; *) die "usage: offsite-list.sh [--newest-age-hours]" ;; esac

# to_epoch "YYYY-mm-dd HH:MM:SS": restic prints UTC inside the container.
to_epoch() {
	date -u -d "$1 UTC" +%s 2>/dev/null || date -u -j -f "%Y-%m-%d %H:%M:%S" "$1" +%s
}

rc=0
offsite_settings || rc=$?
if [ "$rc" -ne 0 ]; then
	[ "$mode" = "" ] || exit "$rc"
	echo "# Off-server snapshots"
	h2 "Result: PROBLEMS"
	if [ "$rc" -eq 2 ]; then
		bullet "Off-server backup is not set up (OFFSITE_REPOSITORY is empty). See docs/BACKUPS.md."
	else
		bullet "$OFFSITE_ERR"
	fi
	exit 1
fi

if [ "$mode" = "--newest-age-hours" ]; then
	out=$(restic snapshots --tag "$OFFSITE_TAG" --host "$OFFSITE_HOST" --latest 1 2>/dev/null) || exit 1
	newest=0
	while IFS= read -r line; do
		d=$(printf '%s' "$line" | sed -n -E 's/^[0-9a-f]{8} +([0-9]{4}-[0-9]{2}-[0-9]{2} [0-9:]{8}).*/\1/p')
		[ -n "$d" ] || continue
		e=$(to_epoch "$d") || continue
		[ "$e" -gt "$newest" ] && newest=$e
	done <<<"$out"
	[ "$newest" -gt 0 ] || exit 1
	echo $((($(date +%s) - newest) / 3600))
	exit 0
fi

echo "# Off-server snapshots"
rc=0
out=$(restic snapshots --tag "$OFFSITE_TAG" --host "$OFFSITE_HOST" 2>&1) || rc=$?
if [ "$rc" -ne 0 ]; then
	h2 "Result: PROBLEMS"
	bullet "restic could not read the repository (exit code $rc)."
	printf '%s\n' "$out" | scrub | fence
	exit 1
fi
h2 "Snapshots"
printf '%s\n' "$out" | scrub | fence
h2 "Result: OK"
para "To bring one back to this server: \`make offsite-fetch ID=<id>\` (or \`ID=latest\`), then restore it as described in docs/BACKUPS.md."
