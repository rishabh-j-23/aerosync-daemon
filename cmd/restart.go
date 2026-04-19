package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the aerosync service",
	Long:  `Stop the running sync service and start it again.`,
	Run: func(cmd *cobra.Command, args []string) {
		pidFile := "aerosync.pid"
		data, err := os.ReadFile(pidFile)
		if err == nil {
			pid, err := strconv.Atoi(string(data))
			if err == nil {
				process, err := os.FindProcess(pid)
				if err == nil {
					fmt.Printf("Stopping service (PID %d)...\n", pid)
					_ = process.Signal(syscall.SIGTERM)
					
					// Wait for it to stop
					for i := 0; i < 10; i++ {
						err := process.Signal(syscall.Signal(0))
						if err != nil {
							break
						}
						time.Sleep(500 * time.Millisecond)
					}
					os.Remove(pidFile)
				}
			}
		}

		fmt.Println("Starting service...")
		// Start anew - use the same binary but with 'start' argument
		executable, _ := os.Executable()
		newCmd := exec.Command(executable, "start")
		newCmd.Stdout = os.Stdout
		newCmd.Stderr = os.Stderr
		
		if err := newCmd.Start(); err != nil {
			fmt.Printf("Failed to restart service: %v\n", err)
			return
		}

		fmt.Println("Service restarted successfully")
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
