#!/usr/bin/env bash
# Shared helpers for the off-server backup scripts (offsite-*.sh). Source it;
# don't run it. Works in bash 3.2 and newer.
#
# restic runs inside a pinned Docker image. Nothing secret is ever put on a
# command line: the repository password is a file in secrets/ (mounted
# read-only), the provider login is a KEY=VALUE file in secrets/ handed over
# with --env-file, and the repository address travels as an environment
# variable. OPS_DOCKER can name a different docker command (the tests use it).
# shellcheck source=lib.sh
. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

# An exact version, never "latest". To upgrade, change it here and run
# 'make offsite-list' to check the new version can still read the repository.
RESTIC_IMAGE="restic/restic:0.18.0"
OFFSITE_HOST="nuclear-ao3"   # fixed, so retention works even if the server is renamed
OFFSITE_TAG="nuclear-db"
OFFSITE_DEFAULT_PASSWORD_FILE="secrets/offsite-password"
OFFSITE_CHECK_STATE="$BACKUPS_DIR/.offsite-last-check"
OFFSITE_CHECK_EVERY=$((7 * 24 * 3600))

DOCKER=()
detect_docker() {
	[ "${#DOCKER[@]}" -gt 0 ] && return 0
	if [ -n "${OPS_DOCKER:-}" ]; then
		# shellcheck disable=SC2206
		DOCKER=($OPS_DOCKER)
	elif command -v docker >/dev/null 2>&1; then
		DOCKER=(docker)
	else
		die "Docker is not installed"
	fi
}

# offsite_abs PATH: PATH made absolute (relative paths are inside OPS_DIR).
offsite_abs() {
	case "$1" in
	/*) printf '%s' "$1" ;;
	*) printf '%s/%s' "$OPS_DIR" "$1" ;;
	esac
}

# offsite_under_secrets PATH: succeeds if PATH is a file name inside secrets/.
offsite_under_secrets() {
	local abs
	abs=$(offsite_abs "$1")
	case "$abs" in *..* | *//*) return 1 ;; esac
	case "$abs" in "$OPS_DIR"/secrets/?*) return 0 ;; esac
	return 1
}

OFFSITE_REPO=""
OFFSITE_PASSWORD_HOST=""   # password file on this server
OFFSITE_PASSWORD_CT=""     # the same file as the container sees it
OFFSITE_ENV_HOST=""
OFFSITE_ERR=""

# offsite_settings: reads the three OFFSITE_* settings from .env and checks
# them. Sets OFFSITE_ERR and returns 1 when something is wrong. Returns 2 when
# OFFSITE_REPOSITORY is not set at all (off-server backup is not configured).
# Pass "init" to skip the check that the password file already exists.
offsite_settings() {
	local pw env
	OFFSITE_REPO=$(env_get OFFSITE_REPOSITORY)
	[ -n "$OFFSITE_REPO" ] || return 2
	# A login inside the address (https://user:password@host) would end up in
	# logs and summaries, so it isn't allowed: logins go in OFFSITE_ENV_FILE.
	# (sftp:user@host:/path has no password in it and is fine.)
	case "$OFFSITE_REPO" in
	*://*@*)
		OFFSITE_ERR="OFFSITE_REPOSITORY must not contain a login (user:password@). Put the login in the OFFSITE_ENV_FILE file instead (for a rest: server, RESTIC_REST_USERNAME and RESTIC_REST_PASSWORD)."
		return 1
		;;
	esac
	pw=$(env_get OFFSITE_PASSWORD_FILE "$OFFSITE_DEFAULT_PASSWORD_FILE")
	env=$(env_get OFFSITE_ENV_FILE)
	if ! offsite_under_secrets "$pw"; then
		OFFSITE_ERR="OFFSITE_PASSWORD_FILE must be a file inside secrets/ (it is '$pw')."
		return 1
	fi
	if [ -z "$env" ]; then
		OFFSITE_ERR="OFFSITE_ENV_FILE is not set. It must name a file inside secrets/ with your storage provider login."
		return 1
	fi
	if ! offsite_under_secrets "$env"; then
		OFFSITE_ERR="OFFSITE_ENV_FILE must be a file inside secrets/ (it is '$env')."
		return 1
	fi
	OFFSITE_PASSWORD_HOST=$(offsite_abs "$pw")
	OFFSITE_PASSWORD_CT="/secrets/${OFFSITE_PASSWORD_HOST#"$OPS_DIR"/secrets/}"
	OFFSITE_ENV_HOST=$(offsite_abs "$env")
	if [ ! -f "$OFFSITE_ENV_HOST" ]; then
		OFFSITE_ERR="The provider login file '$env' does not exist."
		return 1
	fi
	if [ "${1:-}" != "init" ] && [ ! -f "$OFFSITE_PASSWORD_HOST" ]; then
		OFFSITE_ERR="The repository password file '$pw' does not exist. Run 'make offsite-init' (new repository) or put your saved copy there."
		return 1
	fi
	return 0
}

# restic ARGS...: runs restic in the pinned image. backups/ and secrets/ are
# mounted read-only. No terminal, no stdin.
restic() {
	detect_docker
	(
		export RESTIC_REPOSITORY="$OFFSITE_REPO"
		exec "${DOCKER[@]}" run --rm \
			--env-file "$OFFSITE_ENV_HOST" \
			-e RESTIC_REPOSITORY \
			-e "RESTIC_PASSWORD_FILE=$OFFSITE_PASSWORD_CT" \
			-v "$BACKUPS_DIR:/backups:ro" \
			-v "$OPS_DIR/secrets:/secrets:ro" \
			"$RESTIC_IMAGE" "$@" </dev/null
	)
}

# scrub: removes anything between "://" and the last "@" of an address in
# text on stdin (a password may itself contain "/" or "@"), as a second
# layer: offsite_settings already refuses addresses with a login in them.
scrub() { sed -E 's#://[^[:space:]]*@#://#g'; }
