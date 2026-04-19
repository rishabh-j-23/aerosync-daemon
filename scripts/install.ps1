# Install script for Windows (PowerShell)

Write-Host "Installing aerosync-service on Windows..."

# Check if binary exists
$binaryPath = "bin\aerosync-service.exe"
if (!(Test-Path $binaryPath)) {
    Write-Host "Binary not found: $binaryPath"
    Write-Host "Run .\build.ps1 first."
    exit 1
}

# Create bin directory in user profile
$installDir = "$env:USERPROFILE\bin"
if (!(Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir | Out-Null
}

# Copy binary
$installPath = "$installDir\aerosync-service.exe"
Copy-Item $binaryPath $installPath

if ($LASTEXITCODE -eq 0) {
    Write-Host "Installation successful!"
    Write-Host "Binary installed to: $installPath"
    Write-Host "Run '.\scripts\enable-startup.ps1' to enable automatic startup on logon."
    Write-Host "Make sure $installDir is in your PATH to run 'aerosync-service.exe --help'"
} else {
    Write-Host "Installation failed"
    exit 1
}
