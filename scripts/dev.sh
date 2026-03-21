#!/usr/bin/env bash
set -euo pipefail

# Start a Vault dev server, seed it with sample data, and run hv-tui.
# Usage: ./scripts/dev.sh [--restricted]
#
# Flags:
#   --restricted   Also create a restricted token (no PROD access) and use that instead.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RESTRICTED=false

for arg in "$@"; do
  case "$arg" in
    --restricted) RESTRICTED=true ;;
    *) echo "Unknown flag: $arg"; exit 1 ;;
  esac
done

cleanup() {
  if [[ -n "${VAULT_PID:-}" ]]; then
    echo ""
    echo "==> Stopping Vault dev server (pid $VAULT_PID)..."
    kill "$VAULT_PID" 2>/dev/null || true
    wait "$VAULT_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

# Start Vault dev server
echo "==> Starting Vault dev server..."
VAULT_LOG=$(mktemp)
vault server -dev > "$VAULT_LOG" 2>&1 &
VAULT_PID=$!

# Wait for Vault to be ready
export VAULT_ADDR="http://127.0.0.1:8200"
for i in $(seq 1 30); do
  if vault status > /dev/null 2>&1; then
    break
  fi
  if ! kill -0 "$VAULT_PID" 2>/dev/null; then
    echo "Vault failed to start:"
    cat "$VAULT_LOG"
    exit 1
  fi
  sleep 0.2
done

# Extract root token from dev server output
ROOT_TOKEN=$(grep 'Root Token:' "$VAULT_LOG" | awk '{print $NF}')
if [[ -z "$ROOT_TOKEN" ]]; then
  echo "Failed to extract root token from Vault output:"
  cat "$VAULT_LOG"
  exit 1
fi
export VAULT_TOKEN="$ROOT_TOKEN"
rm -f "$VAULT_LOG"

echo "==> Vault running at $VAULT_ADDR"
echo "==> Root token: $ROOT_TOKEN"
echo ""

# Seed sample secrets
"$SCRIPT_DIR/seed-vault.sh"

# Optionally create restricted token
USE_TOKEN="$ROOT_TOKEN"
if [[ "$RESTRICTED" == "true" ]]; then
  echo ""
  "$SCRIPT_DIR/setup-restricted.sh"
  USE_TOKEN=$(vault token create \
    -policy=dev-reader \
    -ttl=8h \
    -display-name="dev-reader-test" \
    -format=json | jq -r '.auth.client_token')
  echo "==> Using restricted token: $USE_TOKEN"
fi

echo ""
echo "====================================="
echo "  Vault Address: $VAULT_ADDR"
echo "  Token:         $USE_TOKEN"
echo "====================================="
echo ""
read -r -p "Press enter to launch hv-tui..."

# Run hv-tui with only the address — let the login form handle auth
unset VAULT_TOKEN
go run .
