#!/usr/bin/env bash

set -e

DOTFILES="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

backup() {
  if [ -e "$1" ] && [ ! -L "$1" ]; then
    local ts
    ts="$(date +%Y%m%d-%H%M%S)"
    echo "Backing up $1 → $1.bak-$ts"
    mv "$1" "$1.bak-$ts"
  fi
}

link() {
  local src="$1"
  local dest="$2"

  # Ensure parent directory exists
  mkdir -p "$(dirname "$dest")"

  # Resolve source to absolute path
  src="$(cd "$(dirname "$src")" && pwd)/$(basename "$src")"

  # Destination does not exist → create symlink
  if [ ! -e "$dest" ] && [ ! -L "$dest" ]; then
    echo "Linking $dest → $src"
    ln -s "$src" "$dest"
    return
  fi

  # Destination is a symlink → check if it's the correct one
  if [ -L "$dest" ]; then
    local target
    target="$(readlink "$dest")"
    if [ "$target" = "$src" ]; then
      # Already the correct symlink — nothing to do
      return
    fi
    # Wrong or broken symlink — replace it
    echo "Linking $dest → $src"
    ln -sfn "$src" "$dest"
    return
  fi

  # Regular file or directory — back it up first
  backup "$dest"
  echo "Linking $dest → $src"
  ln -s "$src" "$dest"
}

mkdir -p ~/.config

# Core configs
link "$DOTFILES/fish" ~/.config/fish
link "$DOTFILES/nvim" ~/.config/nvim
link "$DOTFILES/ghostty" ~/.config/ghostty
link "$DOTFILES/zed" ~/.config/zed

# Starship
link "$DOTFILES/starship/starship.toml" ~/.config/starship.toml

# Atuin
link "$DOTFILES/atuin/config.toml" ~/.config/atuin/config.toml

# Tmux
link "$DOTFILES/tmux/tmux.conf" ~/.config/tmux/tmux.conf

# Yazi
link "$DOTFILES/yazi" ~/.config/yazi

echo "Dotfiles successfully installed from $DOTFILES!"