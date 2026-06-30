#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# scripts/deploy.sh — one-command docker compose deploy (P0 skeleton template).
#
# Authority: requirements/12-build-plan.md (P8 REQ-DEPLOY-07),
#            requirements/10-deployment-ops.md (REQ-OPS-02).
#
# Usage:  ./scripts/deploy.sh [API_SECRET] [ADMIN_PASSWORD]
#   API_SECRET     defaults to a freshly generated random value (openssl).
#   ADMIN_PASSWORD defaults to "admin123" (CHANGE IT after first login).
#
# Steps: tear down any existing stack, export bootstrap secrets, bring the
# stack up, then poll the frozen /health endpoint until the app is ready.
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

cd "$(dirname "$0")/.."

API_SECRET="${1:-$(openssl rand -base64 32)}"
ADMIN_PASSWORD="${2:-admin123}"

echo "==> deploying go-server-monitor"

# Stop any running stack (ignore failure if nothing is up).
docker compose down || true

# Export bootstrap secrets so docker-compose picks them up. ADMIN_USERNAME and
# other keys fall back to their .env / compose defaults.
export API_SECRET ADMIN_PASSWORD

echo "==> starting containers"
docker compose up -d --build

# Give the app a moment to start, then health-check the frozen /health endpoint.
echo "==> waiting for /health"
sleep 5
if curl -fsS http://localhost:8080/health >/dev/null 2>&1; then
  echo "==> health check passed"
else
  echo "!!! health check failed — check logs with: docker compose logs app" >&2
  exit 1
fi

echo ""
echo "✓ deploy complete"
echo "  app health : http://localhost:8080/health"
echo "  dashboard  : http://localhost/   (or https://<DOMAIN>/ via Caddy)"
echo "  admin user : ${ADMIN_USERNAME:-admin}"
echo "  NOTE: change the admin password after first login."
