$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

param(
    [string]$InstallDir = "",
    [switch]$NoPathUpdate
)

function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message"
}

function Write-Success {
    param([string]$Message)
    Write-Host "[OK] $Message" -ForegroundColor Green
}

function Resolve-InstallDir {
    param([string]$Configured)

    if ($Configured -ne "") {
        return $Configured
    }

    return Join-Path $env:LOCALAPPDATA "Programs\GripMock\bin"
}

function Get-LatestVersion {
    $apiUrl = "https://api.github.com/repos/bavix/gripmock/releases/latest"
    $headers = @{}
    if ($env:GITHUB_TOKEN) {
        $headers["Authorization"] = "token $($env:GITHUB_TOKEN)"
    }

    Write-Info "Fetching latest GripMock release"
    $response = Invoke-RestMethod -Uri $apiUrl -Headers $headers
    if (-not $response.tag_name) {
        throw "Unable to resolve latest release tag"
    }

    return $response.tag_name.TrimStart("v")
}

function Get-Arch {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
    switch ($arch) {
        "X64" { return "amd64" }
        "Arm64" { return "arm64" }
        default { throw "Unsupported architecture: $arch" }
    }
}

function Ensure-PathContains {
    param([string]$Dir)

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $parts = @()
    if ($userPath) {
        $parts = $userPath -split ";"
    }

    if ($parts -contains $Dir) {
        return
    }

    $newPath = if ($userPath) { "$userPath;$Dir" } else { $Dir }
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    Write-Info "Added '$Dir' to user PATH. Restart terminal to apply."
}

$resolvedInstallDir = Resolve-InstallDir -Configured $InstallDir
$arch = Get-Arch
$version = Get-LatestVersion

$assetName = "gripmock_${version}_windows_${arch}.zip"
$releaseBaseUrl = "https://github.com/bavix/gripmock/releases/download/v$version"
$assetUrl = "$releaseBaseUrl/$assetName"
$checksumUrl = "$releaseBaseUrl/checksums.txt"

$tempDir = Join-Path $env:TEMP ("gripmock-installer-" + [Guid]::NewGuid())
New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

try {
    $archivePath = Join-Path $tempDir $assetName
    $checksumsPath = Join-Path $tempDir "checksums.txt"

    Write-Info "Downloading checksums"
    Invoke-WebRequest -Uri $checksumUrl -OutFile $checksumsPath

    Write-Info "Downloading $assetName"
    Invoke-WebRequest -Uri $assetUrl -OutFile $archivePath

    $expectedLine = Select-String -Path $checksumsPath -Pattern ([Regex]::Escape($assetName)) | Select-Object -First 1
    if (-not $expectedLine) {
        throw "Checksum for '$assetName' not found"
    }

    $expectedHash = ($expectedLine.Line -split '\s+')[0].ToLowerInvariant()
    $actualHash = (Get-FileHash -Path $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($expectedHash -ne $actualHash) {
        throw "Checksum mismatch for '$assetName'"
    }
    Write-Success "Checksum validated"

    $extractDir = Join-Path $tempDir "extract"
    Expand-Archive -Path $archivePath -DestinationPath $extractDir -Force

    $binarySource = Join-Path $extractDir "gripmock.exe"
    if (-not (Test-Path $binarySource)) {
        throw "Binary not found after archive extraction"
    }

    New-Item -ItemType Directory -Path $resolvedInstallDir -Force | Out-Null
    $binaryTarget = Join-Path $resolvedInstallDir "gripmock.exe"
    Copy-Item -Path $binarySource -Destination $binaryTarget -Force
    Write-Success "Installed to $binaryTarget"

    if (-not $NoPathUpdate) {
        Ensure-PathContains -Dir $resolvedInstallDir
    }

    & $binaryTarget --version
    Write-Success "GripMock installation complete"
}
finally {
    Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
}
