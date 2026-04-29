#!/bin/sh

set -e

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)

case "$1" in
  install)
    (
      cd "$SCRIPT_DIR" || exit 1
      curl --proto '=https' --tlsv1.2 -fsSL https://drop-sh.fullyjustified.net | sh
    )
    # fontconfig.tex erzeugen
    cat > "$SCRIPT_DIR/texmf/tex/latex/bflatex/fontconfig.tex" << EOF
\useTeXTreetrue
\newcommand{\fontpath}{$SCRIPT_DIR/texmf/}
EOF

    ;;
  *.tex)
    TEX_FILE=$(cd "$(dirname "$1")" && pwd)/$(basename "$1")
    OUT_DIR=$(dirname "$TEX_FILE")

    if [ ! -f "$SCRIPT_DIR/tectonic" ]; then
      echo "Error: tectonic not found at $SCRIPT_DIR/tectonic" >&2
      exit 1
    fi

    SEARCH_ARGS=""
    while IFS= read -r dir; do
      SEARCH_ARGS="$SEARCH_ARGS -Z search-path=$dir"
    done < <(find "$SCRIPT_DIR/texmf" -type d)

    TECTONIC_CACHE_DIR="$SCRIPT_DIR/.tectonic-cache" \
      "$SCRIPT_DIR/tectonic" \
          $SEARCH_ARGS \
          -p \
          -o "$OUT_DIR" \
      "$TEX_FILE"
    ;;
  clean)
    find "$SCRIPT_DIR" -type f -name "*.log" -o -name "*.aux" -o -name "*.toc" -o -name "*.out" | xargs rm -f
    rm -rf "$SCRIPT_DIR/.tectonic-cache"
    ;;
  *)
    cat <<EOF
Usage: $0 <command> [file.tex]
Commands:
  install       Install TeX distribution
  clean         Remove build artifacts and cache
  *.tex         Compile the specified TeX file
EOF
    exit 1
    ;;
esac