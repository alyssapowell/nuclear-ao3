#!/usr/bin/env bash
# Shared helpers for the operations scripts in this folder. Source it; don't
# run it. Every script prints a short Markdown summary on stdout (the GitHub
# workflows paste it into the run page) and sends progress messages to stderr.
#
# OPS_DIR is the folder with docker-compose.yml, .env and secrets/. It
# defaults to the repo this script lives in (normally /opt/nuclear-ao3).
# Works in bash 3.2 and newer.

OPS_DIR="${OPS_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
BACKUPS_DIR="$OPS_DIR/backups"
APP_URL="${OPS_APP_URL:-http://127.0.0.1:8088}"
KEEP_BACKUPS=14
RESTORE_FLAG="--i-understand-this-replaces-the-database"

log() { echo "[ops] $*" >&2; }
die() { echo "[ops] error: $*" >&2; exit 1; }

# ---- Docker Compose -------------------------------------------------------

COMPOSE=()
# Picks "docker compose" or "docker-compose", whichever works here. Set
# OPS_COMPOSE to a command to force one (the tests use this).
detect_compose() {
	[ "${#COMPOSE[@]}" -gt 0 ] && return 0
	if [ -n "${OPS_COMPOSE:-}" ]; then
		# shellcheck disable=SC2206
		COMPOSE=($OPS_COMPOSE)
	elif command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
		COMPOSE=(docker compose)
	elif command -v docker-compose >/dev/null 2>&1; then
		COMPOSE=(docker-compose)
	else
		die "Docker Compose is not installed (need 'docker compose' or 'docker-compose')"
	fi
}

# compose ARGS...: runs Docker Compose in OPS_DIR.
compose() {
	detect_compose
	( cd "$OPS_DIR" && "${COMPOSE[@]}" "$@" )
}

# run_nuclear ARGS...: runs "nuclear ARGS" in a throwaway app container.
# -T because there is no terminal over SSH or in GitHub Actions.
run_nuclear() { compose run --rm -T app nuclear "$@"; }
# Same, but without starting the database; for commands that don't need it.
run_nuclear_nodb() { compose run --rm -T --no-deps app nuclear "$@"; }

# ---- .env -----------------------------------------------------------------

# env_get NAME [DEFAULT]: the value of NAME in .env. Reads the file as text;
# it never runs it.
env_get() {
	local line
	line=$(grep -E "^[[:space:]]*$1=" "$OPS_DIR/.env" 2>/dev/null | tail -n 1 || true)
	if [ -z "$line" ]; then printf '%s' "${2:-}"; return 0; fi
	line=${line#*=}
	line=${line%$'\r'}
	case "$line" in
	\"*\") line=${line#\"}; line=${line%\"} ;;
	\'*\') line=${line#\'}; line=${line%\'} ;;
	esac
	if [ -z "$line" ]; then printf '%s' "${2:-}"; else printf '%s' "$line"; fi
}

# ---- Checking what people typed -------------------------------------------

valid_ref() {
	local re='^[A-Za-z0-9._/-]{1,100}$'
	[[ "${1:-}" =~ $re ]] || return 1
	case "$1" in -* | /* | */ | *..*) return 1 ;; esac
	return 0
}
valid_username() {
	local re='^[A-Za-z0-9_]{3,40}$'
	[[ "${1:-}" =~ $re ]]
}
valid_backup_name() {
	local re='^nuclear-[0-9]{8}-[0-9]{6}\.sql\.gz$'
	[[ "${1:-}" =~ $re ]]
}

# ---- Markdown summary -----------------------------------------------------

PROBLEM_TEXTS=()
PROBLEM_LINKS=()

h2() { printf '\n## %s\n\n' "$1"; }
para() { printf '%s\n\n' "$1"; }
bullet() { printf -- '- %s\n' "$1"; }
# fence: copies stdin into a text block, shortened to the last 40 lines.
fence() {
	local body
	body=$(tail -n 40)
	printf '```text\n%s\n```\n\n' "$body"
}

# The runbook's address, as a full link when the repo has a GitHub remote.
runbook_base() {
	local url
	url=$(git -C "$OPS_DIR" remote get-url origin 2>/dev/null || true)
	url=$(printf '%s' "$url" | sed -e 's#//[^@/]*@#//#' -e 's#^git@github.com:#https://github.com/#' \
		-e 's#^ssh://git@github.com/#https://github.com/#' -e 's#\.git$##')
	case "$url" in
	https://*) printf '%s/blob/main/docs/RUNBOOK.md' "$url" ;;
	*) printf 'docs/RUNBOOK.md' ;;
	esac
}

# problem TEXT RUNBOOK-ANCHOR: records something a human has to look at.
problem() {
	PROBLEM_TEXTS+=("$1")
	PROBLEM_LINKS+=("$(runbook_base)#$2")
}
problem_count() { echo "${#PROBLEM_TEXTS[@]}"; }

# report_problems: prints the final OK / PROBLEMS list. Returns 1 if any.
report_problems() {
	local i
	if [ "${#PROBLEM_TEXTS[@]}" -eq 0 ]; then
		h2 "Result: OK"
		para "Nothing needs attention."
		return 0
	fi
	h2 "Result: PROBLEMS (${#PROBLEM_TEXTS[@]})"
	for i in "${!PROBLEM_TEXTS[@]}"; do
		bullet "${PROBLEM_TEXTS[$i]} ([what to do](${PROBLEM_LINKS[$i]}))"
	done
	echo
	return 1
}

# ---- Checks used by several scripts ---------------------------------------

# wait_healthy [SECONDS]: waits until the app answers /health.
wait_healthy() {
	local n=${1:-90} i=0
	while [ "$i" -lt "$n" ]; do
		if curl -fsS -m 5 -o /dev/null "$APP_URL/health" 2>/dev/null; then return 0; fi
		sleep 2
		i=$((i + 2))
	done
	return 1
}

# file_mtime FILE: modification time in seconds since 1970 (GNU or BSD stat).
file_mtime() { stat -c %Y "$1" 2>/dev/null || stat -f %m "$1"; }

# jobs_totals: prints "WAITING FAILED" summed over all job kinds.
jobs_totals() {
	run_nuclear jobs 2>/dev/null | awk '$2 ~ /^[0-9]+$/ { w += $2; f += $4 } END { print w + 0, f + 0 }'
}
