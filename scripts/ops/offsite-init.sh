#!/usr/bin/env bash
# Sets up off-server backups once: makes a random repository password in
# secrets/ and creates the encrypted repository. Run by hand on the server
# after filling in the OFFSITE_* settings in .env. Refuses to overwrite an
# existing password file.
set -euo pipefail
# shellcheck source=offsite-lib.sh
. "$(dirname "${BASH_SOURCE[0]}")/offsite-lib.sh"

rc=0
offsite_settings init || rc=$?
[ "$rc" -ne 2 ] || die "set OFFSITE_REPOSITORY (and OFFSITE_ENV_FILE) in .env first; see docs/BACKUPS.md"
[ "$rc" -eq 0 ] || die "$OFFSITE_ERR"

if [ -e "$OFFSITE_PASSWORD_HOST" ]; then
	die "$OFFSITE_PASSWORD_HOST already exists. Not overwriting it: that password may be the only key to existing backups. If this repository is already set up, you do not need to run init again."
fi

mkdir -p "$(dirname "$OFFSITE_PASSWORD_HOST")"
( umask 077; head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n' >"$OFFSITE_PASSWORD_HOST" )
chmod 600 "$OFFSITE_PASSWORD_HOST"

log "creating the repository"
rc=0
out=$(restic init 2>&1) || rc=$?
if [ "$rc" -ne 0 ]; then
	# Nothing was created with this password, so it is safe to discard.
	rm -f "$OFFSITE_PASSWORD_HOST"
	echo "# Off-server backup setup"
	h2 "Result: PROBLEMS"
	bullet "restic could not create the repository (exit code $rc). The new password file was removed again; fix the problem and re-run."
	printf '%s\n' "$out" | scrub | fence
	exit 1
fi

echo "# Off-server backup setup"
h2 "Result: OK"
bullet "Created the encrypted repository."
bullet "Saved the repository password in \`$OFFSITE_PASSWORD_HOST\` (readable only by its owner)."
h2 "Do this now: save the password somewhere else"
para "Copy the contents of \`$OFFSITE_PASSWORD_HOST\` into your password manager, or print it and keep it safe, **right now**. Show it with: \`cat $OFFSITE_PASSWORD_HOST\`"
para "Everything off-server is encrypted with this password. It is not stored anywhere else. Without it nobody can read these backups, including you, and the storage company cannot help. If this server is lost and the password was only on it, the off-server backups are lost too."
