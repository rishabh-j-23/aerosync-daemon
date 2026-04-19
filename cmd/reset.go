package cmd

import (
	"aerosync-service/internal/config"
	"aerosync-service/internal/sync"
	"context"
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Delete all sync metadata and start fresh",
	Long:  `Stop the service, wipe local database/cache, and delete remote folders on Google Drive.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("CRITICAL WARNING: This will permanently delete ALL synchronization data locally AND on Google Drive (aerosync/ and aerosync_versions/ folders).")
		fmt.Print("Type 'CONFIRM' to proceed: ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "CONFIRM" {
			fmt.Println("Reset aborted.")
			return
		}

		// 1. Stop service
		pidFile := "aerosync.pid"
		if data, err := os.ReadFile(pidFile); err == nil {
			if pid, err := strconv.Atoi(string(data)); err == nil {
				if process, err := os.FindProcess(pid); err == nil {
					fmt.Printf("Stopping running service (PID %d)...\n", pid)
					_ = process.Signal(syscall.SIGTERM)
					time.Sleep(1 * time.Second)
					os.Remove(pidFile)
				}
			}
		}

		// 2. Initialize provider to run cleanup
		fmt.Println("Initializing cleanup...")
		_, err := config.LoadConfig() // Ensure config exists
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			return
		}

		provider, err := sync.NewGDriveProvider()
		if err != nil {
			fmt.Printf("Error initializing GDrive provider: %v\n", err)
			return
		}

		// 3. Execution
		if err := provider.Cleanup(context.Background()); err != nil {
			fmt.Printf("Cleanup failed: %v\n", err)
			return
		}

		fmt.Println("\nSuccess! A complete reset has been performed.")
		fmt.Println("Use 'aerosync start' to begin a brand new synchronization.")
	},
}

func init() {
	rootCmd.AddCommand(resetCmd)
}
