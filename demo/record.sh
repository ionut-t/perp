#!/usr/bin/env bash
# Regenerates assets/demo.gif.
#
#   ./demo/record.sh
#
# Starts a throwaway PostgreSQL (demo/docker-compose.yml), builds perp, runs it
# with an isolated HOME under demo/.tmp (so no real config, history or
# connections are used), records demo/demo.tape with VHS, then tears it all down.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEMO="$ROOT/demo"
TMP="$DEMO/.tmp"
COMPOSE=(docker compose -f "$DEMO/docker-compose.yml")

for bin in docker go vhs; do
	command -v "$bin" >/dev/null || { echo "error: '$bin' is required" >&2; exit 1; }
done

cleanup() {
	"${COMPOSE[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
	rm -rf "$TMP"
}
trap cleanup EXIT

# Start fresh every time.
cleanup
mkdir -p "$TMP/bin" "$TMP/home/.perp" "$TMP/home/.config" "$ROOT/assets"

echo "==> Starting demo database"
"${COMPOSE[@]}" up -d --wait

echo "==> Building perp"
# Stamp the latest release tag so the banner doesn't show a "+dirty" VCS version.
VERSION="$(git -C "$ROOT" describe --tags --abbrev=0 2>/dev/null || echo dev)"
(cd "$ROOT" && go build -buildvcs=false \
	-ldflags "-X github.com/ionut-t/perp/internal/version.version=$VERSION" \
	-o "$TMP/bin/perp" .)

# Swallow clipboard writes so yanking in the demo doesn't touch the real clipboard.
for shim in pbcopy xclip xsel wl-copy; do
	printf '#!/bin/sh\ncat >/dev/null\n' >"$TMP/bin/$shim"
	chmod +x "$TMP/bin/$shim"
done

cat >"$TMP/home/.perp/.config.toml" <<'EOF'
auto_update = false
update_check_interval = 24
editor = "vi"
max_history_length = 1000
max_history_days = 90
llm_provider = ""
llm_model = ""
leader_key = " "
EOF

cat >"$TMP/home/.perp/servers.json" <<'EOF'
[
  {
    "id": "00000000-0000-4000-8000-000000000001",
    "name": "bookshop",
    "address": "localhost",
    "port": 55432,
    "database": "bookshop",
    "username": "demo",
    "password": "demo",
    "createdAt": "2026-01-01T09:00:00Z",
    "updatedAt": "2026-01-01T09:00:00Z",
    "shareDatabaseSchemaLLM": false,
    "timingEnabled": false
  }
]
EOF

echo "==> Recording"
(cd "$ROOT" && vhs demo/demo.tape)

echo "==> Wrote assets/demo.gif"
