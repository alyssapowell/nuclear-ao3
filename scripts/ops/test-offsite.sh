#!/usr/bin/env bash
# Tests backup.sh and the offsite-*.sh scripts without Docker or a real
# repository. A fake "docker" and a fake "compose" record their arguments.
# Run: bash scripts/ops/test-offsite.sh
set -euo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT=$(mktemp -d)
trap 'rm -rf "$ROOT"' EXIT

FAKE_DOCKER="$ROOT/fake-docker"
FAKE_COMPOSE="$ROOT/fake-compose"
cat >"$FAKE_DOCKER" <<'FAKE_EOF'
#!/usr/bin/env bash
# Records every call. FAKE_FAIL_ON=<restic subcommand> makes that one fail.
printf '%s\n' "$*" >>"$FAKE_LOG"
printf 'RESTIC_REPOSITORY=%s\n' "${RESTIC_REPOSITORY:-}" >>"$FAKE_LOG.env"
sub=""; found=0
for a in "$@"; do
	if [ "$found" -eq 1 ]; then sub="$a"; break; fi
	case "$a" in restic/restic:*) found=1 ;; esac
done
if [ "${FAKE_FAIL_ON:-}" = "$sub" ]; then
	echo "Fatal: unable to open repository at s3:https://AKIAUSER:hunter2@host/b: access denied" >&2
	exit 1
fi
case "$sub" in
backup) echo "snapshot abcd1234 saved" ;;
forget) echo "removed 0 snapshots" ;;
check) echo "no errors were found" ;;
init) echo "created restic repository" ;;
snapshots)
	echo "ID        Time                 Host         Tags        Paths"
	echo "-----------------------------------------------------------"
	echo "${FAKE_SNAP_ROW:-abcd1234  2026-10-30 02:47:12  nuclear-ao3  nuclear-db  /backups/nuclear-20261030-024712.sql.gz}"
	;;
ls)
	echo "snapshot abcd1234 of [/backups/nuclear-20261030-024712.sql.gz] at 2026-10-30:"
	echo "/backups"
	echo "/backups/nuclear-20261030-024712.sql.gz"
	;;
dump)
	printf -- '-- PostgreSQL database dump complete\n' | gzip -c
	;;
esac
exit 0
FAKE_EOF
cat >"$FAKE_COMPOSE" <<'FAKE_EOF'
#!/usr/bin/env bash
printf 'compose %s\n' "$*" >>"$FAKE_LOG"
printf -- '-- dump\n-- PostgreSQL database dump complete\n'
FAKE_EOF
chmod +x "$FAKE_DOCKER" "$FAKE_COMPOSE"

FAILS=0
ok() { echo "PASS  $1"; }
bad() { echo "FAIL  $1"; FAILS=$((FAILS + 1)); }
# check DESCRIPTION CMD...: passes if the command succeeds.
check() { local d="$1"; shift; if "$@"; then ok "$d"; else bad "$d"; fi; }
has() { grep -qF -- "$2" "$1"; }
lacks() { ! grep -qF -- "$2" "$1"; }

# new_site: a fresh OPS_DIR with .env, secrets and a fake log.
SECRET_PW="correct-horse-battery-staple-0123456789"
SECRET_KEY="AKIAEXAMPLEKEYVALUE99"
SECRET_SK="wJalrSECRETACCESSKEYVALUE42"
new_site() { # configured? yes|no
	SITE=$(mktemp -d "$ROOT/site.XXXXXX")
	mkdir -p "$SITE/secrets"
	: >"$SITE/.env"
	if [ "$1" = "yes" ]; then
		printf 'OFFSITE_REPOSITORY=s3:https://s3.example.com/bucket/nuclear\nOFFSITE_ENV_FILE=secrets/offsite.env\n' >"$SITE/.env"
		printf 'AWS_ACCESS_KEY_ID=%s\nAWS_SECRET_ACCESS_KEY=%s\n' "$SECRET_KEY" "$SECRET_SK" >"$SITE/secrets/offsite.env"
		printf '%s' "$SECRET_PW" >"$SITE/secrets/offsite-password"
	fi
	export FAKE_LOG="$SITE/calls.log"
	: >"$FAKE_LOG"; : >"$FAKE_LOG.env"
	export OPS_DIR="$SITE" OPS_COMPOSE="$FAKE_COMPOSE" OPS_DOCKER="$FAKE_DOCKER"
	unset FAKE_FAIL_ON FAKE_SNAP_ROW || true
}
run() { # script args...; sets OUT and RC
	RC=0
	OUT=$("$HERE/$1" "${@:2}" 2>"$SITE/stderr.log") || RC=$?
	printf '%s\n' "$OUT" >"$SITE/out.md"
}

