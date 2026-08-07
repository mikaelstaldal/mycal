#!/usr/bin/env bash
# Runs the Playwright suite against a freshly started server.
#
# This is the entry point CI uses, and it is the one to use locally when you
# want the same conditions CI gets. Interactively, `playwright-test` from the
# e2e directory is still the way to run a single spec (see AGENTS.md).
#
# It does not build. Run ./build.sh first — and note that the binary embeds
# web/static/, so a server started from a stale binary serves stale assets and
# the suite then measures something other than what you edited. The freshness
# check below exists to make that impossible rather than merely unlikely.
set -euo pipefail

# Every path below is repo-relative, including the freshness check — without
# this, running from elsewhere fails with a "stale app.css" message that is
# actively misleading about what went wrong.
cd "$(dirname "${BASH_SOURCE[0]}")"

BINARY=./mycal
PORT=8089
# A fresh database per run, in a directory this script owns and removes.
# Reusing one is how an "empty" run silently becomes a run against whatever the
# last one left behind.
DATA_DIR=$(mktemp -d)

SERVER_PID=

cleanup() {
    if [ -n "$SERVER_PID" ]; then
        kill "$SERVER_PID" 2>/dev/null || true
        wait "$SERVER_PID" 2>/dev/null || true
    fi
    rm -rf "$DATA_DIR"
}

trap cleanup EXIT

if [ ! -x "$BINARY" ]; then
    echo "$BINARY not found or not executable — run ./build.sh first" >&2
    exit 1
fi

# Refuse to start if the port is already taken. Ours would exit on bind failure
# while the readiness probe below succeeded against whatever is squatting, and
# the suite would then run against a database and a binary that are not the ones
# under test. That happened during the work that added this script.
#
# Checked here rather than by watching our own process afterwards: a background
# process that has exited is a zombie until reaped, and `kill -0` succeeds on a
# zombie — so the obvious liveness check silently passes in exactly this case.
if (exec 3<>"/dev/tcp/127.0.0.1/${PORT}") 2>/dev/null; then
    exec 3<&- 3>&-
    echo "Something is already listening on port ${PORT}." >&2
    echo "Stop it first — otherwise these tests would run against it, not against this build." >&2
    exit 1
fi

# -public-url must match the baseURL origin in e2e/playwright.config.ts, or CSRF
# rejects mutating requests with 403.
#
# **Not "every mutating request", and not "every write test fails"**, which is
# what this said until it was measured. go-server-common's csrf.Middleware allows
# a non-GET carrying *neither* Origin nor Referer — "native client, allow", in
# its own doc comment (v1.8.0, csrf/csrf.go). Measured here, against a server
# started with a deliberately mismatched -public-url:
#
#     curl, no Origin and no Referer      201
#     curl, Origin: http://localhost:PORT 403
#     curl, Referer only                  403
#     curl, Origin: null                  403
#     Playwright `request` fixture        201
#     in-page fetch()                     403
#
# The last two are the ones that decide it. Everything this suite does through
# the `request` fixture — clearAllEvents in the helpers, every API-level setup
# step — sails through a mismatched flag; anything the browser issues from the
# page is stamped with the page's Origin and is rejected. So the flag is still
# required, and it would bite the moment a spec clicked Save rather than posting
# its fixture.
#
# The wrong rationale mattered because it predicts a 403 for exactly the calls
# that are in fact allowed: someone debugging a write failure would come here,
# see the flag already correct, and rule out the right cause.
"$BINARY" -port "$PORT" -data "$DATA_DIR" -public-url "http://localhost:${PORT}" &
SERVER_PID=$!

for i in $(seq 1 40); do
    if curl -sf "http://localhost:${PORT}/api/v1/events?from=2000-01-01T00:00:00Z&to=2000-01-02T00:00:00Z" > /dev/null 2>&1; then
        break
    fi
    if [ "$i" -eq 40 ]; then
        echo "Server failed to start on port ${PORT}" >&2
        exit 1
    fi
    sleep 0.5
done

# Prove the server is serving the build under test. A green suite run against a
# stale binary is not evidence, and it fails in the reassuring direction: the
# tests pass, describing a version of the app that is not the one on disk.
#
# Every emitted asset, not a sample of them. This used to hash app.css and
# app.js alone, on the reasoning that a change to web/ts/** shows up in the JS —
# but tsc emits one module per source file, and there are ~40 of them under
# web/static/. Editing views/MonthView.ts leaves BOTH hashed files byte-identical,
# so the check passed a server serving 39 stale modules. A guard that samples the
# assets is not a guard against staleness; it is a guard against staleness in the
# two files nobody was going to edit alone.
#
# vendor/ is excluded because it is committed rather than emitted, and the demo
# worker's output is included — it is served the same way and goes stale the
# same way.
stale=0
checked=0
while IFS= read -r path; do
    asset=${path#web/static/}
    if ! served=$(curl -sf "http://localhost:${PORT}/${asset}" | md5sum | cut -d' ' -f1); then
        echo "Could not fetch /${asset} from the test server." >&2
        exit 1
    fi
    ondisk=$(md5sum "$path" | cut -d' ' -f1)
    if [ "$served" != "$ondisk" ]; then
        echo "Server is serving a stale ${asset} (served $served, on disk $ondisk)." >&2
        stale=$((stale + 1))
    fi
    checked=$((checked + 1))
done < <(find web/static \( -name '*.js' -o -name '*.css' \) -not -path '*/vendor/*' | sort)

# A find that matches nothing would report zero stale files and read as a pass —
# the empty-set failure this suite has been bitten by before.
if [ "$checked" -eq 0 ]; then
    echo "Freshness check found no assets to compare — is web/static/ built?" >&2
    exit 1
fi
if [ "$stale" -gt 0 ]; then
    echo "${stale} of ${checked} assets are stale." >&2
    echo "The binary embeds web/static/ — rebuild with ./build.sh and try again." >&2
    exit 1
fi

cd e2e
# `playwright-test` is a local wrapper for exactly this command and is the
# interactive entry point AGENTS.md names; CI has no such wrapper, so fall
# through to the bin npm links. (Both @playwright/test and playwright declare a
# `playwright` bin, so which package npm links is a hoisting detail — but the
# bin is stable either way, which the hardcoded node_modules/playwright/cli.js
# path was not.)
if command -v playwright-test > /dev/null 2>&1; then
    playwright-test "$@"
elif [ -x ./node_modules/.bin/playwright ]; then
    ./node_modules/.bin/playwright test "$@"
else
    echo "Playwright is not installed — run 'npm ci' in e2e/ first." >&2
    exit 1
fi
