#!/bin/sh
set -e
BASE="${HTTP_BASE:-http://127.0.0.1:8081}"

echo "=== Create user ==="
CREATE_RESP=$(curl -sS -d "{\"user\": {\"email\": \"httpclt$(date +%s)@example.com\",
    \"phone\": \"1234567890\", \"name\": \"httpclt\", \"passwd\": \"abc\" }}" \
    "$BASE/v1/create_user")
echo "$CREATE_RESP"

echo "=== Login ==="
LOGIN_RESP=$(curl -sS -d "{\"user\": {\"email\": \"httpclt$(date +%s)@example.com\",
    \"phone\": \"123456789\", \"name\": \"user\", \"passwd\": \"abc\" }}" \
    "$BASE/v1/login_user" 2>/dev/null || echo '{}')
echo "$LOGIN_RESP"

# Use a dummy token for authenticated calls if login failed in standalone test
TOKEN=$(echo "$LOGIN_RESP" | grep -o '"access_tkn":"[^"]*"' | cut -d'"' -f4 || echo "")
AUTH_HEADER=""
if [ -n "$TOKEN" ]; then
    AUTH_HEADER="-H \"Authorization: Bearer $TOKEN\""
fi

echo "=== List roles ==="
eval curl -sS $AUTH_HEADER "$BASE/v1/roles"
echo

echo "=== List gyms ==="
eval curl -sS $AUTH_HEADER "$BASE/v1/gyms"
echo

echo "=== Dashboard report ==="
eval curl -sS $AUTH_HEADER "$BASE/v1/reports/dashboard"
echo

echo "=== List trainers ==="
eval curl -sS $AUTH_HEADER "$BASE/v1/trainers"
echo