echo "== backup.sh without off-server settings"
new_site no
run backup.sh
check "backup succeeds" [ "$RC" -eq 0 ]
check "warning is printed" has "$SITE/out.md" "WARNING: off-server backup is not set up"
check "restic is never run" lacks "$SITE/calls.log" "restic/restic"
check "local backup file exists" test -n "$(ls "$SITE"/backups/nuclear-*.sql.gz 2>/dev/null)"

echo "== backup.sh with off-server settings"
new_site yes
run backup.sh
check "backup succeeds" [ "$RC" -eq 0 ]
check "no warning" lacks "$SITE/out.md" "WARNING"
check "summary has snapshot id" has "$SITE/out.md" "abcd1234"
check "restic backup ran" has "$SITE/calls.log" "restic/restic:0.18.0 backup"
check "tagged nuclear-db" has "$SITE/calls.log" "--tag nuclear-db"
check "fixed host" has "$SITE/calls.log" "--host nuclear-ao3"
check "forget runs after" has "$SITE/calls.log" "forget --tag nuclear-db --host nuclear-ao3 --keep-daily 14 --keep-weekly 8 --keep-monthly 12 --prune"
check "weekly check ran the first time" has "$SITE/calls.log" "check --read-data-subset=1%"
check "--env-file is used" has "$SITE/calls.log" "--env-file $SITE/secrets/offsite.env"
check "backups mounted read-only" has "$SITE/calls.log" "$SITE/backups:/backups:ro"
check "secrets mounted read-only" has "$SITE/calls.log" "$SITE/secrets:/secrets:ro"
check "password is passed as a file path" has "$SITE/calls.log" "RESTIC_PASSWORD_FILE=/secrets/offsite-password"
check "repository comes from the environment" has "$SITE/calls.log.env" "RESTIC_REPOSITORY=s3:https://s3.example.com/bucket/nuclear"
for s in "$SECRET_PW" "$SECRET_KEY" "$SECRET_SK" "repopass"; do
	check "secret '${s:0:8}...' not in argv" lacks "$SITE/calls.log" "$s"
	check "secret '${s:0:8}...' not in summary" lacks "$SITE/out.md" "$s"
done
# Second run the same day: check is not due.
: >"$FAKE_LOG"
sleep 1
run backup.sh
check "second run succeeds" [ "$RC" -eq 0 ]
check "weekly check skipped when not due" lacks "$SITE/calls.log" "check --read-data-subset"

echo "== failing restic"
for stage in backup forget check; do
	new_site yes
	export FAKE_FAIL_ON="$stage"
	run backup.sh
	check "[$stage] exit code is non-zero" [ "$RC" -ne 0 ]
	check "[$stage] summary says FAILED" has "$SITE/out.md" "FAILED"
	check "[$stage] summary says local backup is saved" has "$SITE/out.md" "local backup"
	check "[$stage] summary shows restic's reason" has "$SITE/out.md" "access denied"
	check "[$stage] password in address is hidden" lacks "$SITE/out.md" "hunter2"
	check "[$stage] result is PROBLEMS" has "$SITE/out.md" "Result: PROBLEMS"
	check "[$stage] local file still kept" test -n "$(ls "$SITE"/backups/nuclear-*.sql.gz 2>/dev/null)"
done
new_site yes
rm "$SITE/secrets/offsite.env"
run backup.sh
check "[missing env file] fails with the reason" has "$SITE/out.md" "does not exist"
check "[missing env file] exit non-zero" [ "$RC" -ne 0 ]

