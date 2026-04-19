package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var createTaskCmd = &cobra.Command{
	Use:   "create-task",
	Short: "Create startup task for automatic launch",
	Long:  `Create a startup task that launches the aerosync service automatically on system boot/login.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Creating startup task...")

		if runtime.GOOS == "windows" {
			createWindowsTask()
		} else if runtime.GOOS == "linux" {
			createLinuxTask()
		} else {
			fmt.Printf("Unsupported OS: %s\n", runtime.GOOS)
			os.Exit(1)
		}
	},
}

func createWindowsTask() {
	// Get the executable path
	exePath, err := os.Executable()
	if err != nil {
		fmt.Printf("Failed to get executable path: %v\n", err)
		os.Exit(1)
	}

	// PowerShell command to create scheduled task
	psCmd := fmt.Sprintf(`
$taskName = "AerosyncService"
$existingTask = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
if ($existingTask) {
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false
}
$action = New-ScheduledTaskAction -Execute "%s" -Argument "start"
$trigger = New-ScheduledTaskTrigger -AtLogOn
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable
$principal = New-ScheduledTaskPrincipal -UserId $env:USERNAME -LogonType InteractiveToken
Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Description "Starts Aerosync background sync service on user logon"
`, exePath)

	cmd := exec.Command("powershell", "-Command", psCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to create task: %v\n%s\n", err, string(output))
		os.Exit(1)
	}

	fmt.Println("✅ Startup task created successfully!")
	fmt.Println("Run 'aerosync delete-task' to remove the startup task.")
}

func createLinuxTask() {
	// Get the executable path
	exePath, err := os.Executable()
	if err != nil {
		fmt.Printf("Failed to get executable path: %v\n", err)
		os.Exit(1)
	}

	user := os.Getenv("USER")
	if user == "" {
		user = "user"
	}

	serviceContent := fmt.Sprintf(`[Unit]
Description=Aerosync Background Sync Service
After=network.target

[Service]
Type=simple
User=%s
ExecStart=%s start
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target`, user, exePath)

	// Write service file
	serviceFile := "/etc/systemd/system/aerosync.service"
	file, err := os.CreateTemp("", "aerosync-service")
	if err != nil {
		fmt.Printf("Failed to create temp file: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(file.Name())

	_, err = file.WriteString(serviceContent)
	if err != nil {
		fmt.Printf("Failed to write service file: %v\n", err)
		os.Exit(1)
	}
	file.Close()

	// Copy to /etc/systemd/system/
	cmd := exec.Command("sudo", "cp", file.Name(), serviceFile)
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to copy service file: %v\n", err)
		os.Exit(1)
	}

	// Reload systemd
	cmd = exec.Command("sudo", "systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to reload systemd: %v\n", err)
		os.Exit(1)
	}

	// Enable service
	cmd = exec.Command("sudo", "systemctl", "enable", "aerosync.service")
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to enable service: %v\n", err)
		os.Exit(1)
	}

	// Start service
	cmd = exec.Command("sudo", "systemctl", "start", "aerosync.service")
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to start service: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Startup task created successfully!")
	fmt.Println("Run 'aerosync delete-task' to remove the startup task.")
}

func init() {
	rootCmd.AddCommand(createTaskCmd)
}
