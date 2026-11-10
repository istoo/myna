#!/usr/bin/env bash
# Checks the kept upstream WebSocket against a real ChatGPT account: two
# turns of one session go to the ChatGPT backend on one socket, the second
# as a delta after previous_response_id, and OWL_DEBUG's line says so with
# its bytes.
#
# Needs a ChatGPT account signed in to owl (the `codex` provider has a
# model list; MODEL picks one, codex/gpt-5.5 by default). It builds and starts its own gateway on
# $PORT, so the app's own gateway is untouched; nothing else should serve
# that port.
#
#   scripts/check-codex-kept-socket.sh
#   PORT=8899 OWL_BIN=/path/to/owl scripts/check-codex-kept-socket.sh
#   MODEL=codex/gpt-5.6-terra scripts/check-codex-kept-socket.sh
#
# What it prints: the delta line, and every socket turn's line. It exits
# non-zero when the second turn was whole, the reply failed, or the log
# names no delta.
set -euo pipefail

PORT="${PORT:-8899}"
BASE="http://127.0.0.1:$PORT"
SESSION="kept-socket-check-$$"
MODEL="${MODEL:-codex/gpt-5.5}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="${OWL_BIN:-/tmp/owl-kept}"
LOG="${LOG:-/tmp/owl-kept-socket.log}"
TURN1="${TURN1:-/tmp/owl-kept-turn1.sse}"
TURN2="${TURN2:-/tmp/owl-kept-turn2.sse}"

cleanup() {
	if [[ -n "${PID:-}" ]]; then
		kill "$PID" 2>/dev/null || true
		wait "$PID" 2>/dev/null || true
	fi
}
trap cleanup EXIT

if [[ ! -x "$BIN" ]]; then
	echo "building $BIN"
	(cd "$ROOT" && go build -o "$BIN" .)
fi

: >"$LOG"
OWL_ADDR="127.0.0.1:$PORT" OWL_DEBUG=1 "$BIN" serve >>"$LOG" 2>&1 &
PID=$!

started=
for _ in $(seq 1 240); do
	if curl -fsS "$BASE/v1/models" >/dev/null 2>&1; then
		started=1
		break
	fi
	sleep 0.25
done
if [[ -z "$started" ]]; then
	echo "owl did not answer on $BASE; see $LOG" >&2
	exit 1
fi

USER1='{"type":"message","role":"user","content":[{"type":"input_text","text":"Reply with the single word: one"}]}'
USER2='{"type":"message","role":"user","content":[{"type":"input_text","text":"Reply with the single word: two"}]}'

# turn one: the whole conversation. include the reasoning seal as Codex
# does, so whatever it streamed can be sent back as Codex sends it.
BODY1=$(python3 - "$SESSION" "$USER1" "$MODEL" <<'PY'
import json, sys
session, user1, model = sys.argv[1:4]
print(json.dumps({
    "model": model,
    "stream": True,
    "prompt_cache_key": session,
    "include": ["reasoning.encrypted_content"],
    "input": [json.loads(user1)],
}, separators=(",", ":")))
PY
)
if ! curl -fsS -o "$TURN1" -H 'Content-Type: application/json' \
	-H "session-id: $SESSION" -d "$BODY1" "$BASE/v1/responses"; then
	echo "the first turn failed; see $TURN1" >&2
	cat "$TURN1" >&2 || true
	exit 1
fi

# the reply's output items, in output_item.done order: what Codex sends
# back at the head of the next turn's input
BODY2=$(python3 - "$SESSION" "$USER1" "$USER2" "$TURN1" "$MODEL" <<'PY'
import json, sys
session, user1, user2, turn1, model = sys.argv[1:6]
items = []
for line in open(turn1, encoding="utf-8"):
    if not line.startswith("data: "):
        continue
    try:
        ev = json.loads(line[6:])
    except json.JSONDecodeError:
        continue
    if ev.get("type") == "response.output_item.done":
        items.append(ev["item"])
if not items:
    sys.exit("the first turn streamed no output item")
print(json.dumps({
    "model": model,
    "stream": True,
    "prompt_cache_key": session,
    "include": ["reasoning.encrypted_content"],
    "input": [json.loads(user1)] + items + [json.loads(user2)],
}, separators=(",", ":")))
PY
)

if ! curl -fsS -o "$TURN2" -H 'Content-Type: application/json' \
	-H "session-id: $SESSION" -d "$BODY2" "$BASE/v1/responses"; then
	echo "the second turn failed; see $TURN2" >&2
	cat "$TURN2" >&2 || true
	exit 1
fi
if ! grep -q 'response.completed' "$TURN2"; then
	echo "the second turn didn't complete; see $TURN2" >&2
	cat "$TURN2" >&2 || true
	exit 1
fi

echo "socket turns:"
grep 'turn on the WebSocket' "$LOG" || true
if ! grep -q 'turn on the WebSocket: delta' "$LOG"; then
	echo "the second turn was not a delta: see $LOG, $TURN1, $TURN2" >&2
	exit 1
fi
echo "ok: the session's second turn went on the kept socket as a delta"