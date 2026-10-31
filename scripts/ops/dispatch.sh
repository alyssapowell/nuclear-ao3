#!/usr/bin/env bash
# The SSH "forced command". In ~ops/.ssh/authorized_keys the key is tied to
# this script, so whatever the client asks for arrives in $SSH_ORIGINAL_COMMAND
# and only the operations listed below can run. The text is split on spaces
# and checked word by word; it is never passed to a shell or eval.
#
# Allowed:
#   diagnose
#   emails [--retry]
#   backup
#   upgrade <git-ref>
#   restore <backup-file> --i-understand-this-replaces-the-database
#   logout-everyone
#   logout-user <username>
#
# For tests, OPS_RUN=<command> makes this run that command with the argv it
# would have run, instead of running the real script.
set -euo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
. "$HERE/lib.sh"

deny() { echo "not allowed" >&2; exit 126; }

cmd="${SSH_ORIGINAL_COMMAND:-}"
[ -n "$cmd" ] || deny
# Only plain characters. This already rules out ; & | $ ` ( ) < > quotes,
# backslashes, tabs and newlines.
case "$cmd" in *$'\n'* | *$'\r'* | *$'\t'*) deny ;; esac
re='^[A-Za-z0-9._/ -]+$'
[[ "$cmd" =~ $re ]] || deny

IFS=' ' read -r -a words <<<"$cmd"
[ "${#words[@]}" -ge 1 ] || deny
op="${words[0]}"
nargs=$((${#words[@]} - 1))
# Two spaces in a row would hide an empty word; be strict.
case "$cmd" in *"  "* | " "* | *" ") deny ;; esac

argv=()
case "$op" in
diagnose) [ "$nargs" -eq 0 ] || deny; argv=("$HERE/diagnose.sh") ;;
backup) [ "$nargs" -eq 0 ] || deny; argv=("$HERE/backup.sh") ;;
logout-everyone) [ "$nargs" -eq 0 ] || deny; argv=("$HERE/logout-everyone.sh") ;;
emails)
	if [ "$nargs" -eq 0 ]; then argv=("$HERE/emails.sh")
	elif [ "$nargs" -eq 1 ] && [ "${words[1]}" = "--retry" ]; then argv=("$HERE/emails.sh" --retry)
	else deny; fi
	;;
upgrade)
	[ "$nargs" -eq 1 ] || deny
	valid_ref "${words[1]}" || deny
	argv=("$HERE/upgrade.sh" "${words[1]}")
	;;
restore)
	[ "$nargs" -eq 2 ] || deny
	valid_backup_name "${words[1]}" || deny
	[ "${words[2]}" = "$RESTORE_FLAG" ] || deny
	argv=("$HERE/restore.sh" "$BACKUPS_DIR/${words[1]}" "$RESTORE_FLAG")
	;;
logout-user)
	[ "$nargs" -eq 1 ] || deny
	valid_username "${words[1]}" || deny
	argv=("$HERE/logout-user.sh" "${words[1]}")
	;;
*) deny ;;
esac

if [ -n "${OPS_RUN:-}" ]; then
	exec "$OPS_RUN" "${argv[@]}"
fi
export OPS_DIR
exec "${argv[@]}"
