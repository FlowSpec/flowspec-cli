#!/usr/bin/env bash
set -euo pipefail

CLI="${CLI:-flowspec}"

tmp1="$(mktemp -d)"
tmp2="$(mktemp -d)"

# First run
"$CLI" validate --spec contracts/pass --traces traces/pass.json --ci --format json  > "$tmp1/report.json"
"$CLI" validate --spec contracts/pass --traces traces/pass.json --ci --format human > "$tmp1/report.md"

# Second run
"$CLI" validate --spec contracts/pass --traces traces/pass.json --ci --format json  > "$tmp2/report.json"
"$CLI" validate --spec contracts/pass --traces traces/pass.json --ci --format human > "$tmp2/report.md"

# Compare
if command -v diff >/dev/null 2>&1; then
  diff -q "$tmp1/report.json" "$tmp2/report.json"
  diff -q "$tmp1/report.md"   "$tmp2/report.md"
else
  echo "diff not found; falling back to sha256"
  if command -v sha256sum >/dev/null 2>&1; then
    cmp1=$(sha256sum "$tmp1/report.json" | cut -d' ' -f1)
    cmp2=$(sha256sum "$tmp2/report.json" | cut -d' ' -f1)
    [[ "$cmp1" == "$cmp2" ]] || { echo "JSON differs"; exit 1; }
    cmp1=$(sha256sum "$tmp1/report.md" | cut -d' ' -f1)
    cmp2=$(sha256sum "$tmp2/report.md" | cut -d' ' -f1)
    [[ "$cmp1" == "$cmp2" ]] || { echo "HUMAN differs"; exit 1; }
  else
    echo "Install diff or sha256sum to run this check." >&2
    exit 2
  fi
fi

echo "Golden determinism check: ALL MATCH"
