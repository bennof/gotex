#!/bin/sh

set -e

GOTEX_PATH=${GOTEX_PATH:-"$HOME/.gotex"}
GOTEX_BIN=${GOTEX_BIN:-"$GOTEX_PATH/bin"}
GOTEX_TEXMF=${GOTEX_TEXMF:-"$GOTEX_PATH"}
TECTONIC_BIN="$GOTEX_BIN/tectonic"
TEXMF_ROOT="$GOTEX_TEXMF/texmf"
CACHE_DIR="$GOTEX_TEXMF/.tectonic-cache"

usage() {
  cat <<EOUSAGE
Usage: $0 <command> [args]

Commands:
  install          Install tectonic and local texmf data into $GOTEX_PATH
  clean            Remove temporary logs and cache from $GOTEX_TEXMF
  <tectonic args>  Run tectonic from $TECTONIC_BIN with gotex search paths

Environment:
  GOTEX_PATH   Base path for gotex data (default: $HOME/.gotex)
  GOTEX_BIN    Directory with tectonic binary (default: $GOTEX_PATH/bin)
  GOTEX_TEXMF  Base path for the local TeX tree (default: $GOTEX_PATH)
EOUSAGE
}

install() {
  mkdir -p "$GOTEX_BIN"
  mkdir -p "$TEXMF_ROOT/tex/latex/bflatex"

  (cd "$GOTEX_BIN" || exit 1
    curl --proto '=https' --tlsv1.2 -fsSL https://drop-sh.fullyjustified.net | sh
  )

  cat > "$TEXMF_ROOT/tex/latex/bflatex/fontconfig.tex" <<EOF
\\useTeXTreetrue
\\newcommand{\\fontpath}{$TEXMF_ROOT/}
EOF

  echo "Installed tectonic to $TECTONIC_BIN"
  echo "Created fontconfig at $TEXMF_ROOT/tex/latex/bflatex/fontconfig.tex"
}

ensure_tectonic() {
  if [ ! -x "$TECTONIC_BIN" ]; then
    echo "Error: tectonic not found at $TECTONIC_BIN" >&2
    echo "Run '$0 install' first or set GOTEX_BIN to the tectonic directory." >&2
    exit 1
  fi
}

build_search_args() {
  SEARCH_ARGS=""
  if [ -d "$TEXMF_ROOT" ]; then
    for dir in $(find "$TEXMF_ROOT" -type d); do
      SEARCH_ARGS="$SEARCH_ARGS -Z search-path=$dir"
    done
  fi
}

compile() {
  ensure_tectonic

  if [ ! -d "$TEXMF_ROOT" ]; then
    echo "Error: texmf tree not found at $TEXMF_ROOT" >&2
    exit 1
  fi

  build_search_args

  if [ "$1" = "-" ] || [ "$1" = "" ]; then
    exec env TECTONIC_CACHE_DIR="$CACHE_DIR" "$TECTONIC_BIN" $SEARCH_ARGS -p "$@"
  fi

  if [ $# -ge 1 ] && [ "${1##*.}" = "tex" ]; then
    TEX_FILE=$1
    shift
    OUT_DIR=$(dirname "$TEX_FILE")

    HAVE_O=0
    for arg in "$@"; do
      if [ "$arg" = "-o" ]; then
        HAVE_O=1
        break
      fi
    done

    if [ "$HAVE_O" -eq 0 ]; then
      set -- -o "$OUT_DIR" "$@"
    fi

    exec env TECTONIC_CACHE_DIR="$CACHE_DIR" "$TECTONIC_BIN" $SEARCH_ARGS -p "$TEX_FILE" "$@"
  fi

  exec env TECTONIC_CACHE_DIR="$CACHE_DIR" "$TECTONIC_BIN" $SEARCH_ARGS -p "$@"
}

clean() {
  find "$GOTEX_TEXMF" -type f \( -name "*.log" -o -name "*.aux" -o -name "*.toc" -o -name "*.out" \) -delete
  rm -rf "$CACHE_DIR"
}

if [ $# -eq 0 ]; then
  usage
  exit 1
fi

case "$1" in
  install)
    install
    ;;
  clean)
    clean
    ;;
  help|-h|--help)
    usage
    ;;
  *)
    compile "$@"
    ;;
 esac
