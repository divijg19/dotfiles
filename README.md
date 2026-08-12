# dotfiles

## Installation

### Core Configuration Bootstrap
```bash
curl -fsSL https://raw.githubusercontent.com/divijg19/dotfiles/main/bootstrap.sh | bash
```
> *<sub>Installs and symlinks core terminal & editor configs (`fish`, `nvim`, `ghostty`, `zed`, `starship`, `atuin`, `tmux`, `yazi`) into `~/.config`. View [bootstrap.sh](bootstrap.sh) to inspect.</sub>*

### Interactive Binary Installer
> Installs the Go binaries (`Helm`, `Keen`, `Peony`, `Pulse`, `Sage`) into `~/.local/bin`, with pre-compiled release downloads and Go/Custom-URL fallbacks.

**Linux/macOS**
```bash
curl -fsSL https://raw.githubusercontent.com/divijg19/dotfiles/main/install.sh | bash
```
> <sup>View [install.sh](install.sh)</sup>

**Windows**
```powershell
irm https://raw.githubusercontent.com/divijg19/dotfiles/main/install.ps1 | iex
```
> <sup>View [install.ps1](install.ps1)</sup>