echo "== offsite-init.sh"
new_site yes
rm "$SITE/secrets/offsite-password"
run offsite-init.sh
check "init succeeds" [ "$RC" -eq 0 ]
check "init ran restic init" has "$SITE/calls.log" "restic/restic:0.18.0 init"
mode=$(stat -c %a "$SITE/secrets/offsite-password" 2>/dev/null || stat -f %Lp "$SITE/secrets/offsite-password")
check "password file mode is 600 ($mode)" [ "$mode" = "600" ]
size=$(wc -c <"$SITE/secrets/offsite-password" | tr -d ' ')
check "password is 64 hex characters ($size)" [ "$size" -eq 64 ]
check "instructions say to save a copy elsewhere" has "$SITE/out.md" "somewhere else"
check "instructions explain nobody can read without it" has "$SITE/out.md" "including you"
check "password not printed in summary" lacks "$SITE/out.md" "$(cat "$SITE/secrets/offsite-password")"
check "password not in argv" lacks "$SITE/calls.log" "$(cat "$SITE/secrets/offsite-password")"
before=$(cat "$SITE/secrets/offsite-password")
: >"$FAKE_LOG"
run offsite-init.sh
check "second init refuses" [ "$RC" -ne 0 ]
check "password unchanged" [ "$(cat "$SITE/secrets/offsite-password")" = "$before" ]
check "restic not run on refusal" lacks "$SITE/calls.log" "restic/restic"
new_site yes
rm "$SITE/secrets/offsite-password"
export FAKE_FAIL_ON=init
run offsite-init.sh
check "failed init exits non-zero" [ "$RC" -ne 0 ]
check "failed init removes the unused password" test ! -e "$SITE/secrets/offsite-password"
new_site yes
printf 'OFFSITE_REPOSITORY=s3:x\nOFFSITE_ENV_FILE=/etc/passwd\n' >"$SITE/.env"
run offsite-init.sh
check "env file outside secrets/ is refused" [ "$RC" -ne 0 ]
printf 'OFFSITE_REPOSITORY=s3:x\nOFFSITE_ENV_FILE=secrets/../.env\n' >"$SITE/.env"
run offsite-init.sh
check "env file with .. is refused" [ "$RC" -ne 0 ]
printf 'OFFSITE_REPOSITORY=rest:https://user:p@ss/w0rd@backup.example.org/nuclear\nOFFSITE_ENV_FILE=secrets/offsite.env\n' >"$SITE/.env"
run offsite-init.sh
check "address with a login in it is refused" [ "$RC" -ne 0 ]
check "the refusal doesn't print the password" lacks "$SITE/out.md" "w0rd"
printf 'Fatal: at rest:https://u:p@ss/w0rd@h/x and s3:https://k:s3cr/et@h2/y\n' | (. "$HERE/offsite-lib.sh"; scrub) >"$SITE/scrubbed"
check "scrub hides passwords containing / and @" lacks "$SITE/scrubbed" "w0rd"
check "scrub hides every address on a line" lacks "$SITE/scrubbed" "s3cr"

echo "== offsite-list.sh"
new_site yes
run offsite-list.sh
check "list succeeds" [ "$RC" -eq 0 ]
check "list shows the snapshot" has "$SITE/out.md" "abcd1234"
check "list is read-only mounts" has "$SITE/calls.log" ":/backups:ro"
new_site no
run offsite-list.sh
check "list says not set up" has "$SITE/out.md" "not set up"
check "list exits non-zero when not set up" [ "$RC" -ne 0 ]
new_site yes
export FAKE_SNAP_ROW="abcd1234  $(date -u +'%Y-%m-%d %H:%M:%S')  nuclear-ao3  nuclear-db  /backups/x"
run offsite-list.sh --newest-age-hours
check "age is 0 hours for a fresh snapshot ($OUT)" [ "$OUT" = "0" ]
export FAKE_SNAP_ROW="abcd1234  2020-01-01 00:00:00  nuclear-ao3  nuclear-db  /backups/x"
run offsite-list.sh --newest-age-hours
check "age of an old snapshot is large ($OUT)" [ "$OUT" -gt 1000 ]
new_site no
run offsite-list.sh --newest-age-hours
check "age exits 2 when not set up" [ "$RC" -eq 2 ]

echo "== offsite-fetch.sh"
new_site yes
run offsite-fetch.sh latest
check "fetch latest succeeds" [ "$RC" -eq 0 ]
check "fetched under the original name" test -s "$SITE/backups/nuclear-20261030-024712.sql.gz"
fmode=$(stat -c %a "$SITE/backups/nuclear-20261030-024712.sql.gz" 2>/dev/null || stat -f %Lp "$SITE/backups/nuclear-20261030-024712.sql.gz")
check "fetched file mode is 600" [ "$fmode" = "600" ]
check "fetched file is valid gzip" gunzip -t "$SITE/backups/nuclear-20261030-024712.sql.gz"
check "fetch mounts backups read-only" has "$SITE/calls.log" "$SITE/backups:/backups:ro"
run offsite-fetch.sh latest
check "fetch refuses to overwrite" [ "$RC" -ne 0 ]
run offsite-fetch.sh 'abcd; rm -rf /'
check "fetch rejects a bad id" [ "$RC" -ne 0 ]
rm "$SITE/backups/nuclear-20261030-024712.sql.gz"
run offsite-fetch.sh abcd1234
check "fetch by id succeeds" [ "$RC" -eq 0 ]
check "fetch by id dumps that snapshot" has "$SITE/calls.log" "dump abcd1234 /backups/nuclear-20261030-024712.sql.gz"
check "no leftover partial files" test -z "$(ls -A "$SITE/backups" | grep -E 'fetching|err' || true)"

echo "== dispatch"
check "dispatch test passes" bash "$HERE/test-dispatch.sh" >/dev/null

if [ "$FAILS" -eq 0 ]; then echo "all offsite tests passed"; else echo "$FAILS offsite test(s) FAILED"; exit 1; fi
