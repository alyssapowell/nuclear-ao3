#!/usr/bin/env bash
# usage: upgrade.sh <git-ref>      (a tag, branch or commit, e.g. v1.4.0)
# Backs up, switches the checkout to <git-ref>, rebuilds, starts, waits for
# health and runs the smoke test. If anything fails after the switch it goes
# back to the previous version and starts that. It never restores the
# database by itself: that is a human decision.
set -euo pipefail
# shellcheck source=lib.sh
. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

git_() { git -C "$OPS_DIR" "$@"; }
BUILD_LOG=$(mktemp)
trap 'rm -f "$BUILD_LOG"' EXIT

main() {
	local ref="${1:-}"
	valid_ref "$ref" || die "usage: upgrade.sh <git-ref>  (letters, digits, . _ / - only)"
	echo "# Upgrade to \`$ref\`"

	# 1. Where we are now.
	local prev_sha prev_name
	prev_sha=$(git_ rev-parse HEAD)
	prev_name=$(git_ symbolic-ref -q --short HEAD || true)
	[ -n "$prev_name" ] || prev_name="$prev_sha"
	h2 "Before"
	bullet "Running version: \`${prev_name}\` (\`${prev_sha:0:10}\`)"

	# Refuse early if the tree has local changes; checkout could lose them.
	if [ -n "$(git_ status --porcelain --untracked-files=no)" ]; then
		bullet "The checkout on the server has uncommitted changes, so nothing was changed."
		h2 "Result: PROBLEMS"
		para "A human must clean up \`$OPS_DIR\` (git status) and run this again."
		exit 1
	fi

	# 2. Backup.
	log "backing up"
	local backup_out backup
	if ! backup_out=$("$HERE/backup.sh"); then
		h2 "Result: PROBLEMS"
		para "The backup failed, so the upgrade did not start. Nothing was changed."
		exit 1
	fi
	backup=$(printf '%s\n' "$backup_out" | grep -o 'nuclear-[0-9]\{8\}-[0-9]\{6\}\.sql\.gz' | head -n 1)
	bullet "Backup taken: \`$backup\`"

	# 3. Fetch and switch. A failure to fetch changes nothing yet.
	log "fetching"
	if ! git_ fetch --tags --prune origin >&2; then
		h2 "Result: PROBLEMS"
		para "Could not fetch from GitHub. Nothing was changed."
		exit 1
	fi
	local target=""
	local cand
	for cand in "refs/tags/$ref" "refs/remotes/origin/$ref" "$ref"; do
		if target=$(git_ rev-parse --verify --quiet "$cand^{commit}"); then break; fi
		target=""
	done
	if [ -z "$target" ]; then
		h2 "Result: PROBLEMS"
		para "\`$ref\` is not a tag, branch or commit in the repository. Nothing was changed."
		exit 1
	fi
	bullet "Target: \`$ref\` (\`${target:0:10}\`)"

	# From here on, a failure rolls back.
	local why=""
	if ! git_ checkout --quiet --detach "$target" >&2; then
		why="could not switch to $ref"
	fi

	local pending_before="unknown"
	if [ -z "$why" ]; then
		log "building"
		if ! compose build app >"$BUILD_LOG" 2>&1; then
			why="the build failed"
		fi
	fi
	if [ -z "$why" ]; then
		# The new image knows the new migrations. This lists the ones the
		# upgrade is about to apply (it applies nothing itself).
		pending_before=$(run_nuclear migrate status 2>/dev/null || echo "unknown")
		log "starting"
		if ! compose up -d >&2; then why="docker compose up failed"; fi
	fi
	if [ -z "$why" ]; then
		log "waiting for the app to be healthy"
		if ! wait_healthy 180; then why="the app did not become healthy within 3 minutes"; fi
	fi
	local smoke_out=""
	if [ -z "$why" ]; then
		log "smoke test"
		if ! smoke_out=$(cd "$OPS_DIR" && SMOKE_READ_ONLY=1 bash scripts/smoke-test.sh "$APP_URL" 2>&1); then
			why="the smoke test found a failure"
		elif printf '%s\n' "$smoke_out" | grep -q '^FAIL'; then
			why="the smoke test found a failure"
		fi
	fi

	local pending_after
	pending_after=$(run_nuclear migrate status 2>/dev/null || echo "unknown")
	local migrated="no"
	case "$pending_before" in
	"up to date") ;;
	unknown) migrated="unknown" ;;
	*) migrated="yes" ;;
	esac

	if [ -n "$smoke_out" ]; then
		h2 "Smoke test"
		printf '%s\n' "$smoke_out" | fence
	fi

	if [ -z "$why" ]; then
		h2 "Migrations"
		if [ "$migrated" = "yes" ]; then
			para "Applied by this upgrade:"
			printf '%s\n' "$pending_before" | sed 's/^pending: /applied: /' | fence
		else
			para "No new migrations."
		fi
		para "Status now: $(printf '%s' "$pending_after" | tr '\n' ' ')"
		h2 "Result: OK"
		bullet "Now running \`$ref\` (\`${target:0:10}\`). Previous version was \`$prev_name\`; if something looks wrong later, run this workflow with that value."
		bullet "Backup from before the upgrade: \`$backup\`."
		return 0
	fi

	# ---- Roll back the code (never the database).
	log "FAILED: $why; going back to $prev_name"
	h2 "What failed"
	para "$why."
	if [ -s "$BUILD_LOG" ]; then tail -n 25 "$BUILD_LOG" | fence; fi
	local rb="done"
	if ! git_ checkout --quiet "$prev_name" >&2; then
		git_ checkout --quiet --detach "$prev_sha" >&2 || rb="failed"
	fi
	if [ "$rb" = "done" ]; then
		compose build app >/dev/null 2>&1 || rb="failed"
		compose up -d >&2 || rb="failed"
		wait_healthy 180 || rb="not healthy"
	fi
	h2 "Rollback"
	if [ "$rb" = "done" ]; then
		para "Went back to \`$prev_name\` and it is healthy again."
	else
		para "Went back to \`$prev_name\` but the result is **$rb**. Run the diagnose workflow and read the runbook."
	fi

	h2 "A human must decide"
	if [ "$migrated" = "yes" ]; then
		para "**Database migrations were applied** by the new version before it failed or was undone:"
		printf '%s\n' "$pending_before" | fence
		para "The older version normally still runs on the newer database, and nothing was restored automatically. If the site is wrong, restore the backup \`$backup\` with the **ops-restore** workflow. That erases everything saved after the backup was taken."
	elif [ "$migrated" = "no" ]; then
		para "No database migrations ran, so the database was not touched. The backup \`$backup\` is there if you want it."
	else
		para "Could not tell whether migrations ran. Check with the diagnose workflow. The backup is \`$backup\`; the **ops-restore** workflow can restore it."
	fi
	exit 1
}

# Everything is inside main, and main is called on the last line, so bash has
# read the whole script before git swaps files under it.
main "$@"; exit $?
