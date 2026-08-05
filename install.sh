#!/usr/bin/env bash

set -e

# Support standalone execution via curl | bash (temp working directory)
DOTFILES="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="$DOTFILES/bin"
DEFAULT_INSTALL_DIR="${HOME}/.local/bin"

echo "=========================================="
echo "      Interactive Binary Demonstration    "
echo "=========================================="

# Detect OS and Architecture robustly
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  i386|i686) ARCH="386" ;;
  armv7l) ARCH="armv7" ;;
esac

echo "Detected OS: $OS, Architecture: $ARCH"
echo

# Authoritative project registry (works even if submodules aren't cloned or running via curl | bash)
projects=("Helm" "Keen" "Peony" "Pulse" "Sage")
available_projects=("${projects[@]}")

declare -A selected

# Use fzf only if running in an interactive terminal (tty available AND /dev/tty accessible)
if [ -t 0 ] && [ -r /dev/tty ] && command -v fzf &> /dev/null; then
  echo "Select projects to install (Use TAB or SPACE to select, ENTER to confirm):"
  echo "------------------------------------------"
  fzf_output=$(printf '%s\n' "${available_projects[@]}" | fzf --multi --bind 'space:toggle' --height=10 --reverse --prompt="Select binaries > " < /dev/tty > /dev/tty) || true
  if [ -n "$fzf_output" ]; then
    while IFS= read -r item; do
      [ -n "$item" ] && selected[$item]=1
    done <<< "$fzf_output"
  fi
else
  echo "Interactive Checklist (Toggle with [y/n]):"
  echo "------------------------------------------"
  for proj in "${available_projects[@]}"; do
    read -r -p "Install $proj? (y/N): " choice < /dev/tty || true
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
read -r -p "Enter installation directory [Default: $DEFAULT_INSTALL_DIR]: " custom_install_dir < /dev/tty || true
INSTALL_DIR="${custom_install_dir:-$DEFAULT_INSTALL_DIR}"
mkdir -p "$INSTALL_DIR"

echo
echo "------------------------------------------"
echo "Installing selected projects..."
echo "------------------------------------------"

for proj in "${!selected[@]}"; do
  installed=0

  echo "--> Processing $proj..."

  # 1. Try downloading pre-compiled release binary from GitHub standalone releases
  release_url="https://github.com/divijg19/$proj/releases/latest/download/$proj-$OS-$ARCH"
  if curl --output /dev/null --silent --head --fail "$release_url"; then
    echo "    Downloading pre-compiled release for $proj..."
    if curl -fsSL "$release_url" -o "$INSTALL_DIR/$proj"; then
      chmod +x "$INSTALL_DIR/$proj"
      echo "    Successfully installed pre-compiled $proj to $INSTALL_DIR/$proj"
      installed=1
    fi
  fi

  # 2. Fallback Menu if pre-compiled download failed
  if [ $installed -eq 0 ]; then
    echo "    ⚠️ Could not fetch pre-compiled binary for $proj."
    echo "    Choose fallback method for $proj:"
    echo "      1) go build (Clone source on-the-fly and build)"
    echo "      2) go install (Remote Go module proxy installation)"
    echo "      3) Custom URL (Provide a direct download link)"
    echo "      4) Skip this project"
    read -r -p "    Enter choice [1-4] (default 1): " fallback_choice < /dev/tty || true
    fallback_choice="${fallback_choice:-1}"

    case "$fallback_choice" in
      1)
        proj_path="$BIN_DIR/$proj"
        temp_dir=""
        
        if [ -d "$proj_path" ] && [ -f "$proj_path/go.mod" ]; then
          src_dir="$proj_path"
        else
          temp_dir="$(mktemp -d)"
          echo "    Cloning repository for $proj on-the-fly..."
          git clone --depth 1 "https://github.com/divijg19/$proj.git" "$temp_dir"
          src_dir="$temp_dir"
        fi

        if [ -f "$src_dir/go.mod" ]; then
          pushd "$src_dir" > /dev/null
          
          build_target="."
          if [ -d "cmd" ]; then
            subcmd=$(find cmd -mindepth 1 -maxdepth 1 -type d | head -n 1)
            if [ -n "$subcmd" ]; then
              build_target="./$subcmd"
            fi
          fi
          
          read -r -p "    Enter build target [Default: $build_target]: " custom_target < /dev/tty || true
          build_target="${custom_target:-$build_target}"

          read -r -p "    Override install path for $proj? [Default: $INSTALL_DIR]: " custom_proj_dir < /dev/tty || true
          proj_install_dir="${custom_proj_dir:-$INSTALL_DIR}"
          mkdir -p "$proj_install_dir"

          echo "    Building $proj locally from $build_target..."
          go build -o "$proj_install_dir/$proj" "$build_target"
          echo "    Successfully built and installed $proj to $proj_install_dir/$proj"
          popd > /dev/null
        else
          echo "    ❌ Error: No go.mod found for $proj, cannot build locally."
        fi

        [ -n "$temp_dir" ] && rm -rf "$temp_dir"
        ;;
      2)
        read -r -p "    Enter Go module path (e.g., github.com/divijg19/$proj@latest): " mod_path < /dev/tty || true
        if [ -n "$mod_path" ]; then
          read -r -p "    Override install path for $proj? [Default: $INSTALL_DIR]: " custom_proj_dir < /dev/tty || true
          proj_install_dir="${custom_proj_dir:-$INSTALL_DIR}"
          mkdir -p "$proj_install_dir"

          export GOBIN="$proj_install_dir"
          echo "    Running go install for $mod_path..."
          go install "$mod_path"
          echo "    Successfully installed $proj via go install to $proj_install_dir/"
        else
          echo "    ❌ Error: Module path cannot be empty."
        fi
        ;;
      3)
        read -r -p "    Enter direct binary download URL: " custom_url < /dev/tty || true
        if [ -n "$custom_url" ]; then
          read -r -p "    Override install path for $proj? [Default: $INSTALL_DIR]: " custom_proj_dir < /dev/tty || true
          proj_install_dir="${custom_proj_dir:-$INSTALL_DIR}"
          mkdir -p "$proj_install_dir"

          echo "    Downloading binary from $custom_url..."
          if curl -fsSL "$custom_url" -o "$proj_install_dir/$proj"; then
            chmod +x "$proj_install_dir/$proj"
            echo "    Successfully downloaded and installed $proj to $proj_install_dir/$proj"
          else
            echo "    ❌ Error: Failed to download from $custom_url."
          fi
        else
          echo "    ❌ Error: URL cannot be empty."
        fi
        ;;
      *)
        echo "    Skipping $proj."
        ;;
    esac
  fi
  echo
done

echo "=========================================="
echo "Installation process complete!"
echo "Make sure $INSTALL_DIR is in your PATH."
