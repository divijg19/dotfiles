<#
.SYNOPSIS
    Dotfiles binary installer (Windows / PowerShell 5.1+)
.DESCRIPTION
    Discovers projects from .gitmodules (locally or remotely), prompts for interactive
    project selection and installation directory, resolves latest GitHub release assets,
    and installs direct binary or archive assets.
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

# 6. Binary Installation
function Install-DirectBinary {
    param(
        [string]$assetUrl,
        [string]$proj,
        [string]$targetDir
    )
    $tempFile = [System.IO.Path]::GetTempFileName()
    try {
        Invoke-WebRequest -Uri $assetUrl -OutFile $tempFile -UseBasicParsing
        if (-not (Test-Path $tempFile) -or (Get-Item $tempFile).Length -eq 0) {
            throw "Downloaded binary for $proj is empty or missing."
        }
        $destPath = Join-Path $targetDir "$proj.exe"
        Copy-Item -Path $tempFile -Destination $destPath -Force
        Write-Host "Successfully installed $proj to $destPath" -ForegroundColor Green
    }
    finally {
        if (Test-Path $tempFile) {
            Remove-Item -Path $tempFile -Force -ErrorAction SilentlyContinue
        }
    }
}

function Install-ArchiveAsset {
    param(
        [string]$assetUrl,
        [string]$proj,
        [string]$targetDir
    )
    $tempZip = [System.IO.Path]::GetTempFileName()
    $tempExtractDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
    try {
        Invoke-WebRequest -Uri $assetUrl -OutFile $tempZip -UseBasicParsing
        if (-not (Test-Path $tempZip) -or (Get-Item $tempZip).Length -eq 0) {
            throw "Downloaded archive for $proj is empty or missing."
        }
        New-Item -ItemType Directory -Force -Path $tempExtractDir | Out-Null
        Expand-Archive -Path $tempZip -DestinationPath $tempExtractDir -Force

        $exeName = "$proj.exe"
        $foundExe = Get-ChildItem -Path $tempExtractDir -Recurse -File | Where-Object { $_.Name -ieq $exeName } | Select-Object -First 1

        if (-not $foundExe) {
            throw "Archive for $proj did not contain an executable named '$exeName'."
        }

        $destPath = Join-Path $targetDir "$proj.exe"
        Copy-Item -Path $foundExe.FullName -Destination $destPath -Force
        Write-Host "Successfully installed $proj to $destPath" -ForegroundColor Green
    }
    finally {
        if (Test-Path $tempZip) {
            Remove-Item -Path $tempZip -Force -ErrorAction SilentlyContinue
        }
        if (Test-Path $tempExtractDir) {
            Remove-Item -Path $tempExtractDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

function Install-ReleaseAsset {
    param(
        [pscustomobject]$resolved,
        [string]$proj,
        [string]$targetDir
    )
    switch ($resolved.Kind) {
        "Direct"  { Install-DirectBinary -assetUrl $resolved.AssetUrl -proj $proj -targetDir $targetDir }
        "Archive" { Install-ArchiveAsset -assetUrl $resolved.AssetUrl -proj $proj -targetDir $targetDir }
        default   { throw "Unknown asset kind: $($resolved.Kind)" }
    }
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

    # Installation Execution
    Write-Host ""
    Write-Host "Installing selected projects..."
    foreach ($projectName in $selected) {
        Write-Host ""
        Write-Host "Installing $projectName..."
        try {
            $tag = Resolve-LatestRelease -owner $GITHUB_OWNER -proj $projectName
            Write-Host "  -> Resolved tag: $tag"

            $resolved = Resolve-ReleaseAsset -owner $GITHUB_OWNER -proj $projectName -tag $tag -arch $ARCH
            Write-Host "  -> Resolved asset: $($resolved.AssetName) ($($resolved.Kind))"

            Install-ReleaseAsset -resolved $resolved -proj $projectName -targetDir $targetDir
        }
        catch {
            Write-Error "Failed to install $projectName: $_"
        }
    }

    Write-Host ""
    Write-Host "Installation process complete."
    Write-Host ""

    exit 0
}
catch {
    Write-Error $_.Exception.Message
    exit 1
}
