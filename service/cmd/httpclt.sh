#!/bin/sh
set -e
BASE="${HTTP_BASE:-http://127.0.0.1:39081}"

curl -sS -d '{"user": {"email": "a@b.com", "phone": "1234567890",
            "name": "abc", "passwd": "abc" }}' \
    "$BASE/v1/create_user"
echo
curl -sS "$BASE/v1/roles"
echo
