#!/usr/bin/env sh
# storage CLI installer
# Usage: curl -fsSL https://storage.liteio.dev/cli/install.sh | sh
set -e

REPO_URL="https://storage.liteio.dev/cli/storage"

# Find install directory
find_install_dir() {
  # Prefer ~/.local/bin (XDG standard)
  if [ -d "$HOME/.local/bin" ]; then
    echo "$HOME/.local/bin"
    return
  fi

  # Check PATH for writable directories
  IFS=:
  for dir in $PATH; do
    if [ -w "$dir" ] && [ "$dir" != "." ]; then
      echo "$dir"
      return
    fi
  done
  unset IFS

  # Fallback: create ~/.local/bin
  mkdir -p "$HOME/.local/bin"
  echo "$HOME/.local/bin"
}

main() {
  echo "Installing storage CLI..."
  echo ""

  INSTALL_DIR=$(find_install_dir)
  INSTALL_PATH="$INSTALL_DIR/storage"

  # Download
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$REPO_URL" -o "$INSTALL_PATH"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$INSTALL_PATH" "$REPO_URL"
  else
    echo "error: curl or wget required" >&2
    exit 1
  fi

  chmod +x "$INSTALL_PATH"

  # Verify
  if [ -x "$INSTALL_PATH" ]; then
    echo "  Installed to $INSTALL_PATH"
    echo ""

    # Check if directory is in PATH
    case ":$PATH:" in
      *":$INSTALL_DIR:"*) ;;
      *)
        echo "  Add to your PATH:"
        echo "    export PATH=\"$INSTALL_DIR:\$PATH\""
        echo ""
        ;;
    esac

    echo "  Get started:"
    echo "    storage login"
    echo "    storage --help"
    echo ""
  else
    echo "error: installation failed" >&2
    exit 1
  fi
}

main
