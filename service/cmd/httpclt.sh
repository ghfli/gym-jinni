#!/bin/sh

curl -d '{"user": {"email": "a@b.com", "phone": "1234567890",
            "name": "abc", "passwd": "abc" }}' \
    http://127.0.0.1:8081/v1/create_user
echo
