#!/usr/bin/env bash

set -e

DEFAULT_INSTALL_DIR="${HOME}/.local/bin"
GITHUB_OWNER="divijg19"
GITHUB_REPO="dotfiles"
DOTFILES_RAW_URL="https://raw.githubusercontent.com/$GITHUB_OWNER/$GITHUB_REPO/main"

if [ -n "${BASH_SOURCE[0]:-}" ] && [ -f "${BASH_SOURCE[0]}" ]; then
  DOTFILES="$(cd "$(dirname "${BASH_SOURCE[0]}")" 2>/dev/null && pwd)"
else
  DOTFILES=""
fi
BIN_DIR="${DOTFILES:+$DOTFILES/bin}"

# Obtain raw .gitmodules content (local-first, remote-fallback)
fetch_gitmodules() {
  if [ -n "$DOTFILES" ] && [ -f "$DOTFILES/.gitmodules" ]; then
    cat "$DOTFILES/.gitmodules"
  else
    curl -fsSL "$DOTFILES_RAW_URL/.gitmodules"
  fi
}

# Parse .gitmodules to output "name url" lines
parse_gitmodules() {
  local content
  content="$(fetch_gitmodules)" || return 1

  awk '
    /^\[submodule/ {
      name = ""
    }
    /^\[submodule "bin\// {
      match($0, /"bin\/([^"]+)"/, m)
      name = m[1]
    }
    /url[ \t]*=/ {
      url = $0
      sub(/^.*url[ \t]*=[ \t]*/, "", url)
      sub(/[ \t\r]*$/, "", url)
      sub(/^git@github-[^:]+:/, "https://github.com/", url)
      sub(/^git@github\.com:/, "https://github.com/", url)
      if (name != "") {
        print name " " url
        name = ""
      }
    }
  ' <<< "$content"
}

# Resolve and install pre-compiled release binary using GitHub release redirects and candidate probing
resolve_and_install_release() {
  local proj="$1"
  local proj_lower="${proj,,}"
  local install_path="$INSTALL_DIR/$proj"
  release_fail_reason=""

  # 1. Resolve latest tag via GitHub redirect (HEAD request)
  local headers_file
  headers_file="$(mktemp)"
  local http_code
  if ! http_code=$(curl -sI -D "$headers_file" -o /dev/null -w "%{http_code}" "https://github.com/$GITHUB_OWNER/$proj/releases/latest" 2>/dev/null); then
    rm -f "$headers_file"
    release_fail_reason="Release lookup failed for $proj due to a network or transport error."
    return 1
  fi

  local location_line
  location_line=$(awk -F': ' 'BEGIN { IGNORECASE=1 } /^Location:/ {print $2}' "$headers_file" | tr -d '\r')
  rm -f "$headers_file"

  case "$http_code" in
    403)
      release_fail_reason="GitHub rate limit exceeded while checking $proj."
      return 1
      ;;
    404)
      release_fail_reason="No published GitHub release for $proj."
      return 1
      ;;
    301|302|307|308|200) ;;
    *)
      release_fail_reason="Could not resolve latest GitHub release for $proj."
      return 1
      ;;
  esac

  local tag
  tag=$(echo "$location_line" | sed -n 's|.*/tag/\([^/?#]*\).*|\1|p')

  if [ -z "$tag" ]; then
    release_fail_reason="Could not resolve latest GitHub release for $proj."
    return 1
  fi

  # 2. Construct release asset candidates based on OS and ARCH
  local os_variants=("$OS")
  if [ "$OS" = "darwin" ]; then
    os_variants=("darwin" "mac")
  fi

  local arch_variants=("$ARCH")
  if [ "$ARCH" = "amd64" ]; then
    arch_variants=("amd64" "x86_64")
  elif [ "$ARCH" = "arm64" ]; then
    arch_variants=("arm64" "aarch64")
  fi

  local candidates=()
  for os_v in "${os_variants[@]}"; do
    for arch_v in "${arch_variants[@]}"; do
      candidates+=("${proj_lower}-${tag}-${os_v}-${arch_v}")
      candidates+=("${proj_lower}_${tag}_${os_v}_${arch_v}.tar.gz")
      candidates+=("${proj_lower}-${tag}-${os_v}-${arch_v}.tar.gz")
    done
  done

  local matched_url=""
  for candidate in "${candidates[@]}"; do
    local candidate_url="https://github.com/$GITHUB_OWNER/$proj/releases/latest/download/$candidate"
    local check_code
    if check_code=$(curl -sSIL -o /dev/null -w "%{http_code}" "$candidate_url" 2>/dev/null); then
      if [ "$check_code" = "200" ]; then
        matched_url="$candidate_url"
        break
      fi
    fi
  done

  if [ -z "$matched_url" ]; then
    release_fail_reason="No compatible pre-compiled artifact found for $proj ($OS/$ARCH)."
    return 1
  fi

  echo "    Downloading release asset for $proj..."
  local fname
  fname=$(basename "$matched_url")

  if [[ "$fname" == *.tar.gz ]]; then
    local temp_archive
    temp_archive=$(mktemp /tmp/installer-XXXXXX.tar.gz)
    if curl -fsSL "$matched_url" -o "$temp_archive"; then
      local temp_extract_dir
      temp_extract_dir=$(mktemp -d /tmp/extract-XXXXXX)
      if tar -xzf "$temp_archive" -C "$temp_extract_dir"; then
        local found_bin
        found_bin=$(find "$temp_extract_dir" -type f -name "$proj_lower" -print -quit)
        if [ -n "$found_bin" ]; then
          mv "$found_bin" "$install_path"
          chmod +x "$install_path"
          echo "    Successfully installed pre-compiled $proj to $install_path"
          rm -rf "$temp_extract_dir"
          rm -f "$temp_archive"
          return 0
        fi
        release_fail_reason="Release archive for $proj did not contain an executable named '$proj_lower'."
      fi
      rm -rf "$temp_extract_dir"
    else
      release_fail_reason="Failed to download pre-compiled artifact for $proj."
    fi
    rm -f "$temp_archive"
  else
    if curl -fsSL "$matched_url" -o "$install_path"; then
      chmod +x "$install_path"
      echo "    Successfully installed pre-compiled $proj to $install_path"
      return 0
    fi
    release_fail_reason="Failed to download pre-compiled artifact for $proj."
  fi

  return 1
}

