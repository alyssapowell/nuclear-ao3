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
# It creates two throwaway accounts and one work each time it runs. Needs curl
# and python3.
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

account() {
	local name="smoke$RANDOM$RANDOM"
	curl -s -X POST "$B/auth/register" -H 'Content-Type: application/json' \
		-d "{\"username\":\"$name\",\"email\":\"$name@example.org\",\"password\":\"$PW\",\"confirm_password\":\"$PW\"}" >/dev/null
	curl -s -X POST "$B/auth/login" -H 'Content-Type: application/json' \
		-d "{\"email\":\"$name@example.org\",\"password\":\"$PW\"}" | json "d.get('access_token','')"
}

check "site is up" 200 "$(code "$BASE/health")"

AUTHOR=$(account); READER=$(account)
[ -n "$AUTHOR" ] && [ -n "$READER" ]; check "register and log in" 0 $?

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
