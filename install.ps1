<#
.SYNOPSIS
    Dotfiles binary installer foundation (Windows / PowerShell 5.1+)
.DESCRIPTION
    Discovers projects from .gitmodules (locally or remotely), prompts for interactive
    project selection and installation directory, and prepares the foundation.
#>

$ErrorActionPreference = "Stop"

# 1. Installer Constants
$GITHUB_OWNER = "divijg19"
$GITHUB_REPO = "dotfiles"
$DOTFILES_RAW_URL = "https://raw.githubusercontent.com/$GITHUB_OWNER/$GITHUB_REPO/main"
$DEFAULT_INSTALL_DIR = Join-Path $HOME ".local\bin"

# 2. Architecture Detection
$ARCH = switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { "amd64" }
    "ARM64" { "arm64" }
    "x86"   { "386" }
    "ARM"   { "arm" }
    default {
        throw "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE"
    }
}

# 3. .gitmodules Acquisition
function Get-GitmodulesContent {
    # Check if local .gitmodules exists relative to current execution or PSScriptRoot
    $localPath = if ($PSScriptRoot) { Join-Path $PSScriptRoot ".gitmodules" } else { ".gitmodules" }
    
    if (Test-Path $localPath) {
        Get-Content $localPath -Raw
    } else {
        try {
            $uri = "$DOTFILES_RAW_URL/.gitmodules"
            $response = Invoke-WebRequest -Uri $uri -UseBasicParsing
            if (-not $response.Content) {
                throw "Received empty response from $uri"
            }
            $response.Content
        }
        catch {
            throw "Failed to retrieve .gitmodules: $($_.Exception.Message)"
        }
    }
}

# 4. Parse .gitmodules
function Get-ProjectsFromGitmodules {
    param([string]$content)
    
    $projects = @()
    $currentName = $null
    
    foreach ($line in ($content -split "`r?`n")) {
        if ($line -match '^\[submodule\s+"bin/([^"]+)"\]') {
            $currentName = $matches[1]
        }
        elseif ($line -match '^\s*url\s*=\s*(.+)$') {
            $url = $matches[1].Trim()
            # Normalize SSH GitHub URL to HTTPS
            $url = $url -replace '^git@github-[^:]+:', 'https://github.com/'
            $url = $url -replace '^git@github\.com:', 'https://github.com/'
            
            if ($currentName) {
                $projects += [pscustomobject]@{
                    Name = $currentName
                    Url  = $url
                }
                $currentName = $null
            }
        }
    }
    
    return $projects
}

# Main Execution Flow
try {
    Write-Host "=========================================="
    Write-Host "      Dotfiles Binary Installer           "
    Write-Host "=========================================="
    Write-Host "Detected OS: windows"
    Write-Host "Architecture: $ARCH"
    Write-Host ""
    Write-Host "Discovering tools from .gitmodules..."

    $content = Get-GitmodulesContent
    $projects = Get-ProjectsFromGitmodules -content $content

    if ($null -eq $projects -or $projects.Count -eq 0) {
        throw "Error: no tools found in .gitmodules."
    }

    # Interactive Project Selection
    $selected = @()
    foreach ($p in $projects) {
        $choice = Read-Host "Install $($p.Name)? (y/N)"
        if ($choice -match '^[Yy]$') {
            $selected += $p.Name
        }
    }

    if ($selected.Count -eq 0) {
        Write-Host ""
        Write-Host "No projects selected. Exiting."
        exit 0
    }

    # Installation Directory Selection
    Write-Host ""
    $dirInput = Read-Host "Enter installation directory [Default: $DEFAULT_INSTALL_DIR]"
    $targetDir = if ([string]::IsNullOrWhiteSpace($dirInput)) { $DEFAULT_INSTALL_DIR } else { $dirInput.Trim() }

    try {
        New-Item -ItemType Directory -Force -Path $targetDir | Out-Null
    }
    catch {
        throw "Failed to create installation directory '$targetDir': $($_.Exception.Message)"
    }

    # Foundation Confirmation
    Write-Host ""
    Write-Host "Installer foundation initialized."
    Write-Host ""
    Write-Host "Selected projects:"
    foreach ($s in $selected) {
        Write-Host "  - $s"
    }
    Write-Host ""
    Write-Host "Installation directory:"
    Write-Host "  $targetDir"
    Write-Host ""

    exit 0
}
catch {
    Write-Error $_.Exception.Message
    exit 1
}
