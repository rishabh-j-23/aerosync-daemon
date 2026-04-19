# Enable automatic startup for aerosync-service on Windows

Write-Host "Enabling automatic startup for aerosync-service..."

# Check if binary exists
$installPath = "$env:USERPROFILE\bin\aerosync-service.exe"
if (!(Test-Path $installPath)) {
    Write-Host "❌ Binary not found: $installPath"
    Write-Host "Run .\scripts\install.ps1 first."
    exit 1
}

# Create scheduled task for automatic startup
$taskName = "AerosyncService"
$existingTask = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
if ($existingTask) {
    Write-Host "Scheduled task '$taskName' already exists. Removing it first..."
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false
}

Write-Host "Creating scheduled task '$taskName'..."
$action = New-ScheduledTaskAction -Execute $installPath -Argument "start"
$trigger = New-ScheduledTaskTrigger -AtLogOn
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable
$principal = New-ScheduledTaskPrincipal -UserId $env:USERNAME -LogonType InteractiveToken

Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Description "Starts Aerosync background sync service on user logon"

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Startup task created successfully!"
    Write-Host "The service will start automatically on user logon."
    Write-Host "Run '.\scripts\disable-startup.ps1' to disable automatic startup."
} else {
    Write-Host "❌ Failed to create startup task"
    exit 1
}