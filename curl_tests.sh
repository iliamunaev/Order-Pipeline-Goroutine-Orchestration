#!/usr/bin/env bash
set -euo pipefail

base_url="${1:-http://localhost:8080}"
server_pid=""

pass() { printf "PASS: %s\n" "$1"; }
fail() { printf "FAIL: %s\n" "$1" >&2; exit 1; }

tmp_body="$(mktemp)"
cleanup() {
  if [[ -n "$server_pid" ]]; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  fi
  rm -f "$tmp_body"
}
trap cleanup EXIT

start_server() {
  if command -v go >/dev/null 2>&1; then
    go run ./cmd/server >/dev/null 2>&1 &
    server_pid="$!"
  else
    fail "go command not found; start the server manually"
  fi

  for _ in {1..50}; do
    if curl -s "$base_url/health" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.1
  done
  fail "server did not become ready"
}

start_server

status="$(curl -s -o "$tmp_body" -w "%{http_code}" "$base_url/health")"
if [[ "$status" != "200" ]]; then
  fail "/health expected 200, got $status"
fi
if [[ "$(cat "$tmp_body")" != "ok" ]]; then
  fail "/health expected body ok, got $(cat "$tmp_body")"
fi
pass "/health"

payload='{"order_id":"o-1","amount":1200,"delay_ms":{"payment":1,"vendor":1,"courier":1}}'
status="$(curl -s -o "$tmp_body" -w "%{http_code}" \
  -H "Content-Type: application/json" \
  -d "$payload" \
  "$base_url/order")"
if [[ "$status" != "200" ]]; then
  fail "/order expected 200, got $status"
fi
if ! grep -q '"status":"ok"' "$tmp_body"; then
  fail "/order expected status ok, got $(cat "$tmp_body")"
fi
pass "/order"
