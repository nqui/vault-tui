#!/usr/bin/env bash
set -euo pipefail

# Creates a restricted Vault token that can't access PROD paths.
# Run this against a dev server where you have root access.
#
# Usage:
#   export VAULT_ADDR=http://127.0.0.1:8200
#   export VAULT_TOKEN=<root-token>
#   ./scripts/setup-restricted.sh
#
# It will print a restricted token at the end. Use that token with hv-tui.

: "${VAULT_ADDR:?Set VAULT_ADDR}"
: "${VAULT_TOKEN:?Set VAULT_TOKEN}"

echo "==> Creating policy 'dev-reader'..."
vault policy write dev-reader - <<'EOF'
# Allow listing secret engines
path "sys/mounts" {
  capabilities = ["read", "list"]
}

# Token self-lookup (needed for Vault client)
path "auth/token/lookup-self" {
  capabilities = ["read"]
}

# Allow full access to all paths
path "secret/data/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
path "secret/metadata/*" {
  capabilities = ["read", "list", "delete"]
}

# Deny PROD paths using + (single-segment wildcard)
# +  matches any app name (APP1, APP2, etc.)
# /* matches anything under PROD
path "secret/data/+/PROD/*" {
  capabilities = ["deny"]
}
path "secret/metadata/+/PROD/*" {
  capabilities = ["deny"]
}
# Also deny listing the PROD directory itself
path "secret/metadata/+/PROD" {
  capabilities = ["deny"]
}
EOF

echo "==> Creating restricted token..."
RESTRICTED_TOKEN=$(vault token create \
  -policy=dev-reader \
  -ttl=8h \
  -display-name="dev-reader-test" \
  -format=json | jq -r '.auth.client_token')

echo ""
echo "====================================="
echo "Restricted token (no PROD access):"
echo ""
echo "  $RESTRICTED_TOKEN"
echo ""
echo "Usage:"
echo "  export VAULT_ADDR=$VAULT_ADDR"
echo "  export VAULT_TOKEN=$RESTRICTED_TOKEN"
echo "  go run ."
echo "====================================="
