#!/usr/bin/env bash

set -e

DOTFILES="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

backup() {
  if [ -e "$1" ] && [ ! -L "$1" ]; then
    echo "Backing up $1 → $1.bak"
    mv "$1" "$1.bak"
  fi
}

link() {
  src="$1"
  dest="$2"

  backup "$dest"

  echo "Linking $dest → $src"
  ln -sfn "$src" "$dest"
}

mkdir -p ~/.config
mkdir -p ~/.config/Code/User

# Core configs
link "$DOTFILES/fish" ~/.config/fish
link "$DOTFILES/nvim" ~/.config/nvim
link "$DOTFILES/ghostty" ~/.config/ghostty

# Starship
link "$DOTFILES/starship/starship.toml" ~/.config/starship.toml

# VS Code
link "$DOTFILES/vscode/settings.json" ~/.config/Code/User/settings.json
link "$DOTFILES/vscode/keybindings.json" ~/.config/Code/User/keybindings.json

echo "Dotfiles installed from $DOTFILES"
