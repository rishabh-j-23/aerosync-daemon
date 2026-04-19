package ui

import (
	"aerosync-service/internal/config"
	"aerosync-service/internal/tui"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// LogViewer launches a live tail of the background service log file
func (ui *AerosyncUI) LogViewer() {
	logFile := filepath.Join(config.GetLogsDir(), "service.log")
	
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		fmt.Println("\n[Status] No log file discovered. Ensure the background service has been started.")
		tui.WaitForEnter()
		return
	}

	tui.ClearScreen()
	fmt.Println("=========================================================")
	fmt.Println("             AEROSYNC LIVE SERVICE LOGS                  ")
	fmt.Println("        (Press Ctrl+C to return to Dashboard)            ")
	fmt.Println("=========================================================")
	fmt.Println()
	
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Use PowerShell's Get-Content -Wait for a robust 'tail -f' equivalent on Windows
		cmd = exec.Command("powershell", "-NoProfile", "-Command", "Get-Content", fmt.Sprintf("'%s'", logFile), "-Tail", "50", "-Wait")
	} else {
		cmd = exec.Command("tail", "-f", logFile)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	// We run it and wait for interrupt
	if err := cmd.Run(); err != nil {
		// Ctrl+C will cause an error exit, which is expected here
		fmt.Print("\nReturning to Dashboard...")
	}
	
	tui.ClearScreen()
	fmt.Println()
}
