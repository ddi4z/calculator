#!/bin/sh
set -eu

/usr/local/bin/calculator &
backend_pid=$!

cleanup() {
    kill "$backend_pid" 2>/dev/null || true
}

trap cleanup INT TERM EXIT

nginx -g 'daemon off;'
