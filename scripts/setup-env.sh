#!/usr/bin/env bash
# Generate .env from .env.example with random secrets.
# Refuses to overwrite an existing .env unless --force is passed.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FORCE=0
if [ "${1:-}" = "--force" ]; then
  FORCE=1
fi

if [ -f "$ROOT/.env" ] && [ "$FORCE" -ne 1 ]; then
  echo ".env already exists; pass --force to regenerate it" >&2
  exit 1
fi

rand_secret() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -base64 48 | tr -d '\n'
  else
    head -c 36 /dev/urandom | base64 | tr -d '\n'
  fi
}

JWT_SECRET="$(rand_secret)"
TURN_PASSWORD="$(rand_secret)"

sed -e "s|^JWT_SECRET=.*|JWT_SECRET=$JWT_SECRET|" \
  -e "s|^TURN_PASSWORD=.*|TURN_PASSWORD=$TURN_PASSWORD|" \
  "$ROOT/.env.example" > "$ROOT/.env"
chmod 600 "$ROOT/.env"

echo "Wrote $ROOT/.env (JWT_SECRET + TURN_PASSWORD randomized, mode 600)."
echo "Review the remaining values, then: docker compose up --build -d"
