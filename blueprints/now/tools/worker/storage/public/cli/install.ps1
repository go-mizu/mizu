# Liteio Storage CLI installer for Windows
# Usage: irm https://storage.liteio.dev/cli/install.ps1 | iex
#
# Downloads the storage CLI binary from R2 (via signed redirect)
# and installs it to %LOCALAPPDATA%\Programs\storage, adding it to PATH.
#
# Environment variables:
#   STORAGE_VERSION   Pin to a specific version (default: latest)
#   INSTALL_DIR       Override install directory

$ErrorActionPreference = "Stop"

$BaseUrl = "https://storage.liteio.dev/cli/releases"
$Version = if ($env:STORAGE_VERSION) { $env:STORAGE_VERSION } else { "latest" }

function Get-Arch {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
    switch ($arch) {
        "X64"   { return "amd64" }
        "Arm64" { return "arm64" }
        default { throw "Unsupported architecture: $arch" }
    }
}

function Install-Storage {
    $arch = Get-Arch
    $filename = "storage-windows-${arch}.exe"
    $downloadUrl = "${BaseUrl}/${Version}/${filename}"

    # Install to user's local AppData (no admin needed)
    $installDir = if ($env:INSTALL_DIR) {
        $env:INSTALL_DIR
    } else {
        Join-Path $env:LOCALAPPDATA "Programs\storage"
    }

    if (-not (Test-Path $installDir)) {
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    }

    $installPath = Join-Path $installDir "storage.exe"

    Write-Host ""
    Write-Host "  Liteio Storage CLI installer" -ForegroundColor White
    Write-Host ""
    Write-Host "  OS: windows, Arch: $arch" -ForegroundColor DarkGray
    Write-Host ""

    # Download (follows redirects to signed R2 URL)
    Write-Host "  Downloading $downloadUrl" -ForegroundColor Green
    $tmpFile = Join-Path $env:TEMP "storage-$(Get-Random).exe"

    try {
        # MaximumRedirection ensures we follow the signed URL redirect
        Invoke-WebRequest -Uri $downloadUrl -OutFile $tmpFile -UseBasicParsing -MaximumRedirection 5
    } catch {
        Write-Host ""
        Write-Host "  error: Download failed." -ForegroundColor Red
        Write-Host "  $_" -ForegroundColor DarkGray
        Write-Host "  Visit https://storage.liteio.dev/cli for help." -ForegroundColor DarkGray
        exit 1
    }

    # Verify download is a real binary (not an error page)
    $fileSize = (Get-Item $tmpFile).Length
    if ($fileSize -lt 1000) {
        $content = Get-Content $tmpFile -Raw -ErrorAction SilentlyContinue
        if ($content -match "not_found|error") {
            Remove-Item $tmpFile -ErrorAction SilentlyContinue
            Write-Host "  error: Binary not available for windows/$arch" -ForegroundColor Red
            Write-Host "  Visit https://storage.liteio.dev/cli for alternatives." -ForegroundColor DarkGray
            exit 1
        }
    }

    # Install
    Move-Item -Path $tmpFile -Destination $installPath -Force

    # Add to user PATH if not already there
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$installDir*") {
        [Environment]::SetEnvironmentVariable("Path", "$installDir;$userPath", "User")
        $env:Path = "$installDir;$env:Path"
        Write-Host "  Added $installDir to PATH" -ForegroundColor Green
    }

    # Verify
    if (Test-Path $installPath) {
        Write-Host ""
        Write-Host "  Installed storage to $installPath" -ForegroundColor Green

        try {
            $ver = & $installPath --version 2>&1
            Write-Host "  $ver" -ForegroundColor DarkGray
        } catch {}

        Write-Host ""
        Write-Host "  Get started:" -ForegroundColor White
        Write-Host "    storage login"
        Write-Host "    storage --help"
        Write-Host ""
    } else {
        Write-Host "  error: Installation failed" -ForegroundColor Red
        exit 1
    }
}

Install-Storage
