#!/usr/bin/env bash
set -euo pipefail

usage() {
    cat <<'EOF'
Usage: ./install.sh [DECOMP_ROOT]
       FE8_DIR=/path/to/fireemblem8u ./install.sh

Installs poryscript-fe8 into:
  DECOMP_ROOT/tools/poryscript/

DECOMP_ROOT must look like a fireemblem8u checkout. Build first with:
  go build -o poryscript-fe8 .
EOF
}

die() {
    echo "install.sh: $*" >&2
    exit 1
}

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

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
        "include/event.h"
        "include/eventinfo.h"
        "include/eventcall.h"
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

find_binary() {
    local candidate
    for candidate in \
        "$script_dir/poryscript-fe8" \
        "$script_dir/poryscript-fe8.exe" \
        "$script_dir/poryscript" \
        "$script_dir/poryscript.exe"; do
        if [ -f "$candidate" ]; then
            printf '%s\n' "$candidate"
            return 0
        fi
    done

    return 1
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
    usage
    exit 0
fi

if [ "$#" -gt 1 ]; then
    usage >&2
    die "too many arguments"
fi

decomp_root="$(resolve_decomp_root "${1:-}")"
binary="$(find_binary)" || die "could not find executable to install; run 'go build -o poryscript-fe8 .' first"

install_dir="$decomp_root/tools/poryscript"
mkdir -p "$install_dir"

binary_name="poryscript-fe8"
case "$(basename "$binary")" in
    *.exe) binary_name="poryscript-fe8.exe" ;;
esac

cp "$binary" "$install_dir/$binary_name"
chmod +x "$install_dir/$binary_name"
cp "$script_dir/font_config.json" \
   "$script_dir/command_config.json" \
   "$script_dir/command_config.fe8.json" \
   "$install_dir/"

cat > "$install_dir/README.poryscript-fe8.txt" <<EOF
poryscript-fe8 was installed here by:
  $script_dir/install.sh

Run from the fireemblem8u root with explicit local config paths, for example:
  tools/poryscript/$binary_name \\
    -cc tools/poryscript/command_config.json \\
    -fcc tools/poryscript/command_config.fe8.json \\
    -fc tools/poryscript/font_config.json \\
    -i src/events/my-event.pory \\
    -o src/events/my-event.h
EOF

echo "Installed $binary_name and configs to $install_dir"
