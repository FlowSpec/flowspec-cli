#!/usr/bin/env bash
set -euo pipefail

CLI="${CLI:-flowspec-cli}"

ok()   { echo "[OK ] $*"; }
fail() { echo "[ERR] $*" >&2; exit 1; }

run_expect() {
  local desc="$1"; shift
  local expect="$1"; shift
  set +e
  "$CLI" verify "$@" >/dev/null 2>&1
  local code=$?
  set -e
  if [[ "$code" -eq "$expect" ]]; then
    ok "$desc → exit=$code (expect=$expect)"
  else
    fail "$desc → exit=$code (expect=$expect)"
  fi
}

echo "== E2E: Exit code matrix =="
run_expect "PASS"           0 --spec contracts/pass    --traces traces/pass.json    --ci
run_expect "FAIL_ALIGN"     1 --spec contracts/pass    --traces traces/mismatch.json --ci
run_expect "FAIL_COVERAGE"  2 --spec contracts/partial --traces traces/partial.json --ci --min-coverage 0.9
run_expect "CONFIG_ERROR"   3 --spec contracts/not_exists --traces traces/pass.json  --ci || true

echo "== E2E: Route templates & YAML priority =="
run_expect "ROUTE_TEMPLATES" 0 --spec contracts/route_templates --traces traces/routes.json --ci

echo "All e2e checks passed."
