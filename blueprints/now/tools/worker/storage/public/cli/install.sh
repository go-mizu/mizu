#!/usr/bin/env sh
# Liteio Storage CLI installer
# Usage: curl -fsSL https://storage.liteio.dev/cli/install.sh | sh
#
# Detects OS and architecture, downloads the correct binary,
# and installs to /usr/local/bin (or ~/.local/bin if no sudo).
#
# Environment variables:
#   STORAGE_VERSION   Pin to a specific version (default: latest)
#   INSTALL_DIR       Override install directory

set -eu

BASE_URL="https://storage.liteio.dev/cli/releases"

# Detect OS
detect_os() {
  os="$(uname -s)"
  case "$os" in
    Linux)  echo "linux" ;;
    Darwin) echo "darwin" ;;
    *)      error "Unsupported OS: $os" ;;
  esac
}

# Detect architecture
detect_arch() {
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64)   echo "amd64" ;;
    aarch64|arm64)   echo "arm64" ;;
    *)               error "Unsupported architecture: $arch" ;;
  esac
}

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

# Find a writable install directory
find_install_dir() {
  # User override
  if [ -n "${INSTALL_DIR:-}" ]; then
    mkdir -p "$INSTALL_DIR"
    echo "$INSTALL_DIR"
    return
  fi

  # Try /usr/local/bin first (standard for user-installed binaries)
  if [ -w "/usr/local/bin" ]; then
    echo "/usr/local/bin"
    return
  fi

  # Try with sudo
  if command -v sudo >/dev/null 2>&1; then
    echo "/usr/local/bin"
    return
  fi

  # Fallback to ~/.local/bin
  mkdir -p "$HOME/.local/bin"
  echo "$HOME/.local/bin"
}

# Download a URL to a file
download() {
  url="$1"
  dest="$2"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$dest"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$dest" "$url"
  else
    error "curl or wget is required"
  fi
}

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

  # Build download URL
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
      *":$INSTALL_DIR:"*) ;;
      *)
        printf "\n"
        printf "  Add to your PATH:\n"
        printf "    export PATH=\"%s:\$PATH\"\n" "$INSTALL_DIR"
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
