#!/bin/bash
# Panelox - Seed test data
# Usage: ./scripts/seed.sh [API_URL]

API="${1:-http://localhost:8080/api/v1}"
echo "Seeding Panelox at $API..."

# 1. Register admin
echo "--- Creating admin user ---"
REGISTER=$(curl -s -X POST "$API/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@panelox.io","password":"admin123","name":"Admin"}')
echo "$REGISTER" | head -c 200
echo ""

# 2. Login
echo "--- Logging in ---"
LOGIN=$(curl -s -X POST "$API/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@panelox.io","password":"admin123"}')
TOKEN=$(echo "$LOGIN" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "Login failed! Response:"
  echo "$LOGIN"
  exit 1
fi
echo "Token: ${TOKEN:0:20}..."

AUTH="Authorization: Bearer $TOKEN"

# 3. Check dashboard
echo ""
echo "--- Dashboard ---"
curl -s "$API/dashboard/overview" -H "$AUTH" | python3 -m json.tool 2>/dev/null || curl -s "$API/dashboard/overview" -H "$AUTH"
echo ""

# 4. Create test website
echo ""
echo "--- Creating test website ---"
curl -s -X POST "$API/websites" \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"domain":"test.example.com","type":"static","remark":"Test website"}' | head -c 200
echo ""

# 5. Create DNS zone
echo ""
echo "--- Creating DNS zone ---"
curl -s -X POST "$API/dns/zones" \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"name":"example.com"}' | head -c 200
echo ""

# 6. Create test user
echo ""
echo "--- Creating test user ---"
curl -s -X POST "$API/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@panelox.io","password":"user123","name":"Test User"}' | head -c 200
echo ""

# 7. Health check
echo ""
echo "--- Health check ---"
curl -s "$API/../health"
echo ""

# 8. System stats
echo ""
echo "--- Monitoring ---"
curl -s "$API/monitoring/current" -H "$AUTH" | head -c 300
echo ""

echo ""
echo "============================================"
echo " Seed complete!"
echo " Admin: admin@panelox.io / admin123"
echo " User:  user@panelox.io / user123"
echo " API:   $API"
echo "============================================"
