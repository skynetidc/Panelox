#!/bin/bash
# Panelox - API Test Script
# Tests all major API endpoints
# Usage: ./scripts/test-api.sh [API_URL]

API="${1:-http://localhost:8080/api/v1}"
PASS=0
FAIL=0

test_endpoint() {
  local method="$1"
  local path="$2"
  local data="$3"
  local expected_status="${4:-200}"
  local desc="$5"

  local args=(-s -o /tmp/panelox-test-response -w "%{http_code}" -X "$method")
  [ -n "$TOKEN" ] && args+=(-H "Authorization: Bearer $TOKEN")
  args+=(-H "Content-Type: application/json")
  [ -n "$data" ] && args+=(-d "$data")
  args+=("$API$path")

  local status
  status=$(curl "${args[@]}")

  if [ "$status" = "$expected_status" ]; then
    echo "  ✓ $method $path ($status) $desc"
    PASS=$((PASS + 1))
  else
    echo "  ✗ $method $path (got $status, expected $expected_status) $desc"
    FAIL=$((FAIL + 1))
  fi
}

echo "============================================"
echo "  Panelox API Test Suite"
echo "  API: $API"
echo "============================================"
echo ""

# --- Health ---
echo "[Health]"
test_endpoint GET "/../health" "" 200 "health check"

# --- Auth ---
echo ""
echo "[Auth]"
test_endpoint POST "/auth/register" '{"email":"test-api@panelox.io","password":"test123","name":"API Test"}' 201 "register"
test_endpoint POST "/auth/login" '{"email":"test-api@panelox.io","password":"test123"}' 200 "login"

