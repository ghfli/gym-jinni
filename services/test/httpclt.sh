#!/bin/sh

curl -d '{"user": {"email": "a@b.com", "name": "abc"}}' \
    http://127.0.0.1:8081/v1/create_user
echo
