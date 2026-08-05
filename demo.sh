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
available_projects=()

for proj in "${projects[@]}"; do
  if [ -d "$BIN_DIR/$proj" ]; then
    available_projects+=("$proj")
  fi
done

if [ ${#available_projects[@]} -eq 0 ]; then
  echo "No projects found in $BIN_DIR."
  exit 0
fi

declare -A selected

if command -v fzf &> /dev/null; then
  echo "Select projects to install (Use TAB or SPACE to select, ENTER to confirm):"
  echo "------------------------------------------"
  fzf_output=$(printf '%s\n' "${available_projects[@]}" | fzf --multi --bind 'space:toggle' --height=10 --reverse --prompt="Select binaries > ")
  if [ -n "$fzf_output" ]; then
    while IFS= read -r item; do
      [ -n "$item" ] && selected[$item]=1
    done <<< "$fzf_output"
  fi
else
  echo "Interactive Checklist (Toggle with [y/n]):"
  echo "------------------------------------------"
  for proj in "${available_projects[@]}"; do
    read -p "Install $proj? (y/N): " choice
    if [[ "$choice" =~ ^[Yy]$ ]]; then
      selected[$proj]=1
    fi
  done
fi

if [ ${#selected[@]} -eq 0 ]; then
  echo "No projects selected. Exiting."
  exit 0
fi

echo
echo "------------------------------------------"
echo "Installing selected projects..."
echo "------------------------------------------"

for proj in "${!selected[@]}"; do
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
    echo "    Building $proj locally from source using go build..."
    if [ -f "$proj_path/go.mod" ]; then
      pushd "$proj_path" > /dev/null
      
      build_target="."
      if [ -d "cmd" ]; then
        subcmd=$(find cmd -mindepth 1 -maxdepth 1 -type d | head -n 1)
        if [ -n "$subcmd" ]; then
          build_target="./$subcmd"
        fi
      fi
      
      go build -o "$INSTALL_DIR/$proj" "$build_target"
      echo "    Successfully built and installed $proj to $INSTALL_DIR/$proj"
      popd > /dev/null
    else
      echo "    ❌ Error: No go.mod found in $proj_path, cannot build locally."
    fi
  fi
  echo
done

echo "=========================================="
echo "Installation process complete!"
echo "Make sure $INSTALL_DIR is in your PATH."
