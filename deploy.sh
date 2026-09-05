#!/usr/bin/env bash
# One-shot deploy for a fresh Ubuntu server (run as root or with sudo).
# Installs Docker if missing, then builds & starts the full stack on port 80.
set -euo pipefail

REPO="${REPO:-https://github.com/ggalperine/election-2027.git}"
DIR="${DIR:-/opt/election-2027}"

echo "==> Installing Docker (if needed)"
if ! command -v docker >/dev/null 2>&1; then
  curl -fsSL https://get.docker.com | sh
fi

echo "==> Fetching source into $DIR"
if [ -d "$DIR/.git" ]; then
  git -C "$DIR" pull --ff-only
else
  git clone "$REPO" "$DIR"
fi
cd "$DIR"

echo "==> Building & starting stack (port 80)"
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build

echo "==> Waiting for the API to come up"
for i in $(seq 1 30); do
  if curl -fsS localhost/api/cycles >/dev/null 2>&1; then
    echo "OK — up after ${i} tries"; break
  fi
  sleep 3
done

echo "==> Done. App: http://$(curl -fsS ifconfig.me 2>/dev/null || echo SERVER_IP)/"
docker compose -f docker-compose.yml -f docker-compose.prod.yml ps
