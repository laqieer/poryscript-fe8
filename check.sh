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
#   ./check.sh /path/to/fireemblem8u
#   FE8_DIR=/path/to/fireemblem8u ./check.sh
#
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

usage() {
  cat <<'EOF'
Usage: ./check.sh [DECOMP_ROOT]
       FE8_DIR=/path/to/fireemblem8u ./check.sh

DECOMP_ROOT must be a fireemblem8u checkout. The check is read-only with
respect to the decomp root.
EOF
}

die() {
  echo "check.sh: $*" >&2
  exit 1
}

is_decomp_root() {
  local root="$1"
  [ -f "$root/include/EAstdlib.h" ] &&
    [ -f "$root/include/global.h" ] &&
    [ -f "$root/include/event.h" ] &&
    [ -d "$root/src/events" ] &&
    [ -f "$root/Makefile" ]
}

find_decomp_root_upward() {
  local dir
  dir="$(cd "$1" 2>/dev/null && pwd)" || return 1

  while true; do
    if is_decomp_root "$dir"; then
      printf '%s\n' "$dir"
      return 0
    fi
    [ "$dir" = "/" ] && return 1
    dir="$(dirname "$dir")"
  done
}

validate_decomp_root() {
  local root="$1"
  local missing=0
  local required=(
    "include/EAstdlib.h"
    "include/global.h"
    "include/bmunit.h"
    "include/event.h"
    "include/eventinfo.h"
    "include/eventcall.h"
    "include/constants/characters.h"
    "include/constants/songs.h"
    "src/events"
    "Makefile"
  )

  for rel in "${required[@]}"; do
    if [ ! -e "$root/$rel" ]; then
      echo "  missing: $rel" >&2
      missing=1
    fi
  done

  if [ "$missing" -ne 0 ]; then
    die "'$root' does not look like a fireemblem8u decomp root"
  fi
}

resolve_decomp_root() {
  local candidate="${1:-${FE8_DIR:-}}"
  local root

  if [ -z "$candidate" ]; then
    if root="$(find_decomp_root_upward "$PWD")"; then
      validate_decomp_root "$root"
      printf '%s\n' "$root"
      return 0
    elif root="$(find_decomp_root_upward "$HERE/..")"; then
      validate_decomp_root "$root"
      printf '%s\n' "$root"
      return 0
    else
      usage >&2
      die "pass DECOMP_ROOT or set FE8_DIR"
    fi
  fi

  root="$(find_decomp_root_upward "$candidate")" ||
    die "cannot find a fireemblem8u decomp root at or above '$candidate'"
  validate_decomp_root "$root"
  printf '%s\n' "$root"
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

if [ "$#" -gt 1 ]; then
  usage >&2
  die "too many arguments"
fi

FE8_DIR="$(resolve_decomp_root "${1:-}")"
INC="$FE8_DIR/include"
SRC="$FE8_DIR/src"
AGBCC_INC="$FE8_DIR/tools/agbcc/include"
AGBCC="$FE8_DIR/tools/agbcc/bin/agbcc"
OUT_DIR="$HERE/.check"

mkdir -p "$OUT_DIR"

echo "==> Using fireemblem8u decomp root: $FE8_DIR"

echo "==> Building poryscript-fe8"
( cd "$HERE" && go build -o poryscript-fe8 . )

echo "==> Compiling examples/sample.pory -> examples/sample.h"
"$HERE/poryscript-fe8" -i "$HERE/examples/sample.pory" -o "$HERE/examples/sample.h" -fcc "$HERE/command_config.fe8.json"

# check/check.c expects sample.h next to it.
cp "$HERE/examples/sample.h" "$HERE/check/sample.h"

echo "==> cpp macro-resolution check"
cpp -iquote "$INC" -iquote "$SRC" -I "$AGBCC_INC" -nostdinc -undef "$HERE/check/check.c" -o "$OUT_DIR/poryscript-fe8-check.pp"
echo "    OK: all macros/includes resolved (cpp exit 0)"

if [ -x "$AGBCC" ] && command -v iconv >/dev/null 2>&1; then
  echo "==> full agbcc compile (decomp C build rule)"
  cpp -iquote "$INC" -iquote "$SRC" -I "$AGBCC_INC" -nostdinc -undef "$HERE/check/check.c" \
    | iconv -f UTF-8 -t CP932 \
    | "$AGBCC" -mthumb-interwork -Wimplicit -Wparentheses -O2 -fhex-asm -o "$OUT_DIR/poryscript-fe8-check.s"
  echo "    OK: agbcc produced $OUT_DIR/poryscript-fe8-check.s"
else
  echo "==> skipping agbcc compile (agbcc or iconv not available)"
fi

echo "==> ALL CHECKS PASSED"
