#!/usr/bin/env bash
# Runs ShopBot on this machine without Docker for the app itself:
#   - builds the Mini App if needed,
#   - starts a Cloudflare quick tunnel (in Docker) for the HTTPS address,
#   - runs the Go server, which finds the tunnel address and sets up the bot.
set -euo pipefail
cd "$(dirname "$0")/.."

if [ -f .env ]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

if [ ! -f webapp/dist/index.html ]; then
  (cd webapp && npm ci --no-audit --no-fund && npm run build)
fi

METRICS=127.0.0.1:20241
docker rm -f shopbot-tunnel >/dev/null 2>&1 || true
docker run -d --rm --name shopbot-tunnel --network host cloudflare/cloudflared:2026.10.3 \
  tunnel --no-autoupdate --metrics "$METRICS" --url http://127.0.0.1:8080 >/dev/null
trap 'docker stop shopbot-tunnel >/dev/null 2>&1 || true' EXIT

cd backend
TUNNEL_METRICS_URL="http://$METRICS" DB_PATH=../data/shopbot.db WEB_DIR=../webapp/dist go run ./cmd/shopbot
