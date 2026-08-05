# dotfiles

## Installation

### Core Configuration Bootstrap
```bash
curl -fsSL https://raw.githubusercontent.com/divijg19/dotfiles/main/bootstrap.sh | bash
```
> *<sub>Installs and symlinks core terminal & editor configs (`fish`, `nvim`, `ghostty`, `zed`, `starship`, `atuin`, `tmux`, `yazi`) into `~/.config`. View [bootstrap.sh](bootstrap.sh) to inspect.</sub>*

### Interactive Binary Installer
```bash
curl -fsSL https://raw.githubusercontent.com/divijg19/dotfiles/main/install.sh | bash
```
> *<sub>Interactively selects, downloads pre-compiled releases, or locally builds/installs Go binaries (`Helm`, `Keen`, `Peony`, `Pulse`, `Sage`) into `~/.local/bin`. View [install.sh](install.sh) to inspect.</sub>*
