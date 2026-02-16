#!/bin/bash
# Test: Language-specific symbol kinds and features
set -euo pipefail

SCRATCH="/tmp/claude-exercise-canopy-20260216-232119/scratch"
PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="${ERRORS}\n  FAIL: $1\n    Expected: $2\n    Actual:   $3"; echo "  FAIL: $1"; }

echo "=== Test Suite: Language-Specific Features ==="

# Go: interface, struct, method, function, package
echo "--- Go symbol kinds ---"
DB="$SCRATCH/go-project/.canopy/index.db"
KINDS=$(canopy query symbols --db "$DB" 2>/dev/null | jq -r '[.results[].kind] | unique | sort | .[]')
for kind in function interface method package struct variable; do
  if echo "$KINDS" | grep -q "^${kind}$"; then
    pass "Go has kind '$kind'"
  else
    fail "Go has kind '$kind'" "present" "not found in: $KINDS"
  fi
done

# TypeScript: class, method, function, interface, variable
echo "--- TypeScript symbol kinds ---"
TS_DB="$SCRATCH/ts-project/.canopy/index.db"
TS_KINDS=$(canopy query symbols --db "$TS_DB" 2>/dev/null | jq -r '[.results[].kind] | unique | sort | .[]')
for kind in class function method; do
  if echo "$TS_KINDS" | grep -q "^${kind}$"; then
    pass "TypeScript has kind '$kind'"
  else
    fail "TypeScript has kind '$kind'" "present" "not found in: $TS_KINDS"
  fi
done

# JavaScript: class, function, method, variable
echo "--- JavaScript symbol kinds ---"
JS_DB="$SCRATCH/js-project/.canopy/index.db"
JS_KINDS=$(canopy query symbols --db "$JS_DB" 2>/dev/null | jq -r '[.results[].kind] | unique | sort | .[]')
for kind in class function method; do
  if echo "$JS_KINDS" | grep -q "^${kind}$"; then
    pass "JavaScript has kind '$kind'"
  else
    fail "JavaScript has kind '$kind'" "present" "not found in: $JS_KINDS"
  fi
done

# Python: class, function, method, variable
echo "--- Python symbol kinds ---"
PY_DB="$SCRATCH/python-project/.canopy/index.db"
PY_KINDS=$(canopy query symbols --db "$PY_DB" 2>/dev/null | jq -r '[.results[].kind] | unique | sort | .[]')
for kind in class function method; do
  if echo "$PY_KINDS" | grep -q "^${kind}$"; then
    pass "Python has kind '$kind'"
  else
    fail "Python has kind '$kind'" "present" "not found in: $PY_KINDS"
  fi
done

# Rust: function, struct, method, trait
echo "--- Rust symbol kinds ---"
RUST_DB="$SCRATCH/rust-project/.canopy/index.db"
RUST_KINDS=$(canopy query symbols --db "$RUST_DB" 2>/dev/null | jq -r '[.results[].kind] | unique | sort | .[]')
for kind in function struct method trait; do
  if echo "$RUST_KINDS" | grep -q "^${kind}$"; then
    pass "Rust has kind '$kind'"
  else
    fail "Rust has kind '$kind'" "present" "not found in: $RUST_KINDS"
  fi
done

# C: function, variable
echo "--- C symbol kinds ---"
C_DB="$SCRATCH/c-project/.canopy/index.db"
C_KINDS=$(canopy query symbols --db "$C_DB" 2>/dev/null | jq -r '[.results[].kind] | unique | sort | .[]')
if echo "$C_KINDS" | grep -q "function"; then
  pass "C has kind 'function'"
else
  fail "C has kind 'function'" "present" "not found in: $C_KINDS"
fi

# C++: class, function, method
echo "--- C++ symbol kinds ---"
CPP_DB="$SCRATCH/cpp-project/.canopy/index.db"
CPP_KINDS=$(canopy query symbols --db "$CPP_DB" 2>/dev/null | jq -r '[.results[].kind] | unique | sort | .[]')
for kind in class function method; do
  if echo "$CPP_KINDS" | grep -q "^${kind}$"; then
    pass "C++ has kind '$kind'"
  else
    fail "C++ has kind '$kind'" "present" "not found in: $CPP_KINDS"
  fi
done

# Java: class, method
echo "--- Java symbol kinds ---"
JAVA_DB="$SCRATCH/java-project/.canopy/index.db"
JAVA_KINDS=$(canopy query symbols --db "$JAVA_DB" 2>/dev/null | jq -r '[.results[].kind] | unique | sort | .[]')
for kind in class method; do
  if echo "$JAVA_KINDS" | grep -q "^${kind}$"; then
    pass "Java has kind '$kind'"
  else
    fail "Java has kind '$kind'" "present" "not found in: $JAVA_KINDS"
  fi
done

# PHP: class, function, method
echo "--- PHP symbol kinds ---"
PHP_DB="$SCRATCH/php-project/.canopy/index.db"
PHP_KINDS=$(canopy query symbols --db "$PHP_DB" 2>/dev/null | jq -r '[.results[].kind] | unique | sort | .[]')
for kind in class function method; do
  if echo "$PHP_KINDS" | grep -q "^${kind}$"; then
    pass "PHP has kind '$kind'"
  else
    fail "PHP has kind '$kind'" "present" "not found in: $PHP_KINDS"
  fi
done

# Ruby: class, method, module
echo "--- Ruby symbol kinds ---"
RUBY_DB="$SCRATCH/ruby-project/.canopy/index.db"
RUBY_KINDS=$(canopy query symbols --db "$RUBY_DB" 2>/dev/null | jq -r '[.results[].kind] | unique | sort | .[]')
for kind in class method module; do
  if echo "$RUBY_KINDS" | grep -q "^${kind}$"; then
    pass "Ruby has kind '$kind'"
  else
    fail "Ruby has kind '$kind'" "present" "not found in: $RUBY_KINDS"
  fi
done

# Test: Go visibility - exported (public) vs unexported (private)
echo "--- Go visibility ---"
PUB=$(canopy query symbols --visibility public --db "$DB" 2>/dev/null | jq -r '.total_count')
PRIV=$(canopy query symbols --visibility private --db "$DB" 2>/dev/null | jq -r '.total_count')
TOTAL=$(canopy query symbols --db "$DB" 2>/dev/null | jq -r '.total_count')
SUM=$((PUB + PRIV))
if [ "$SUM" -eq "$TOTAL" ]; then
  pass "Go public + private = total symbols ($PUB + $PRIV = $TOTAL)"
else
  fail "Go public + private = total symbols" "$TOTAL" "$SUM ($PUB + $PRIV)"
fi

# Test: TypeScript visibility
echo "--- TypeScript visibility ---"
PUB=$(canopy query symbols --visibility public --db "$TS_DB" 2>/dev/null | jq -r '.total_count')
PRIV=$(canopy query symbols --visibility private --db "$TS_DB" 2>/dev/null | jq -r '.total_count')
TOTAL=$(canopy query symbols --db "$TS_DB" 2>/dev/null | jq -r '.total_count')
SUM=$((PUB + PRIV))
if [ "$SUM" -eq "$TOTAL" ]; then
  pass "TS public + private = total symbols ($PUB + $PRIV = $TOTAL)"
else
  fail "TS public + private = total symbols" "$TOTAL" "$SUM ($PUB + $PRIV)"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
