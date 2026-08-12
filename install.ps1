<#
.SYNOPSIS
    Dotfiles binary installer (Windows / PowerShell 5.1+)
.DESCRIPTION
    Discovers projects from .gitmodules (locally or remotely), prompts for interactive
    project selection and installation directory, resolves latest GitHub release assets
    without the GitHub API, installs direct binary or archive assets, offers fallback
    methods, accounts for every project as Installed/Skipped/Failed, and reports a
    final summary with a meaningful exit status.
#>

$ErrorActionPreference = "Stop"

# 1. Installer Constants
$GITHUB_OWNER = "divijg19"
$GITHUB_REPO = "dotfiles"
$DOTFILES_RAW_URL = "https://raw.githubusercontent.com/$GITHUB_OWNER/$GITHUB_REPO/main"
$DEFAULT_INSTALL_DIR = Join-Path $HOME ".local\bin"

# 2. Architecture Detection
# PROCESSOR_ARCHITEW6432 reveals the native OS architecture when a 32-bit
# PowerShell process runs on 64-bit Windows (PROCESSOR_ARCHITECTURE=x86).
$processArch = $env:PROCESSOR_ARCHITECTURE
$nativeArch = $env:PROCESSOR_ARCHITEW6432

$ARCH = switch ($nativeArch) {
    "AMD64" { "amd64"; break }
    "ARM64" { "arm64"; break }
    default {
        switch ($processArch) {
            "AMD64" { "amd64"; break }
            "ARM64" { "arm64"; break }
            "x86"   { "386"; break }
            "ARM"   { "arm"; break }
            default {
                throw "Unsupported architecture: $processArch"
            }
        }
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
    $tempZip = Join-Path ([System.IO.Path]::GetTempPath()) "$([System.Guid]::NewGuid().ToString()).zip"
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

# 7. Fallback Machinery
function Test-CommandExists {
    param([string]$name)
    return $null -ne (Get-Command $name -ErrorAction SilentlyContinue)
}

function Invoke-Fallback {
    param(
        [string]$proj,
        [string]$repoUrl,
        [string]$targetDir
    )

    Write-Host ""
    Write-Host "Could not fetch a pre-compiled binary for $proj."
    Write-Host "Choose a fallback method:"
    Write-Host "  1) go build (build from local submodule or a shallow on-the-fly clone)"
    Write-Host "  2) go install (remote Go module proxy installation)"
    Write-Host "  3) Custom URL (provide a direct download link)"
    Write-Host "  4) Skip this project"

    $fallbackChoice = $null
    while ($null -eq $fallbackChoice) {
        $choiceInput = Read-Host "Enter choice [1-4] (default 1)"
        if ([string]::IsNullOrWhiteSpace($choiceInput)) {
            $choiceInput = "1"
        }
        if ($choiceInput -in @("1", "2", "3", "4")) {
            $fallbackChoice = $choiceInput
        } else {
            Write-Host "Invalid choice. Please enter 1-4."
        }
    }

    switch ($fallbackChoice) {
        "1" {
            return Invoke-FallbackGoBuild -proj $proj -repoUrl $repoUrl -targetDir $targetDir
        }
        "2" {
            return Invoke-FallbackGoInstall -proj $proj -targetDir $targetDir
        }
        "3" {
            return Invoke-FallbackCustomUrl -proj $proj -targetDir $targetDir
        }
        "4" {
            Write-Host "Skipping $proj."
            return [pscustomobject]@{
                Status  = "Skipped"
                Message = "User selected skip."
            }
        }
    }
}

function Invoke-FallbackGoBuild {
    param(
        [string]$proj,
        [string]$repoUrl,
        [string]$targetDir
    )

    if (-not (Test-CommandExists "go")) {
        Write-Host "Go is not available on this system."
        return [pscustomobject]@{
            Status  = "Failed"
            Message = "Go is not available; cannot build $proj."
        }
    }

    $srcDir = $null
    $tempDir = $null
    try {
        $localSub = if ($PSScriptRoot) { Join-Path $PSScriptRoot "bin\$proj" } else { $null }
        if ($localSub -and (Test-Path (Join-Path $localSub "go.mod"))) {
            $srcDir = $localSub
            Write-Host "Using local submodule for $proj."
        } else {
            $tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
            New-Item -ItemType Directory -Force -Path $tempDir | Out-Null
            if (-not $repoUrl) {
                $repoUrl = "https://github.com/$GITHUB_OWNER/$proj.git"
            }
            Write-Host "Cloning repository for $proj from $repoUrl..."
            & git clone --depth 1 $repoUrl $tempDir
            if ($LASTEXITCODE -ne 0 -or -not (Test-Path (Join-Path $tempDir "go.mod"))) {
                Write-Host "Failed to clone repository for $proj."
                return [pscustomobject]@{
                    Status  = "Failed"
                    Message = "Failed to clone repository for $proj."
                }
            }
            $srcDir = $tempDir
        }

        $buildTarget = "."
        $cmdDir = Join-Path $srcDir "cmd"
        if (Test-Path $cmdDir) {
            $sub = Get-ChildItem -Path $cmdDir -Directory | Select-Object -First 1
            if ($sub) {
                $buildTarget = ".\cmd\$($sub.Name)"
            }
        }

        $targetInput = Read-Host "Enter build target [Default: $buildTarget]"
        if ($targetInput) {
            $buildTarget = $targetInput.Trim()
        }

        $dirInput = Read-Host "Override install path for $proj? [Default: $targetDir]"
        $projInstallDir = if ($dirInput) { $dirInput.Trim() } else { $targetDir }
        New-Item -ItemType Directory -Force -Path $projInstallDir | Out-Null

        $outPath = Join-Path $projInstallDir "$proj.exe"
        Write-Host "Building $proj locally from $buildTarget..."

        Push-Location $srcDir
        try {
            & go build -o $outPath $buildTarget
            $buildRc = $LASTEXITCODE
        }
        finally {
            Pop-Location
        }

        if ($buildRc -eq 0 -and (Test-Path $outPath)) {
            Write-Host "Successfully built and installed $proj to $outPath"
            return [pscustomobject]@{
                Status  = "Installed"
                Message = "Installed via go build."
            }
        }
        Write-Host "Failed to build $proj from $buildTarget."
        return [pscustomobject]@{
            Status  = "Failed"
            Message = "Failed to build $proj from $buildTarget."
        }
    }
    catch {
        Write-Host "go build fallback failed: $($_.Exception.Message)"
        return [pscustomobject]@{
            Status  = "Failed"
            Message = "go build failed: $($_.Exception.Message)"
        }
    }
    finally {
        if ($tempDir -and (Test-Path $tempDir)) {
            Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

function Invoke-FallbackGoInstall {
    param(
        [string]$proj,
        [string]$targetDir
    )

    if (-not (Test-CommandExists "go")) {
        Write-Host "Go is not available on this system."
        return [pscustomobject]@{
            Status  = "Failed"
            Message = "Go is not available; cannot go install $proj."
        }
    }

    $modPath = Read-Host "Enter Go module path (e.g., github.com/$GITHUB_OWNER/$proj@latest)"
    if ([string]::IsNullOrWhiteSpace($modPath)) {
        Write-Host "Module path cannot be empty."
        return [pscustomobject]@{
            Status  = "Failed"
            Message = "Module path cannot be empty."
        }
    }
    $modPath = $modPath.Trim()

    $oldGOBIN = $env:GOBIN
    $projInstallDir = $targetDir
    try {
        $dirInput = Read-Host "Override install path for $proj? [Default: $targetDir]"
        if ($dirInput) {
            $projInstallDir = $dirInput.Trim()
        }
        New-Item -ItemType Directory -Force -Path $projInstallDir | Out-Null

        $env:GOBIN = $projInstallDir
        Write-Host "Running go install for $modPath..."
        & go install $modPath
        $installRc = $LASTEXITCODE
        $destPath = Join-Path $projInstallDir "$proj.exe"
        if ($installRc -eq 0 -and (Test-Path $destPath)) {
            Write-Host "Successfully installed $proj via go install to $projInstallDir"
            return [pscustomobject]@{
                Status  = "Installed"
                Message = "Installed via go install."
            }
        }
        Write-Host "Failed to install $proj via go install."
        return [pscustomobject]@{
            Status  = "Failed"
            Message = "go install failed: $modPath."
        }
    }
    catch {
        Write-Host "go install fallback failed: $($_.Exception.Message)"
        return [pscustomobject]@{
            Status  = "Failed"
            Message = "go install failed: $($_.Exception.Message)"
        }
    }
    finally {
        $env:GOBIN = $oldGOBIN
    }
}

function Invoke-FallbackCustomUrl {
    param(
        [string]$proj,
        [string]$targetDir
    )

    $customUrl = Read-Host "Enter download URL"
    if ([string]::IsNullOrWhiteSpace($customUrl)) {
        Write-Host "URL cannot be empty."
        return [pscustomobject]@{
            Status  = "Failed"
            Message = "URL cannot be empty."
        }
    }
    $customUrl = $customUrl.Trim()

    $tempFile = [System.IO.Path]::GetTempFileName()
    try {
        Write-Host "Downloading binary from $customUrl..."
        Invoke-WebRequest -Uri $customUrl -OutFile $tempFile -UseBasicParsing
        if (-not (Test-Path $tempFile) -or (Get-Item $tempFile).Length -eq 0) {
            Write-Host "Downloaded binary is empty or missing."
            return [pscustomobject]@{
                Status  = "Failed"
                Message = "Downloaded binary from custom URL is empty or missing."
            }
        }
        $destPath = Join-Path $targetDir "$proj.exe"
        Copy-Item -Path $tempFile -Destination $destPath -Force
        Write-Host "Successfully downloaded and installed $proj to $destPath"
        return [pscustomobject]@{
            Status  = "Installed"
            Message = "Installed via custom URL."
        }
    }
    catch {
        Write-Host "Custom URL download failed: $($_.Exception.Message)"
        return [pscustomobject]@{
            Status  = "Failed"
            Message = "Custom URL download failed: $($_.Exception.Message)"
        }
    }
    finally {
        if (Test-Path $tempFile) {
            Remove-Item -Path $tempFile -Force -ErrorAction SilentlyContinue
        }
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
    $selectedUrls = @{}
    foreach ($p in $projects) {
        $choice = Read-Host "Install $($p.Name)? (y/N)"
        if ($choice -match '^[Yy]$') {
            $selected += $p.Name
            $selectedUrls[$p.Name] = $p.Url
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

    # Outcome State
    $installed = @()
    $skipped = @()
    $failed = @()
    $notes = @{}

    # Installation Execution
    Write-Host ""
    Write-Host "Installing selected projects..."
    foreach ($projectName in $selected) {
        Write-Host ""
        Write-Host "Installing $projectName..."
        $outcome = $null
        $note = ""
        try {
            $tag = Resolve-LatestRelease -owner $GITHUB_OWNER -proj $projectName
            Write-Host "  -> Resolved tag: $tag"

            $resolved = Resolve-ReleaseAsset -owner $GITHUB_OWNER -proj $projectName -tag $tag -arch $ARCH
            Write-Host "  -> Resolved asset: $($resolved.AssetName) ($($resolved.Kind))"

            Install-ReleaseAsset -resolved $resolved -proj $projectName -targetDir $targetDir
            $outcome = "Installed"
            $note = "Installed from release asset."
        }
        catch {
            $note = $_.Exception.Message
            Write-Host "  ! $note"
            $fallbackResult = Invoke-Fallback -proj $projectName -repoUrl $selectedUrls[$projectName] -targetDir $targetDir
            $outcome = $fallbackResult.Status
            $note = "$note; Fallback: $($fallbackResult.Message)"
        }

        switch ($outcome) {
            "Installed" { $installed += $projectName }
            "Skipped"   { $skipped += $projectName }
            default     { $failed += $projectName }
        }
        if ($note) {
            $notes[$projectName] = $note
        }
    }

    # Final Summary
    Write-Host ""
    Write-Host "=========================================="
    Write-Host "Installation Summary"
    Write-Host "=========================================="
    Write-Host ""
    Write-Host "Installed:"
    if ($installed.Count -gt 0) {
        foreach ($p in $installed) { Write-Host "  $p" }
    } else {
        Write-Host "  None"
    }
    Write-Host "Skipped:"
    if ($skipped.Count -gt 0) {
        foreach ($p in $skipped) { Write-Host "  $p" }
    } else {
        Write-Host "  None"
    }
    Write-Host "Failed:"
    if ($failed.Count -gt 0) {
        foreach ($p in $failed) { Write-Host "  $p" }
    } else {
        Write-Host "  None"
    }
    Write-Host ""
    Write-Host "Installation directory:"
    Write-Host "  $targetDir"
    Write-Host ""
    if ($notes.Count -gt 0) {
        Write-Host "Notes:"
        foreach ($key in $notes.Keys) {
            Write-Host "  $key — $($notes[$key])"
        }
        Write-Host ""
    }
    if ($failed.Count -gt 0) {
        Write-Host "Installation completed with issues."
    } else {
        Write-Host "Installation completed successfully."
    }
    Write-Host "Make sure $targetDir is in your PATH."
    Write-Host ""

    if ($failed.Count -gt 0) {
        exit 1
    } else {
        exit 0
    }
}
catch {
    Write-Error $_.Exception.Message
    exit 1
}
