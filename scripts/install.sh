#!/usr/bin/env sh
# Draconiforge installer — POSIX sh
# One-liner: curl -fsSL https://raw.githubusercontent.com/draconis-engineering/draconiforge/main/scripts/install.sh | sh
# or:       wget -qO- https://raw.githubusercontent.com/draconis-engineering/draconiforge/main/scripts/install.sh | sh
set -eu

REPO="https://github.com/draconis-engineering/draconiforge.git"
BIN="draconiforge"
ALIAS="forge"
PREFIX="${PREFIX:-$HOME/.local}"
BINDIR="${BINDIR:-$PREFIX/bin}"
DATADIR="${DATADIR:-$PREFIX/share}"
VERSION="${VERSION:-main}"
NO_MODIFY_PATH="${NO_MODIFY_PATH:-}"

for arg in "$@"; do
  case "$arg" in
    --prefix=*) PREFIX="${arg#--prefix=}"; BINDIR="$PREFIX/bin" ;;
    --version=*) VERSION="${arg#--version=}" ;;
    --no-modify-path) NO_MODIFY_PATH=1 ;;
    -h|--help) echo "Usage: install.sh [--prefix=PATH] [--version=TAG] [--no-modify-path]"; exit 0 ;;
  esac
done

info(){ printf "\033[1;34m==>%s\033[0m\n" "$*"; }
ok(){ printf "\033[1;32m✓ %s\033[0m\n" "$*"; }
warn(){ printf "\033[1;33m⚠ %s\033[0m\n" "$*"; }
err(){ printf "\033[1;31m✖ %s\033[0m\n" "$*" >&2; }

if ! command -v git >/dev/null 2>&1; then err "git not found — install git first"; exit 1; fi
if ! command -v go >/dev/null 2>&1; then err "go not found — install Go 1.24+ from https://go.dev/dl/"; exit 1; fi

info "Installing $BIN ($ALIAS) from $REPO@$VERSION"
info "Prefix: $PREFIX  Bin: $BINDIR"

TMPDIR="$(mktemp -d 2>/dev/null || mktemp -d -t draconiforge)"
trap 'rm -rf "$TMPDIR"' EXIT INT TERM

if [ -d "$TMPDIR/src/.git" ]; then
  git -C "$TMPDIR/src" fetch --depth 1 origin "$VERSION" 2>/dev/null || git -C "$TMPDIR/src" fetch --depth 1 origin main
  git -C "$TMPDIR/src" checkout "$VERSION" 2>/dev/null || git -C "$TMPDIR/src" checkout FETCH_HEAD
else
  info "Cloning $REPO ..."
  git clone --depth 1 --branch "$VERSION" "$REPO" "$TMPDIR/src" 2>/dev/null || git clone --depth 1 "$REPO" "$TMPDIR/src"
fi

info "Building $BIN ..."
if command -v make >/dev/null 2>&1 && [ -f "$TMPDIR/src/Makefile" ]; then
  make -C "$TMPDIR/src" build PREFIX="$PREFIX" BINDIR="$BINDIR" 2>&1 | sed 's/^/  /'
  BIN_SRC="$TMPDIR/src/draconiforge"
else
  (cd "$TMPDIR/src" && go build -ldflags "-s -w" -o "$TMPDIR/$BIN" ./cmd/goforge-cli)
  BIN_SRC="$TMPDIR/$BIN"
fi

if [ ! -f "$BIN_SRC" ] && [ ! -f "$TMPDIR/src/$BIN" ]; then err "build failed"; exit 1; fi
[ -f "$BIN_SRC" ] || BIN_SRC="$TMPDIR/src/$BIN"

mkdir -p "$BINDIR" "$DATADIR/draconiforge/scripts"
info "Installing to $BINDIR/$BIN ..."
install -m 0755 "$BIN_SRC" "$BINDIR/$BIN"
ln -sf "$BIN" "$BINDIR/$ALIAS" 2>/dev/null || cp "$BINDIR/$BIN" "$BINDIR/$ALIAS"
ok "installed $BINDIR/$BIN + $BINDIR/$ALIAS"

if [ -d "$TMPDIR/src/scripts" ]; then
  for s in "$TMPDIR/src/scripts"/*.sh "$TMPDIR/src/scripts"/*.ps1; do
    [ -e "$s" ] || continue
    case "$s" in *install.sh|*install.ps1) continue;; esac
    install -m 0755 "$s" "$DATADIR/draconiforge/scripts/" 2>/dev/null || cp "$s" "$DATADIR/draconiforge/scripts/"
  done
  ok "scripts → $DATADIR/draconiforge/scripts/"
fi

if "$BINDIR/$BIN" version >/dev/null 2>&1; then ok "version: $($BINDIR/$BIN version 2>/dev/null | head -1)"; fi

if ! echo ":$PATH:" | grep -q ":$BINDIR:"; then
  warn "$BINDIR not in PATH"
  if [ -z "$NO_MODIFY_PATH" ]; then
    SHELL_NAME="$(basename "${SHELL:-sh}")"
    RC=""
    case "$SHELL_NAME" in
      zsh) [ -f "$HOME/.zshrc" ] && RC="$HOME/.zshrc" || RC="$HOME/.profile" ;;
      bash) [ -f "$HOME/.bashrc" ] && RC="$HOME/.bashrc" || RC="$HOME/.bash_profile" ;;
      fish) RC="$HOME/.config/fish/config.fish" ;;
      *) RC="$HOME/.profile" ;;
    esac
    if [ "$SHELL_NAME" = "fish" ]; then
      echo "set -gx PATH $HOME/.local/bin $PATH" >> "$RC"
    else
      echo "" >> "$RC"; echo "# added by draconiforge installer" >> "$RC"; echo "export PATH="$BINDIR:$PATH"" >> "$RC"
    fi
    ok "added PATH to $RC — restart shell or run: export PATH="$BINDIR:$PATH""
  else
    echo "  add manually: export PATH="$BINDIR:$PATH""
  fi
else
  ok "$BINDIR already in PATH"
fi

if command -v df >/dev/null 2>&1 && [ "$(command -v df)" = "/usr/bin/df" ]; then ok "use 'forge' (avoids conflict with system 'df')"; fi
echo ""; ok "Try: forge doctor --verbose"
