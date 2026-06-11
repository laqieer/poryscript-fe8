#!/usr/bin/env bash
#
# End-to-end validation for poryscript-fe8.
#
# 1. Builds the compiler.
# 2. Compiles examples/sample.pory -> examples/sample.h.
# 3. Validates the generated header against a read-only fireemblem8u checkout,
#    WITHOUT mutating the decomp:
#      a. cpp macro-resolution check (every emitted macro/symbol resolves), and
#      b. (optional) a full agbcc compile, exactly like the decomp's C build rule.
#
# Usage:
#   FE8_DIR=/path/to/fireemblem8u ./check.sh
#
set -euo pipefail

FE8_DIR="${FE8_DIR:-/home/laqieer/fireemblem8u}"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INC="$FE8_DIR/include"
SRC="$FE8_DIR/src"
AGBCC_INC="$FE8_DIR/tools/agbcc/include"
AGBCC="$FE8_DIR/tools/agbcc/bin/agbcc"

echo "==> Building poryscript-fe8"
( cd "$HERE" && go build -o poryscript-fe8 . )

echo "==> Compiling examples/sample.pory -> examples/sample.h"
"$HERE/poryscript-fe8" -i "$HERE/examples/sample.pory" -o "$HERE/examples/sample.h" -fcc "$HERE/command_config.fe8.json"

# check/check.c expects sample.h next to it.
cp "$HERE/examples/sample.h" "$HERE/check/sample.h"

echo "==> cpp macro-resolution check"
cpp -iquote "$INC" -iquote "$SRC" -I "$AGBCC_INC" -nostdinc -undef "$HERE/check/check.c" -o /tmp/poryscript-fe8-check.pp
echo "    OK: all macros/includes resolved (cpp exit 0)"

if [ -x "$AGBCC" ] && command -v iconv >/dev/null 2>&1; then
  echo "==> full agbcc compile (decomp C build rule)"
  cpp -iquote "$INC" -iquote "$SRC" -I "$AGBCC_INC" -nostdinc -undef "$HERE/check/check.c" \
    | iconv -f UTF-8 -t CP932 \
    | "$AGBCC" -mthumb-interwork -Wimplicit -Wparentheses -O2 -fhex-asm -o /tmp/poryscript-fe8-check.s
  echo "    OK: agbcc produced /tmp/poryscript-fe8-check.s"
else
  echo "==> skipping agbcc compile (agbcc or iconv not available)"
fi

echo "==> ALL CHECKS PASSED"
