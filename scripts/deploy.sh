#!/usr/bin/env bash
set -euo pipefail

# Target host to deploy to. Keep empty by default to avoid leaking infra details.
SERVER_HOST=${SERVER_HOST:-""}
# Default to standard SSH port; override as needed.
SERVER_PORT=${SERVER_PORT:-22}
SERVER_USER=${SERVER_USER:-ubuntu}
SSH_KEY=${SSH_KEY:-$HOME/.ssh/id_ed25519}
REMOTE_BASE=${REMOTE_BASE:-/data/clawfleet}
REMOTE_BIN=${REMOTE_BIN:-$REMOTE_BASE/bin}
REMOTE_WEB=${REMOTE_WEB:-$REMOTE_BASE/web}

ADMIN_USER=${ADMIN_USER:-admin}
ADMIN_PASS=${ADMIN_PASS:-""}
AGENT_KEY=${AGENT_KEY:-""}

if [ -z "$SERVER_HOST" ]; then
  echo "ERROR: SERVER_HOST must be set (do not hardcode infra details in this script)." >&2
  exit 1
fi

if [ -z "$ADMIN_PASS" ] || [ -z "$AGENT_KEY" ]; then
  echo "ERROR: ADMIN_PASS and AGENT_KEY must be set (do not hardcode secrets in this script)." >&2
  exit 1
fi

ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)

export PATH="$PATH:/usr/local/go/bin"


# Build server
cd "$ROOT_DIR"
mkdir -p dist/bin
go build -o dist/bin/clawfleet-server ./cmd/server

# Build web
cd "$ROOT_DIR/web"
if command -v npm >/dev/null 2>&1; then
  npm run build
else
  echo "npm not found; skip web build" >&2
fi

# Upload binaries
scp -P "$SERVER_PORT" -i "$SSH_KEY" "$ROOT_DIR/dist/bin/clawfleet-server" "$SERVER_USER@$SERVER_HOST:/tmp/clawfleet-server"

# Upload web
if [ -d "$ROOT_DIR/web/dist" ]; then
  scp -P "$SERVER_PORT" -i "$SSH_KEY" "$ROOT_DIR/web/dist/index.html" "$SERVER_USER@$SERVER_HOST:$REMOTE_WEB/"
  ssh -p "$SERVER_PORT" -i "$SSH_KEY" "$SERVER_USER@$SERVER_HOST" "mkdir -p $REMOTE_WEB/dist/assets"
  scp -P "$SERVER_PORT" -i "$SSH_KEY" "$ROOT_DIR/web/dist/assets"/* "$SERVER_USER@$SERVER_HOST:$REMOTE_WEB/dist/assets/"
fi

# Restart server
ssh -p "$SERVER_PORT" -i "$SSH_KEY" "$SERVER_USER@$SERVER_HOST" \
  "fuser -k 8090/tcp 2>/dev/null; rm -f $REMOTE_BIN/clawfleet-server; cp /tmp/clawfleet-server $REMOTE_BIN/; chmod +x $REMOTE_BIN/clawfleet-server; nohup $REMOTE_BIN/clawfleet-server -addr :8090 -db $REMOTE_BASE/clawfleet.db -admin-user $ADMIN_USER -admin-pass '$ADMIN_PASS' -agent-key '$AGENT_KEY' > $REMOTE_BASE/server.log 2>&1 &"

echo "Deploy complete: http://$SERVER_HOST:8090"
