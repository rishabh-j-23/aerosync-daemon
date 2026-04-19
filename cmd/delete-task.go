package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var deleteTaskCmd = &cobra.Command{
	Use:   "delete-task",
	Short: "Delete startup task",
	Long:  `Delete the startup task that launches the aerosync service automatically.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Deleting startup task...")

		if runtime.GOOS == "windows" {
			deleteWindowsTask()
		} else if runtime.GOOS == "linux" {
			deleteLinuxTask()
		} else {
			fmt.Printf("Unsupported OS: %s\n", runtime.GOOS)
			os.Exit(1)
		}
	},
}

func deleteWindowsTask() {
	psCmd := `
$taskName = "AerosyncService"
$existingTask = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
if ($existingTask) {
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false
    Write-Host "Scheduled task removed successfully."
} else {
    Write-Host "Scheduled task does not exist."
}
`

	cmd := exec.Command("powershell", "-Command", psCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to delete task: %v\n%s\n", err, string(output))
		os.Exit(1)
	}

	fmt.Println("✅ Startup task deleted successfully!")
}

func deleteLinuxTask() {
	serviceFile := "/etc/systemd/system/aerosync.service"

	// Check if service exists
	if _, err := os.Stat(serviceFile); os.IsNotExist(err) {
		fmt.Println("Service does not exist.")
		fmt.Println("✅ Startup task already deleted.")
		return
	}

	// Stop service
	cmd := exec.Command("sudo", "systemctl", "stop", "aerosync.service")
	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: Failed to stop service: %v\n", err)
	}

	// Disable service
	cmd = exec.Command("sudo", "systemctl", "disable", "aerosync.service")
	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: Failed to disable service: %v\n", err)
	}

	// Remove service file
	cmd = exec.Command("sudo", "rm", serviceFile)
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to remove service file: %v\n", err)
		os.Exit(1)
	}

	// Reload systemd
	cmd = exec.Command("sudo", "systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: Failed to reload systemd: %v\n", err)
	}

	fmt.Println("✅ Startup task deleted successfully!")
}

func init() {
	rootCmd.AddCommand(deleteTaskCmd)
}
