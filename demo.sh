#!/usr/bin/env bash

set -e

DOTFILES="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="$DOTFILES/bin"
INSTALL_DIR="${HOME}/.local/bin"

mkdir -p "$INSTALL_DIR"

# Detect OS and Architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
esac

echo "=========================================="
echo "      Interactive Binary Demonstration    "
echo "=========================================="
echo "Detected OS: $OS, Architecture: $ARCH"
echo "Target install directory: $INSTALL_DIR"
echo

projects=("Helm" "Keen" "Peony" "Pulse" "Sage")

echo "Select projects to install:"
echo "------------------------------------------"

declare -A selected

for proj in "${projects[@]}"; do
  if [ -d "$BIN_DIR/$proj" ]; then
    read -p "Install $proj? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
      selected[$proj]=1
    fi
  fi
done

echo
echo "------------------------------------------"
echo "Installing selected projects..."
echo "------------------------------------------"

for proj in "${!selected[@]}"; do
  proj_lower="$(echo "$proj" | tr '[:upper:]' '[:lower:]')"
  proj_path="$BIN_DIR/$proj"
  installed=0

  echo "--> Processing $proj..."

  # 1. Try downloading pre-compiled release binary from GitHub
  release_url="https://github.com/divijg19/$proj/releases/latest/download/$proj-$OS-$ARCH"
  if curl --output /dev/null --silent --head --fail "$release_url"; then
    echo "    Downloading pre-compiled release for $proj..."
    if curl -fsSL "$release_url" -o "$INSTALL_DIR/$proj"; then
      chmod +x "$INSTALL_DIR/$proj"
      echo "    Successfully installed pre-compiled $proj to $INSTALL_DIR/$proj"
      installed=1
    fi
  fi

  # 2. Fallback to local build (go build) if pre-compiled download failed
  if [ $installed -eq 0 ]; then
    echo "    ⚠️ Could not fetch pre-compiled binary for $proj."
    read -p "    Would you like to build $proj locally from source using go build? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
      if [ -f "$proj_path/go.mod" ]; then
        pushd "$proj_path" > /dev/null
        
        build_target="."
        if [ -d "cmd" ]; then
          subcmd=$(find cmd -mindepth 1 -maxdepth 1 -type d | head -n 1)
          if [ -n "$subcmd" ]; then
            build_target="./$subcmd"
          fi
        fi
        
        echo "    Building $proj locally from $build_target..."
        go build -o "$INSTALL_DIR/$proj" "$build_target"
        echo "    Successfully built and installed $proj to $INSTALL_DIR/$proj"
        popd > /dev/null
      else
        echo "    ❌ Error: No go.mod found in $proj_path, cannot build locally."
      fi
    else
      echo "    Skipping $proj."
    fi
  fi
  echo
done

echo "=========================================="
echo "Installation process complete!"
echo "Make sure $INSTALL_DIR is in your PATH."
