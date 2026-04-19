# Disable automatic startup for aerosync-service on Windows

Write-Host "Disabling automatic startup for aerosync-service..."

$taskName = "AerosyncService"
$existingTask = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue

if (!$existingTask) {
    Write-Host "❌ Scheduled task '$taskName' does not exist."
    Write-Host "Automatic startup is already disabled."
    exit 0
}

Write-Host "Removing scheduled task '$taskName'..."
Unregister-ScheduledTask -TaskName $taskName -Confirm:$false

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Automatic startup disabled successfully!"
    Write-Host "The service will no longer start automatically on logon."
    Write-Host "Run '.\scripts\enable-startup.ps1' to re-enable automatic startup."
} else {
    Write-Host "❌ Failed to remove startup task"
    exit 1
}