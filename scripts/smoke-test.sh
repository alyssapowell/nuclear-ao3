#!/usr/bin/env bash
# Checks that the core of the archive works, end to end, through the public
# API: accounts, posting, drafts staying private, publishing, reading,
# comments, kudos, search and notifications.
#
# Run it after an upgrade, or any time you're not sure the site works:
#
#   scripts/smoke-test.sh                       # against http://localhost:8088
#   scripts/smoke-test.sh https://your.archive  # against a real site
#
# It creates three throwaway accounts and one work each time it runs. Accounts
# must be activated from their email, so it reads that email from MailHog (the
# development mail catcher) at $MAILHOG_URL, http://localhost:8025 by default.
# Needs curl and python3.
# Login, password reset and activation allow 20 tries per address every 5
# minutes, and one full run uses about 13, so a second full run straight
# after the first gets "too many requests". Wait 5 minutes between full runs.
set -u
BASE="${1:-http://localhost:8088}"
B="$BASE/api/v1"
FAILED=0
PW="Smoke-Test-$RANDOM-Password1"

check() { # name expected actual
	if [ "$2" = "$3" ]; then echo "PASS  $1"; else echo "FAIL  $1 (expected $2, got $3)"; FAILED=1; fi
}
json() { python3 -c "import sys,json
try: d=json.load(sys.stdin); print($1)
except Exception: print('')"; }
code() { curl -s -o /dev/null -w '%{http_code}' "$@"; }

MAILHOG="${MAILHOG_URL:-http://localhost:8025}"

# token_for ADDRESS PATH: the token in the newest emailed link to PATH (such
# as /auth/activate) sent to ADDRESS. Emails go out in the background, so it
# waits a few seconds for one.
token_for() {
	local i
	for i in 1 2 3 4 5 6 7 8 9 10; do
		curl -s "$MAILHOG/api/v2/search?kind=to&query=$1" | python3 -c "
import sys, json, quopri, re
try:
    items = json.load(sys.stdin).get('items') or []
    body = quopri.decodestring(items[0]['Content']['Body'].encode()).decode('utf-8', 'replace')
    m = re.search(r'$2\?token=([A-Za-z0-9_-]+)', body)
    print(m.group(1) if m else '')
except Exception:
    print('')" | grep . && return
		sleep 1
	done
}

# activate TOKEN PASSWORD -> prints the HTTP status. (The JSON body is built
# in a variable: macOS's bash 3.2 brace-expands an inline body with a comma.)
activate() {
	local body="{\"token\":\"$1\",\"password\":\"$2\"}"
	code -X POST "$B/auth/verify-email" -H 'Content-Type: application/json' -d "$body"
}

register() { # name
	curl -s -X POST "$B/auth/register" -H 'Content-Type: application/json' \
		-d "{\"username\":\"$1\",\"email\":\"$1@example.org\",\"password\":\"$PW\",\"confirm_password\":\"$PW\"}"
}
login() { # name password -> prints the response
	curl -s -w '\n%{http_code}' -X POST "$B/auth/login" -H 'Content-Type: application/json' \
		-d "{\"login\":\"$1@example.org\",\"password\":\"$2\"}"
}
token_of() { sed '$d' | json "d.get('access_token','')"; }

account() {
	local name="smoke$RANDOM$RANDOM"
	register "$name" >/dev/null
	activate "$(token_for "$name@example.org" /auth/activate)" "$PW" >/dev/null
	login "$name" "$PW" | token_of
}

check "site is up" 200 "$(code "$BASE/health")"

# SMOKE_READ_ONLY=1 checks a live archive without changing anything: no
# accounts, works or emails. This is what upgrades run against the real site.
if [ "${SMOKE_READ_ONLY:-}" = "1" ]; then
	check "works list answers" 200 "$(code "$B/works?limit=1")"
	check "search answers" 200 "$(code "$B/search/works?q=the&limit=1")"
	# Check the content, not only the status: a front door that doesn't send
	# this address to the app would answer 200 with the website's page.
	check "login keys are published" yes "$(curl -s "$BASE/auth/jwks" | json "'yes' if d.get('keys') else 'no'")"
	check "OpenID discovery is published" yes "$(curl -s "$BASE/.well-known/openid-configuration" | json "'yes' if d.get('jwks_uri') else 'no'")"
	BAD_LOGIN='{"login":"no-such-account-smoke","password":"not-a-real-password"}'
	check "login rejects a wrong password" 401 "$(code -X POST "$B/auth/login" -H 'Content-Type: application/json' -d "$BAD_LOGIN")"
	check "logged-out requests are refused" 401 "$(code "$B/auth/sessions")"
	if [ "$FAILED" = 0 ]; then echo "All checks passed."; else echo "Some checks failed."; fi
	exit "$FAILED"
fi

AUTHOR=$(account); READER=$(account)
[ -n "$AUTHOR" ] && [ -n "$READER" ]; check "register and log in" 0 $?

# The account lifecycle, on a third account.
NAME="smoke$RANDOM$RANDOM"
check "register returns no token" "" "$(register "$NAME" | json "d.get('access_token','')")"
check "can't log in before activating" 403 "$(login "$NAME" "$PW" | tail -n1)"
check "wrong password doesn't reveal the account" 401 "$(login "$NAME" "wrong-$PW" | tail -n1)"
ACT=$(token_for "$NAME@example.org" /auth/activate)
[ -n "$ACT" ]; check "activation email arrives" 0 $?
check "activating needs the account's password" 401 "$(activate "$ACT" "wrong-$PW")"
check "activate the account" 204 "$(activate "$ACT" "$PW")"
check "activation link works once" 400 "$(activate "$ACT" "$PW")"
AT_NAME='{"username":"a@b.org","email":"ab@example.org","password":"long-enough-1"}'
check "usernames can't contain @" 400 "$(code -X POST "$B/auth/register" -H 'Content-Type: application/json' -d "$AT_NAME")"
OLD=$(login "$NAME" "$PW" | token_of)
[ -n "$OLD" ]; check "log in once activated" 0 $?
NEWPW="Changed-$RANDOM-Password2"
# The body goes in a variable first: macOS's bash 3.2 brace-expands a JSON
# body with a comma when it's written inline inside "$(...)".
CHANGE="{\"current_password\":\"$PW\",\"new_password\":\"$NEWPW\"}"
check "change password" 204 "$(code -X POST "$B/auth/change-password" -H "Authorization: Bearer $OLD" -H 'Content-Type: application/json' -d "$CHANGE")"
check "current login survives a password change" 200 "$(code "$B/auth/sessions" -H "Authorization: Bearer $OLD")"
check "old password no longer logs in" 401 "$(login "$NAME" "$PW" | tail -n1)"
check "new password logs in" 200 "$(login "$NAME" "$NEWPW" | tail -n1)"
check "password reset request is accepted" 202 "$(code -X POST "$B/auth/reset-password" -H 'Content-Type: application/json' -d "{\"email\":\"$NAME@example.org\"}")"

WORK=$(curl -s -X POST "$B/works" -H "Authorization: Bearer $AUTHOR" -H 'Content-Type: application/json' \
	-d '{"title":"Smoke test","language":"en","rating":"general","fandoms":["Original Work"],"freeform_tags":["Testing"],"chapter_content":"<p>One two three four.</p>"}' |
	json "d['work']['id']")
[ -n "$WORK" ]; check "post a work" 0 $?

check "draft is hidden from others" 404 "$(code "$B/works/$WORK" -H "Authorization: Bearer $READER")"
check "draft is visible to its author" 200 "$(code "$B/works/$WORK" -H "Authorization: Bearer $AUTHOR")"
check "only the author can publish" 403 "$(code -X PUT "$B/works/$WORK" -H "Authorization: Bearer $READER" -H 'Content-Type: application/json' -d '{"status":"posted"}')"
check "author publishes" 200 "$(code -X PUT "$B/works/$WORK" -H "Authorization: Bearer $AUTHOR" -H 'Content-Type: application/json' -d '{"status":"posted"}')"
check "logged-out reader can read it" 200 "$(code "$B/works/$WORK")"
check "first chapter is there" 1 "$(curl -s "$B/works/$WORK/chapters" | json "len(d.get('chapters',[]))")"
found() { # how many times $WORK appears in a search; extra curl args allowed
	curl -s "$B/search/works?q=smoke&sort=published_at&limit=100" "$@" | json "sum(1 for w in d.get('works', []) if w['id'] == '$WORK')"
}
check "search finds the posted work" 1 "$(found)"

COMMENT=$(curl -s -X POST "$B/works/$WORK/comments" -H "Authorization: Bearer $READER" -H 'Content-Type: application/json' -d '{"content":"Lovely."}')
READER_PSEUD=$(echo "$COMMENT" | json "d.get('pseudonym_id','')")
[ -n "$READER_PSEUD" ]; check "reader comments" 0 $?
FORGED="{\"content\":\"x\",\"pseudonym_id\":\"$READER_PSEUD\"}"
check "can't comment as someone else's pseud" 403 "$(code -X POST "$B/works/$WORK/comments" -H "Authorization: Bearer $AUTHOR" -H 'Content-Type: application/json' -d "$FORGED")"
check "guest comments" 201 "$(code -X POST "$B/works/$WORK/comments" -H 'Content-Type: application/json' -d '{"content":"Hi!","guest_name":"Lurker"}')"
check "reader leaves kudos" 201 "$(code -X POST "$B/works/$WORK/kudos" -H "Authorization: Bearer $READER")"

check "author restricts the work to logged-in users" 200 "$(code -X PUT "$B/works/$WORK" -H "Authorization: Bearer $AUTHOR" -H 'Content-Type: application/json' -d '{"restricted_to_users":true}')"
check "restricted work is hidden when logged out" 404 "$(code "$B/works/$WORK")"
logged_out_chapters=$(code "$B/works/$WORK/chapters")
[ "$logged_out_chapters" = 403 ] || [ "$logged_out_chapters" = 404 ]; check "restricted chapters are hidden when logged out" 0 $?
check "restricted work is readable when logged in" 200 "$(code "$B/works/$WORK" -H "Authorization: Bearer $READER")"
check "search hides the restricted work when logged out" 0 "$(found)"
check "search shows the restricted work when logged in" 1 "$(found -H "Authorization: Bearer $READER")"

sleep 1 # notifications are delivered in the background
check "author is notified of comments" 2 "$(curl -s "$B/notifications" -H "Authorization: Bearer $AUTHOR" | json "len(d.get('notifications',[]))")"

if [ $FAILED = 0 ]; then echo "All checks passed."; else echo "Some checks failed."; fi
exit $FAILED
