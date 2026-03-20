#!/usr/bin/env sh
# Liteio Storage CLI installer
# Usage: curl -fsSL https://storage.liteio.dev/cli/install.sh | sh
#
# Detects OS and architecture, downloads the correct binary from R2
# (via a signed redirect), and installs it to a directory in PATH.
#
# Environment variables:
#   STORAGE_VERSION   Pin to a specific version (default: latest)
#   INSTALL_DIR       Override install directory

set -eu

BASE_URL="https://storage.liteio.dev/cli/releases"

# ── Detection ─────────────────────────────────────────────────────────

detect_os() {
  os="$(uname -s)"
  case "$os" in
    Linux)  echo "linux" ;;
    Darwin) echo "darwin" ;;
    MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
    *)      error "Unsupported OS: $os" ;;
  esac
}

detect_arch() {
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64)    echo "amd64" ;;
    aarch64|arm64)   echo "arm64" ;;
    *)               error "Unsupported architecture: $arch" ;;
  esac
}

# ── Output helpers ────────────────────────────────────────────────────

error() {
  printf "\033[31merror:\033[0m %s\n" "$1" >&2
  exit 1
}

info() {
  printf "  \033[32m%s\033[0m %s\n" "$1" "$2"
}

dim() {
  printf "  \033[2m%s\033[0m\n" "$1"
}

# ── Install directory ─────────────────────────────────────────────────

find_install_dir() {
  # User override
  if [ -n "${INSTALL_DIR:-}" ]; then
    mkdir -p "$INSTALL_DIR"
    echo "$INSTALL_DIR"
    return
  fi

  # Try /usr/local/bin first (standard location)
  if [ -w "/usr/local/bin" ]; then
    echo "/usr/local/bin"
    return
  fi

  # Try with sudo
  if command -v sudo >/dev/null 2>&1; then
    echo "/usr/local/bin"
    return
  fi

  # Fallback to ~/.local/bin (XDG standard)
  mkdir -p "$HOME/.local/bin"
  echo "$HOME/.local/bin"
}

# ── Download (follows redirects for signed R2 URLs) ───────────────────

download() {
  url="$1"
  dest="$2"

  if command -v curl >/dev/null 2>&1; then
    HTTP_CODE=$(curl -fsSL -w '%{http_code}' "$url" -o "$dest" 2>/dev/null) || {
      error "Download failed (HTTP $HTTP_CODE). Check https://storage.liteio.dev/cli for help."
    }
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$dest" "$url" 2>/dev/null || {
      error "Download failed. Check https://storage.liteio.dev/cli for help."
    }
  else
    error "curl or wget is required"
  fi

  # Verify we got a binary, not an error page
  if [ -f "$dest" ]; then
    file_size=$(wc -c < "$dest" | tr -d '[:space:]')
    if [ "$file_size" -lt 1000 ]; then
      # Likely an error response, not a binary
      content=$(cat "$dest" 2>/dev/null || echo "")
      case "$content" in
        *"not_found"*|*"error"*)
          error "Binary not available for your platform. Visit https://storage.liteio.dev/cli"
          ;;
      esac
    fi
  fi
}

# ── Main ──────────────────────────────────────────────────────────────

main() {
  OS="$(detect_os)"
  ARCH="$(detect_arch)"
  VERSION="${STORAGE_VERSION:-latest}"

  printf "\n"
  printf "  Liteio Storage CLI installer\n"
  printf "\n"
  dim "OS: $OS, Arch: $ARCH"
  printf "\n"

  INSTALL_DIR="$(find_install_dir)"
  INSTALL_PATH="$INSTALL_DIR/storage"
  NEEDS_SUDO=false

  if [ ! -w "$INSTALL_DIR" ]; then
    NEEDS_SUDO=true
  fi

  # Build download URL (server redirects to signed R2 link)
  FILENAME="storage-${OS}-${ARCH}"
  DOWNLOAD_URL="${BASE_URL}/${VERSION}/${FILENAME}"

  # Download to temp file
  TMPDIR="${TMPDIR:-/tmp}"
  TMPFILE="$(mktemp "$TMPDIR/storage-XXXXXX")"
  trap 'rm -f "$TMPFILE"' EXIT

  info "Downloading" "$DOWNLOAD_URL"
  download "$DOWNLOAD_URL" "$TMPFILE"
  chmod +x "$TMPFILE"

  # Install
  if [ "$NEEDS_SUDO" = true ]; then
    info "Installing" "$INSTALL_PATH (requires sudo)"
    sudo mv "$TMPFILE" "$INSTALL_PATH"
    sudo chmod +x "$INSTALL_PATH"
  else
    mv "$TMPFILE" "$INSTALL_PATH"
  fi

  # Verify
  if [ -x "$INSTALL_PATH" ]; then
    printf "\n"
    info "Installed" "storage to $INSTALL_PATH"

    # Print version
    INSTALLED_VERSION="$("$INSTALL_PATH" --version 2>/dev/null || echo "unknown")"
    dim "$INSTALLED_VERSION"

    # Check if directory is in PATH
    case ":$PATH:" in
      *":$INSTALL_DIR:"*)
        ;;
      *)
        printf "\n"
        printf "  \033[33mNote:\033[0m %s is not in your PATH.\n" "$INSTALL_DIR"
        printf "  Add it to your shell profile:\n"
        printf "\n"
        SHELL_NAME="$(basename "${SHELL:-/bin/sh}")"
        case "$SHELL_NAME" in
          zsh)
            printf "    echo 'export PATH=\"%s:\$PATH\"' >> ~/.zshrc\n" "$INSTALL_DIR"
            printf "    source ~/.zshrc\n"
            ;;
          bash)
            printf "    echo 'export PATH=\"%s:\$PATH\"' >> ~/.bashrc\n" "$INSTALL_DIR"
            printf "    source ~/.bashrc\n"
            ;;
          fish)
            printf "    fish_add_path %s\n" "$INSTALL_DIR"
            ;;
          *)
            printf "    export PATH=\"%s:\$PATH\"\n" "$INSTALL_DIR"
            ;;
        esac
        ;;
    esac

    printf "\n"
    printf "  Get started:\n"
    printf "    storage login\n"
    printf "    storage --help\n"
    printf "\n"
  else
    error "Installation failed"
  fi
}

main