# Extract token
LOGIN_RESP=$(curl -s -X POST "$API/auth/login" -H "Content-Type: application/json" -d '{"email":"test-api@panelox.io","password":"test123"}')
TOKEN=$(echo "$LOGIN_RESP" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "  ✗ Failed to get token. Cannot continue."
  exit 1
fi
echo "  Token acquired."

test_endpoint GET "/auth/me" "" 200 "get current user"
test_endpoint POST "/auth/login" '{"email":"wrong@test.com","password":"wrong"}' 401 "reject bad login"

# --- Dashboard ---
echo ""
echo "[Dashboard]"
test_endpoint GET "/dashboard/overview" "" 200 "overview"
test_endpoint GET "/dashboard/stats" "" 200 "system stats"

# --- Websites ---
echo ""
echo "[Websites]"
test_endpoint POST "/websites" '{"domain":"api-test.example.com","type":"static"}' 201 "create website"
test_endpoint GET "/websites?page=1&size=10" "" 200 "list websites"

# Get website ID
WEB_ID=$(curl -s "$API/websites?page=1&size=10" -H "Authorization: Bearer $TOKEN" | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2)
if [ -n "$WEB_ID" ]; then
  test_endpoint GET "/websites/$WEB_ID" "" 200 "get website"
  test_endpoint GET "/websites/$WEB_ID/domains" "" 200 "list domains"
fi

# --- SSL ---
echo ""
echo "[SSL]"
test_endpoint GET "/ssl?page=1&size=10" "" 200 "list certificates"
test_endpoint GET "/acme-accounts" "" 200 "list ACME accounts"

# --- DNS ---
echo ""
echo "[DNS]"
test_endpoint POST "/dns/zones" '{"name":"apitest.example.com"}' 201 "create zone"
test_endpoint GET "/dns/zones?page=1&size=10" "" 200 "list zones"

ZONE_ID=$(curl -s "$API/dns/zones?page=1&size=10" -H "Authorization: Bearer $TOKEN" | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2)
if [ -n "$ZONE_ID" ]; then
  test_endpoint GET "/dns/zones/$ZONE_ID" "" 200 "get zone"
  test_endpoint POST "/dns/zones/$ZONE_ID/records" '{"name":"www","type":"A","content":"1.2.3.4"}' 201 "create record"
  test_endpoint GET "/dns/zones/$ZONE_ID/records" "" 200 "list records"
fi

# --- Mail ---
echo ""
echo "[Mail]"
test_endpoint GET "/mail/domains?page=1&size=10" "" 200 "list mail domains"

# --- Databases ---
echo ""
echo "[Databases]"
test_endpoint GET "/databases/servers?page=1&size=10" "" 200 "list DB servers"

# --- Firewall ---
echo ""
echo "[Firewall]"
test_endpoint GET "/firewall/status" "" 200 "firewall status"
test_endpoint GET "/firewall/rules" "" 200 "list rules"

# --- Files ---
echo ""
echo "[File Manager]"
test_endpoint GET "/files?path=/tmp" "" 200 "list /tmp"

# --- Containers ---
echo ""
echo "[Containers]"
test_endpoint GET "/containers" "" 200 "list containers"

# --- Docker ---
echo ""
echo "[Docker Resources]"
test_endpoint GET "/docker/networks" "" 200 "list networks"
test_endpoint GET "/docker/volumes" "" 200 "list volumes"
test_endpoint GET "/docker/images" "" 200 "list images"

# --- Compose ---
echo ""
echo "[Compose]"
test_endpoint GET "/compose" "" 200 "list compose projects"

# --- App Store ---
echo ""
echo "[App Store]"
test_endpoint GET "/apps/catalog" "" 200 "app catalog"
test_endpoint GET "/apps/installed" "" 200 "installed apps"

# --- Cron Jobs ---
echo ""
echo "[Cron Jobs]"
test_endpoint GET "/cronjobs" "" 200 "list cron jobs"
test_endpoint POST "/cronjobs" '{"name":"test-job","schedule":"@daily","command":"echo hello"}' 201 "create cron job"

# --- Backups ---
echo ""
echo "[Backups]"
test_endpoint GET "/backups?page=1&size=10" "" 200 "list backups"
test_endpoint GET "/backup-accounts" "" 200 "list backup accounts"
test_endpoint GET "/backup-schedules" "" 200 "list backup schedules"

# --- Monitoring ---
echo ""
echo "[Monitoring]"
test_endpoint GET "/monitoring/current" "" 200 "current metrics"
test_endpoint GET "/monitoring/history?hours=1" "" 200 "history"
test_endpoint GET "/monitoring/containers" "" 200 "container metrics"
test_endpoint GET "/monitoring/network" "" 200 "network stats"
test_endpoint GET "/monitoring/disk-io" "" 200 "disk I/O"
test_endpoint GET "/monitoring/bandwidth" "" 200 "website bandwidth"

# --- Security ---
echo ""
echo "[Security]"
test_endpoint GET "/login-log?page=1&size=10" "" 200 "login log"
test_endpoint GET "/2fa/status" "" 200 "2FA status"
test_endpoint GET "/fail2ban/rules" "" 200 "fail2ban rules"
test_endpoint GET "/fail2ban/banned" "" 200 "banned IPs"

# --- Notifications ---
echo ""
echo "[Notifications]"
test_endpoint GET "/notifications/channels" "" 200 "list channels"
test_endpoint GET "/alerts/rules" "" 200 "alert rules"

# --- Snapshots ---
echo ""
echo "[Snapshots]"
test_endpoint GET "/snapshots" "" 200 "list snapshots"

# --- Git Deploy ---
echo ""
echo "[Git Deploy]"
test_endpoint GET "/git-deploy" "" 200 "list deploys"

# --- Webhooks ---
echo ""
echo "[Webhooks]"
test_endpoint GET "/webhooks" "" 200 "list webhooks"
test_endpoint GET "/webhooks/events" "" 200 "event types"

# --- Billing ---
echo ""
echo "[Billing]"
test_endpoint GET "/billing/usage" "" 200 "current usage"
test_endpoint GET "/users/billing" "" 200 "billing dashboard"

# --- Quotas ---
echo ""
echo "[Quotas]"
test_endpoint GET "/quotas/my" "" 200 "my quota"
test_endpoint GET "/packages" "" 200 "list packages"

# --- Settings ---
echo ""
echo "[Settings]"
test_endpoint GET "/settings" "" 200 "get settings"
test_endpoint GET "/audit?page=1&size=10" "" 200 "audit log"
test_endpoint GET "/update/check" "" 200 "update check"

# --- PHP ---
echo ""
echo "[PHP Runtimes]"
test_endpoint GET "/php/versions" "" 200 "PHP versions"
test_endpoint GET "/php/runtimes" "" 200 "installed runtimes"

# --- FTP ---
echo ""
echo "[FTP]"
test_endpoint GET "/ftp/users" "" 200 "FTP users"

# --- Staging ---
echo ""
echo "[Staging]"
test_endpoint GET "/staging" "" 200 "list staging"

# --- Results ---
echo ""
echo "============================================"
echo "  Results: $PASS passed, $FAIL failed"
echo "  Total:   $((PASS + FAIL)) tests"
echo "============================================"

if [ $FAIL -gt 0 ]; then
  exit 1
fi
