#!/bin/bash
# Record the buzz demo GIF (scripts/demo/demo.gif).
#
# Builds buzz, starts the mock Beeminder server (scripts/demo/mockserver), points
# buzz at it via an isolated $HOME/.buzzrc (so the real ~/.buzzrc and a real
# account are never touched), then drives the recording with VHS.
#
# Requires: go, curl, vhs (which needs ttyd and ffmpeg). Install with `brew install vhs`.
#
# Usage: scripts/demo/record.sh

set -euo pipefail

# Resolve repo root from this script's location so it works from anywhere.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$REPO_ROOT"

PORT="${BUZZ_DEMO_PORT:-7180}"

for tool in go curl vhs; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "error: '$tool' is required but not installed" >&2
    [ "$tool" = vhs ] && echo "       install with: brew install vhs" >&2
    exit 1
  fi
done

# Isolated workspace: a temp HOME (keeps the real ~/.buzzrc safe) and a bin dir
# holding the demo-built buzz, prepended to PATH so the tape's `buzz` resolves.
WORK_DIR="$(mktemp -d)"
BIN_DIR="$WORK_DIR/bin"
mkdir -p "$BIN_DIR"

# Fail fast if something is already on the port — otherwise the readiness
# probe below would happily pass against a stale/foreign server and record a
# misleading demo.
if curl -sf --connect-timeout 1 --max-time 2 "http://127.0.0.1:$PORT/api/v1/users/demo.json" >/dev/null 2>&1; then
  echo "error: something is already listening on port $PORT; stop it or set BUZZ_DEMO_PORT" >&2
  exit 1
fi

MOCK_PID=""
cleanup() {
  [ -n "$MOCK_PID" ] && kill "$MOCK_PID" 2>/dev/null || true
  rm -rf "$WORK_DIR"
}
trap cleanup EXIT

echo "Building buzz..."
go build -o "$BIN_DIR/buzz" .

echo "Starting mock Beeminder server on port $PORT..."
go run ./scripts/demo/mockserver --port "$PORT" &
MOCK_PID=$!

# Demo config: dummy credentials pointed at the mock server.
cat > "$WORK_DIR/.buzzrc" <<EOF
{"username":"demo","auth_token":"demo","base_url":"http://127.0.0.1:$PORT"}
EOF

# Wait for the mock server to accept connections.
ready=false
for _ in $(seq 1 50); do
  if curl -sf --connect-timeout 1 --max-time 2 "http://127.0.0.1:$PORT/api/v1/users/demo.json" >/dev/null 2>&1; then
    ready=true
    break
  fi
  sleep 0.2
done
if [ "$ready" != true ]; then
  echo "error: mock server did not become ready on port $PORT" >&2
  exit 1
fi

echo "Recording demo with VHS..."
HOME="$WORK_DIR" PATH="$BIN_DIR:$PATH" vhs scripts/demo/demo.tape

echo "Done: scripts/demo/demo.gif"
