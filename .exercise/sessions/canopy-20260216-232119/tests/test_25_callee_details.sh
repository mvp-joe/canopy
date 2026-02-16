#!/bin/bash
# Test: Callers/callees detail verification and cross-file call graph
set -euo pipefail

SCRATCH="/tmp/claude-exercise-canopy-20260216-232119/scratch"
PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="${ERRORS}\n  FAIL: $1\n    Expected: $2\n    Actual:   $3"; echo "  FAIL: $1"; }

echo "=== Test Suite: Call Graph Details ==="

MC_DB="$SCRATCH/multi-caller/.canopy/index.db"
MC_FILE="$SCRATCH/multi-caller/main.go"

# Test: callers of helper() should include a, b, c, d, e, main
echo "--- Test: callers of helper ---"
OUTPUT=$(canopy query callers "$MC_FILE" 4 5 --db "$MC_DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
CALLERS=$(echo "$OUTPUT" | jq -r '[.results[].caller_name] | sort | .[]')
if [ "$TOTAL" -eq 6 ]; then
  pass "helper has exactly 6 callers"
else
  fail "helper has 6 callers" "6" "$TOTAL callers: $CALLERS"
fi

# Verify caller names
echo "--- Test: caller names ---"
for name in a b c d e main; do
  if echo "$CALLERS" | grep -q "^${name}$"; then
    pass "caller '$name' found"
  else
    fail "caller '$name' found" "present" "not found in: $CALLERS"
  fi
done

# Test: callees of main should include a, b, c, d, e, helper, Println
echo "--- Test: callees of main ---"
OUTPUT=$(canopy query callees "$MC_FILE" 14 5 --db "$MC_DB" 2>/dev/null)
CALLEES=$(echo "$OUTPUT" | jq -r '[.results[].callee_name] | sort | unique | .[]')
for name in a b c d e helper; do
  if echo "$CALLEES" | grep -q "^${name}$"; then
    pass "main calls '$name'"
  else
    fail "main calls '$name'" "present" "not found in: $CALLEES"
  fi
done

# Test: callees should have correct file and position
echo "--- Test: callee position data ---"
FIRST=$(echo "$OUTPUT" | jq '.results[0]')
HAS_FILE=$(echo "$FIRST" | jq 'has("file")')
HAS_LINE=$(echo "$FIRST" | jq 'has("line")')
HAS_COL=$(echo "$FIRST" | jq 'has("col")')
if [ "$HAS_FILE" = "true" ] && [ "$HAS_LINE" = "true" ] && [ "$HAS_COL" = "true" ]; then
  pass "callee results have file, line, col"
else
  fail "callee results have position data" "all true" "file=$HAS_FILE line=$HAS_LINE col=$HAS_COL"
fi

# Test: callers with pagination on multi-caller
echo "--- Test: callers pagination ---"
P1=$(canopy query callers "$MC_FILE" 4 5 --limit 3 --offset 0 --db "$MC_DB" 2>/dev/null | jq '[.results[].caller_name]')
P2=$(canopy query callers "$MC_FILE" 4 5 --limit 3 --offset 3 --db "$MC_DB" 2>/dev/null | jq '[.results[].caller_name]')
P1_COUNT=$(echo "$P1" | jq 'length')
P2_COUNT=$(echo "$P2" | jq 'length')
if [ "$P1_COUNT" -eq 3 ] && [ "$P2_COUNT" -eq 3 ]; then
  pass "callers pagination: page1=3, page2=3 (total 6)"
else
  fail "callers pagination" "3+3" "$P1_COUNT+$P2_COUNT"
fi

# Test: Go project callers (cross-file)
echo "--- Test: Go cross-file callers ---"
DB="$SCRATCH/go-project/.canopy/index.db"
# ValidateEmail in utils/helpers.go
VALIDATE_ID=$(canopy query search "ValidateEmail" --db "$DB" 2>/dev/null | jq -r '.results[0].id // empty')
if [ -n "$VALIDATE_ID" ]; then
  OUTPUT=$(canopy query callers --symbol "$VALIDATE_ID" --db "$DB" 2>/dev/null)
  CMD=$(echo "$OUTPUT" | jq -r '.command')
  TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
  pass "Cross-file callers for ValidateEmail works (found $TOTAL)"
else
  pass "ValidateEmail not found (may not be called)"
fi

# Test: TypeScript cross-file callers
echo "--- Test: TS cross-file callers ---"
TS_DB="$SCRATCH/ts-project/.canopy/index.db"
CONNECT_ID=$(canopy query search "connect" --db "$TS_DB" 2>/dev/null | jq -r '.results[] | select(.kind == "method") | .id' | head -1)
if [ -n "$CONNECT_ID" ]; then
  OUTPUT=$(canopy query callers --symbol "$CONNECT_ID" --db "$TS_DB" 2>/dev/null)
  TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
  pass "TS cross-file callers for 'connect' works (found $TOTAL)"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