echo "=========================================="
echo "          Dotfiles Binary Installer       "
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

# --- Discover tools: top-level repository names from .gitmodules ---
echo "Discovering tools from .gitmodules..."
declare -a projects=()
declare -A project_urls=()

while read -r p_name p_url; do
  if [ -n "$p_name" ]; then
    projects+=("$p_name")
    project_urls["$p_name"]="$p_url"
  fi
done < <(parse_gitmodules)

if [ ${#projects[@]} -eq 0 ]; then
  echo "Error: no tools found in .gitmodules." >&2
  exit 1
fi

declare -A selected

# --- Interactive selection --------------------------------------------------
# Use fzf when a real terminal is available; it opens /dev/tty itself, so a
# plain pipe supplies the tool list and $(...) captures the selection.
if [ -r /dev/tty ] && command -v fzf &> /dev/null; then
  echo "Select projects to install (Use TAB or SPACE to select, ENTER to confirm):"
  echo "------------------------------------------"
  fzf_output=$(printf '%s\n' "${projects[@]}" | fzf --multi --bind 'space:toggle' --height=10 --reverse --prompt="Select binaries > ") || true
  if [ -n "$fzf_output" ]; then
    while IFS= read -r item; do
      [ -n "$item" ] && selected[$item]=1
    done <<< "$fzf_output"
  fi
else
  echo "Interactive Checklist (Toggle with [y/n]):"
  echo "------------------------------------------"
  for proj in "${projects[@]}"; do
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

installed_count=0
skipped_count=0
failed_count=0
installed_projects=()
skipped_projects=()
failed_projects=()

for proj in "${!selected[@]}"; do
  outcome=""

  echo "--> Processing $proj..."

  echo "    Resolving latest release for $proj..."
  if resolve_and_install_release "$proj"; then
    outcome="installed"
  else
    echo "    ${release_fail_reason:-Release lookup failed for $proj.}"
  fi

  # 2. Per-project fallback menu if the pre-compiled download failed
  if [ -z "$outcome" ]; then
    echo "    ⚠️  Could not fetch a pre-compiled binary for $proj."
    echo "    Choose a fallback method:"
    echo "      1) go build (build from local submodule or a shallow on-the-fly clone)"
    echo "      2) go install (remote Go module proxy installation)"
    echo "      3) Custom URL (provide a direct download link)"
    echo "      4) Skip this project"
    read -r -p "    Enter choice [1-4] (default 1): " fallback_choice < /dev/tty || true
    fallback_choice="${fallback_choice:-1}"

    # An unrecognized choice is an invalid selection, not an implicit skip.
    # Only an explicit "4" means the user chose to skip this project.

    case "$fallback_choice" in
      1)
        proj_path="$BIN_DIR/$proj"
        temp_dir=""

        if [ -d "$proj_path" ] && [ -f "$proj_path/go.mod" ]; then
          # Local submodule exists and is populated
          src_dir="$proj_path"
        else
          # Run via curl | bash or uninitialized submodule: shallow clone on-the-fly
          temp_dir="$(mktemp -d)"
          repo_url="${project_urls[$proj]:-https://github.com/$GITHUB_OWNER/$proj.git}"
          echo "    Cloning repository for $proj from $repo_url..."
          if ! git clone --depth 1 "$repo_url" "$temp_dir"; then
            echo "    ❌ Error: Failed to clone repository for $proj."
            rm -rf "$temp_dir"
            temp_dir=""
          fi
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
          if go build -o "$proj_install_dir/$proj" "$build_target"; then
            echo "    Successfully built and installed $proj to $proj_install_dir/$proj"
            outcome="installed"
          else
            echo "    ❌ Error: Failed to build $proj from $build_target."
          fi
          popd > /dev/null
        else
          echo "    ❌ Error: No go.mod found for $proj, cannot build locally."
        fi

        # Cleanup temp dir if created
        if [ -n "$temp_dir" ]; then
          rm -rf "$temp_dir"
        fi
        ;;
      2)
        read -r -p "    Enter Go module path (e.g., github.com/$GITHUB_OWNER/$proj@latest): " mod_path < /dev/tty || true
        if [ -n "$mod_path" ]; then
          read -r -p "    Override install path for $proj? [Default: $INSTALL_DIR]: " custom_proj_dir < /dev/tty || true
          proj_install_dir="${custom_proj_dir:-$INSTALL_DIR}"
          mkdir -p "$proj_install_dir"

          export GOBIN="$proj_install_dir"
          echo "    Running go install for $mod_path..."
          if go install "$mod_path"; then
            echo "    Successfully installed $proj via go install to $proj_install_dir/"
            outcome="installed"
          else
            echo "    ❌ Error: Failed to install $proj via go install."
          fi
        else
          echo "    ❌ Error: Module path cannot be empty."
        fi
        ;;
      3)
        read -r -p "    Enter direct binary download URL for $proj: " custom_url < /dev/tty || true
        if [ -n "$custom_url" ]; then
          read -r -p "    Override install path for $proj? [Default: $INSTALL_DIR]: " custom_proj_dir < /dev/tty || true
          proj_install_dir="${custom_proj_dir:-$INSTALL_DIR}"
          mkdir -p "$proj_install_dir"

          echo "    Downloading binary from $custom_url..."
          if curl -fsSL "$custom_url" -o "$proj_install_dir/$proj"; then
            chmod +x "$proj_install_dir/$proj"
            echo "    Successfully downloaded and installed $proj to $proj_install_dir/$proj"
            outcome="installed"
          else
            echo "    ❌ Error: Failed to download from $custom_url."
          fi
        else
          echo "    ❌ Error: URL cannot be empty."
        fi
        ;;
      4)
        echo "    Skipping $proj."
        outcome="skipped"
        ;;
      *)
        echo "    ❌ Error: Invalid fallback choice '$fallback_choice' for $proj."
        echo "             Expected 1, 2, 3, or 4."
        ;;
    esac
  fi

  case "$outcome" in
    installed) installed_count=$((installed_count + 1)); installed_projects+=("$proj") ;;
    skipped)   skipped_count=$((skipped_count + 1));     skipped_projects+=("$proj") ;;
    *)         failed_count=$((failed_count + 1));       failed_projects+=("$proj") ;;
  esac
  echo
done

echo "=========================================="
echo "Installation Summary"
echo "=========================================="
echo
echo "Installed: $installed_count"
if [ "${#installed_projects[@]}" -gt 0 ]; then
  printf '  %s\n' "${installed_projects[@]}"
fi
echo "Skipped:   $skipped_count"
if [ "${#skipped_projects[@]}" -gt 0 ]; then
  printf '  %s\n' "${skipped_projects[@]}"
fi
echo "Failed:    $failed_count"
if [ "${#failed_projects[@]}" -gt 0 ]; then
  printf '  %s\n' "${failed_projects[@]}"
fi
echo
echo "Installation directory:"
echo "  $INSTALL_DIR"
echo
if [ "$failed_count" -eq 0 ]; then
  echo "Installation completed successfully."
else
  echo "Installation completed with issues."
fi
echo "Make sure $INSTALL_DIR is in your PATH."

exit $((failed_count > 0 ? 1 : 0))
