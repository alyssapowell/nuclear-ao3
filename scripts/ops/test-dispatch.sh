#!/usr/bin/env bash
# Tests dispatch.sh without Docker. A fake OPS_RUN command prints the argv
# dispatch would have run, so we can compare. Run: bash scripts/ops/test-dispatch.sh
set -euo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OPS_DIR="$(cd "$HERE/../.." && pwd)"
export OPS_DIR

FAKE=$(mktemp)
trap 'rm -f "$FAKE"' EXIT
cat >"$FAKE" <<'FAKE_EOF'
#!/usr/bin/env bash
printf 'RUN'
for a in "$@"; do printf ' [%s]' "$a"; done
printf '\n'
FAKE_EOF
chmod +x "$FAKE"
export OPS_RUN="$FAKE"

FAILS=0
H="$HERE"
B="$OPS_DIR/backups"
FLAG="--i-understand-this-replaces-the-database"

expect_run() { # input expected-output
	local out
	if ! out=$(SSH_ORIGINAL_COMMAND="$1" "$HERE/dispatch.sh" 2>&1); then
		echo "FAIL  should run: '$1' -> $out"; FAILS=$((FAILS + 1)); return
	fi
	if [ "$out" = "$2" ]; then echo "PASS  '$1'"; else
		echo "FAIL  '$1' -> $out (wanted $2)"; FAILS=$((FAILS + 1))
	fi
}
expect_refused() { # input
	local out rc=0
	out=$(SSH_ORIGINAL_COMMAND="$1" "$HERE/dispatch.sh" 2>&1) || rc=$?
	if [ "$rc" -ne 0 ] && [ "$out" = "not allowed" ]; then echo "PASS  refused '$1'"; else
		echo "FAIL  not refused: '$1' (exit $rc, output: $out)"; FAILS=$((FAILS + 1))
	fi
}

expect_run "diagnose" "RUN [$H/diagnose.sh]"
expect_run "emails" "RUN [$H/emails.sh]"
expect_run "emails --retry" "RUN [$H/emails.sh] [--retry]"
expect_run "backup" "RUN [$H/backup.sh]"
expect_run "offsite-list" "RUN [$H/offsite-list.sh]"
expect_run "upgrade v1.2.3" "RUN [$H/upgrade.sh] [v1.2.3]"
expect_run "upgrade origin/main" "RUN [$H/upgrade.sh] [origin/main]"
expect_run "restore nuclear-20260101-120000.sql.gz $FLAG" "RUN [$H/restore.sh] [$B/nuclear-20260101-120000.sql.gz] [$FLAG]"
expect_run "logout-everyone" "RUN [$H/logout-everyone.sh]"
expect_run "logout-user alice.b+1@example.org" "RUN [$H/logout-user.sh] [alice.b+1@example.org]"
expect_run "logout-user 123e4567-e89b-12d3-a456-426614174000" "RUN [$H/logout-user.sh] [123e4567-e89b-12d3-a456-426614174000]"

expect_refused 'diagnose; rm -rf /'
expect_refused 'upgrade $(id)'
expect_refused 'upgrade `id`'
expect_refused 'upgrade ../x'
expect_refused 'upgrade a..b'
expect_refused 'upgrade -x'
expect_refused 'upgrade --force'
expect_refused 'upgrade /etc/passwd'
expect_refused 'upgrade'
expect_refused 'upgrade v1 v2'
expect_refused 'restore ../../etc/passwd'
expect_refused 'restore nuclear-1.sql.gz'
expect_refused 'restore nuclear-20260101-120000.sql.gz'
expect_refused "restore /etc/nuclear-20260101-120000.sql.gz $FLAG"
expect_refused "restore nuclear-20260101-120000.sql $FLAG"
expect_refused 'logout-user a;b@example.org'
expect_refused 'logout-user alice'
expect_refused 'logout-user alice@example.org bob@example.org'
expect_refused 'logout-user $(id)@example.org'
expect_refused 'logout-everyone now'
expect_refused 'diagnose extra'
expect_refused 'emails --retry --x'
expect_refused 'offsite-list now'
expect_refused 'offsite-list --newest-age-hours'
expect_refused 'offsite-fetch latest'
expect_refused 'offsite-init'
expect_refused 'offsite-push x'
expect_refused 'emails --RETRY'
expect_refused 'diagnose  '
expect_refused 'bash -c id'
expect_refused 'launch-missiles'
expect_refused ''
expect_refused $'diagnose\nid'
expect_refused $'diagnose\tid'
expect_refused 'diagnose | cat'

# No SSH_ORIGINAL_COMMAND at all (an interactive login attempt).
rc=0
out=$(env -u SSH_ORIGINAL_COMMAND "$HERE/dispatch.sh" 2>&1) || rc=$?
if [ "$rc" -ne 0 ] && [ "$out" = "not allowed" ]; then echo "PASS  refused unset command"; else echo "FAIL  unset command"; FAILS=$((FAILS + 1)); fi

if [ "$FAILS" -eq 0 ]; then echo "all dispatch tests passed"; else echo "$FAILS dispatch test(s) FAILED"; exit 1; fi
