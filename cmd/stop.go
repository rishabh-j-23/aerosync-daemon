package cmd

import (
	"fmt"
	"os"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the aerosync service",
	Long:  `Stop the running sync service.`,
	Run: func(cmd *cobra.Command, args []string) {
		pidFile := "aerosync.pid"
		data, err := os.ReadFile(pidFile)
		if err != nil {
			fmt.Println("Service does not appear to be running (no PID file)")
			return
		}

		pid, err := strconv.Atoi(string(data))
		if err != nil {
			fmt.Printf("Invalid PID file: %v\n", err)
			return
		}

		process, err := os.FindProcess(pid)
		if err != nil {
			fmt.Printf("Failed to find process %d: %v\n", pid, err)
			return
		}

		if err := process.Signal(syscall.SIGTERM); err != nil {
			fmt.Printf("Failed to send SIGTERM to process %d: %v\n", pid, err)
			return
		}

		fmt.Printf("Sent SIGTERM to service (PID %d)\n", pid)
		os.Remove(pidFile)
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
