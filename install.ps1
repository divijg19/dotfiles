<#
.SYNOPSIS
    Dotfiles binary installer release resolver (Windows / PowerShell 5.1+)
.DESCRIPTION
    Discovers projects from .gitmodules (locally or remotely), prompts for interactive
    project selection, and resolves latest GitHub release assets without the GitHub API.
#>

$ErrorActionPreference = "Stop"

# 1. Installer Constants
$GITHUB_OWNER = "divijg19"
$GITHUB_REPO = "dotfiles"
$DOTFILES_RAW_URL = "https://raw.githubusercontent.com/$GITHUB_OWNER/$GITHUB_REPO/main"

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

# 5. API-free Release Resolver
function Resolve-LatestRelease {
    param(
        [string]$owner,
        [string]$proj
    )
    $url = "https://github.com/$owner/$proj/releases/latest"
    $req = [System.Net.HttpWebRequest]::Create($url)
    $req.AllowAutoRedirect = $false
    $res = $null
    try {
        $res = $req.GetResponse()
        $statusCode = [int]$res.StatusCode
        if ($statusCode -ge 300 -and $statusCode -lt 400) {
            $loc = $res.Headers["Location"]
            if ($loc -match '/tag/([^/?#]+)') {
                return $Matches[1]
            }
            throw "Could not resolve latest release tag from redirect location for $proj."
        }
        throw "Unexpected status code resolving latest release for $proj: $statusCode"
    }
    catch {
        if ($_.Exception.Response) {
            $code = [int]$_.Exception.Response.StatusCode
            if ($code -eq 404) {
                throw "No published GitHub release for $proj."
            }
        }
        # If it was already thrown by our code, propagate it
        if ($_.Exception.Message -match "No published GitHub release" -or $_.Exception.Message -match "Could not resolve latest release") {
            throw $_.Exception.Message
        }
        throw "Could not resolve latest release for $proj: $($_.Exception.Message)"
    }
    finally {
        if ($null -ne $res) {
            $res.Dispose()
        }
    }
}

function Get-ReleaseAssetCandidates {
    param(
        [string]$proj,
        [string]$tag,
        [string]$arch
    )
    $projLower = $proj.ToLower()
    $osVariants = @("windows", "win")

    $archVariants = @($arch)
    if ($arch -eq "amd64") {
        $archVariants = @("amd64", "x86_64")
    } elseif ($arch -eq "arm64") {
        $archVariants = @("arm64", "aarch64")
    }

    $candidates = @()
    foreach ($osV in $osVariants) {
        foreach ($archV in $archVariants) {
            $candidates += "$projLower-$tag-$osV-$archV"
            $candidates += "$projLower-$tag-$osV-$archV.exe"
            $candidates += "$projLower-$tag-$osV-$archV.zip"
            $candidates += "${projLower}_${tag}_${osV}_${archV}.zip"
        }
    }
    return $candidates
}

function Resolve-ReleaseAsset {
    param(
        [string]$owner,
        [string]$proj,
        [string]$tag,
        [string]$arch
    )
    $candidates = Get-ReleaseAssetCandidates -proj $proj -tag $tag -arch $arch
    foreach ($candidate in $candidates) {
        $candidateUrl = "https://github.com/$owner/$proj/releases/latest/download/$candidate"
        $req = [System.Net.HttpWebRequest]::Create($candidateUrl)
        $req.Method = "HEAD"
        $req.AllowAutoRedirect = $true
        $res = $null
        try {
            $res = $req.GetResponse()
            if ([int]$res.StatusCode -eq 200) {
                $kind = if ($candidate -match '\.zip$') { "Archive" } else { "Direct" }
                return [pscustomobject]@{
                    Tag       = $tag
                    AssetName = $candidate
                    AssetUrl  = $candidateUrl
                    Kind      = $kind
                }
            }
        }
        catch {
            # Probe next candidate
        }
        finally {
            if ($null -ne $res) {
                $res.Dispose()
            }
        }
    }
    throw "No compatible Windows asset found for $proj."
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

    # Release Resolution Demonstration
    Write-Host ""
    Write-Host "Resolving latest releases (API-free)..."
    foreach ($projectName in $selected) {
        Write-Host "  -> Checking $projectName..."
        try {
            $tag = Resolve-LatestRelease -owner $GITHUB_OWNER -proj $projectName
            Write-Host "     Latest tag: $tag"

            $resolved = Resolve-ReleaseAsset -owner $GITHUB_OWNER -proj $projectName -tag $tag -arch $ARCH
            Write-Host "     Found asset: $($resolved.AssetName) [Kind: $($resolved.Kind)]" -ForegroundColor Green
            Write-Host "     URL: $($resolved.AssetUrl)"
        }
        catch {
            Write-Host "     Error: $_" -ForegroundColor Yellow
        }
    }

    Write-Host ""
    Write-Host "Release resolver check complete. (No files installed)"
    Write-Host ""

    exit 0
}
catch {
    Write-Error $_.Exception.Message
    exit 1
}
