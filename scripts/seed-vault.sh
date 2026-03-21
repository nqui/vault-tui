#!/usr/bin/env bash
set -euo pipefail

# Seed a Vault dev server with test data.
# Usage:
#   vault server -dev &
#   export VAULT_ADDR=http://127.0.0.1:8200
#   export VAULT_TOKEN=<root-token>
#   ./scripts/seed-vault.sh

: "${VAULT_ADDR:?Set VAULT_ADDR}"
: "${VAULT_TOKEN:?Set VAULT_TOKEN}"

apps=(APP1 APP2 APP3)
envs=(DEV QA PROD)
projs=(PROJ1 PROJ2 PROJ3)

lc() { echo "$1" | tr '[:upper:]' '[:lower:]'; }

for app in "${apps[@]}"; do
  for env in "${envs[@]}"; do
    for proj in "${projs[@]}"; do
      path="${app}/${env}/${proj}"
      app_lc=$(lc "$app")
      env_lc=$(lc "$env")
      proj_lc=$(lc "$proj")
      echo "Writing secret/${path}"
      vault kv put "secret/${path}" \
        db_host="${app_lc}-${env_lc}-${proj_lc}.db.internal" \
        db_port="5432" \
        db_user="${proj_lc}_svc" \
        db_password="$(openssl rand -hex 12)" \
        api_key="$(openssl rand -hex 16)" \
        api_url="https://${env_lc}.${app_lc}.example.com/api/${proj_lc}" \
        log_level="$([ "$env" = PROD ] && echo warn || echo debug)" \
        feature_flag="$([ "$env" = DEV ] && echo true || echo false)"
    done
  done
done

echo ""
echo "Seeded $(( ${#apps[@]} * ${#envs[@]} * ${#projs[@]} )) secrets under secret/"
