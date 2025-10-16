#!/bin/sh

BASE_URL="http://127.0.0.1:8083"

echo "=== RBAC Service HTTP Client Test ==="
echo

echo "1. List all roles"
curl -s "${BASE_URL}/v1/roles" | jq '.'
echo
echo

echo "2. List all permissions"
curl -s "${BASE_URL}/v1/permissions" | jq '.'
echo
echo

echo "3. Get role by name (customer)"
curl -s "${BASE_URL}/v1/roles/by-name/customer" | jq '.'
echo
echo

echo "4. Get permissions for role ID 4 (customer)"
curl -s "${BASE_URL}/v1/roles/4/permissions" | jq '.'
echo
echo

echo "5. Get roles for user ID 1"
curl -s "${BASE_URL}/v1/users/1/roles" | jq '.'
echo
echo

echo "6. Get permissions for user ID 1"
curl -s "${BASE_URL}/v1/users/1/permissions" | jq '.'
echo
echo

echo "7. Check if user 1 has 'book_class' permission"
curl -s -X POST "${BASE_URL}/v1/check-permission" \
  -H "Content-Type: application/json" \
  -d '{"user_id": 1, "permission_name": "book_class"}' | jq '.'
echo
echo

echo "8. Create a new role"
curl -s -X POST "${BASE_URL}/v1/roles" \
  -H "Content-Type: application/json" \
  -d '{"name": "test_http_role", "description": "Test role created via HTTP"}' | jq '.'
echo
echo

echo "9. Create a new permission"
curl -s -X POST "${BASE_URL}/v1/permissions" \
  -H "Content-Type: application/json" \
  -d '{"name": "test_http_permission", "resource": "test", "action": "http", "description": "Test permission via HTTP"}' | jq '.'
echo
echo

echo "10. Assign role to user"
curl -s -X POST "${BASE_URL}/v1/users/1/roles/4" \
  -H "Content-Type: application/json" \
  -d '{"user_id": 1, "role_id": 4, "granted_by": 1}' | jq '.'
echo
echo

echo "=== All HTTP tests completed ==="

