#!/usr/bin/env bash
# usage: logout-user.sh <username>
# Ends every login session and OAuth token of one account.
set -euo pipefail
# shellcheck source=lib.sh
. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

user="${1:-}"
valid_username "$user" || die "usage: logout-user.sh <username>  (3 to 40 letters, digits or _)"
echo "# Log out \`$user\`"
out=$(run_nuclear sessions revoke "$user" 2>&1) || { printf '%s\n' "$out" | fence; h2 "Result: PROBLEMS"; para "The command failed (is the name spelled right?). Nobody was logged out."; exit 1; }
h2 "Done"
printf '%s\n' "$out" | fence
again=$(run_nuclear sessions revoke "$user" 2>&1 || true)
h2 "Check"
printf '%s\n' "$again" | fence
h2 "Result: OK"
para "\`$user\` has to log in again on every device."
