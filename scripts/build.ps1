# Build script for Windows (PowerShell)

Write-Host "Building aerosync-service for Windows..."

# Create bin directory if it doesn't exist
if (!(Test-Path bin)) {
    New-Item -ItemType Directory -Path bin | Out-Null
}

# Build for Windows
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go mod tidy
go build -o bin\aerosync-service.exe -v

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful: bin\aerosync-service.exe"
} else {
    Write-Host "Build failed"
    exit 1
}
