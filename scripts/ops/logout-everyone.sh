#!/usr/bin/env bash
# Ends every login session and OAuth token. Everyone has to log in again.
set -euo pipefail
# shellcheck source=lib.sh
. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

echo "# Log everyone out"
out=$(run_nuclear sessions revoke-all 2>&1) || { printf '%s\n' "$out" | fence; h2 "Result: PROBLEMS"; para "The command failed. Nobody was logged out."; exit 1; }
h2 "Done"
printf '%s\n' "$out" | fence
# Check: running it again should find nothing left (except new logins).
again=$(run_nuclear sessions revoke-all 2>&1 || true)
h2 "Check"
printf '%s\n' "$again" | fence
h2 "Result: OK"
para "Every account has to log in again. If the reason was a leaked signing key, this is not enough on its own: replace the key (runbook, \"Change the signing key right now\")."
