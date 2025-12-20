package cmd

import (
	"aerosync-service/internal/config"
	"aerosync-service/internal/service"
	"aerosync-service/internal/sync"
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check service status",
	Long:  `Check if the service is configured and running.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("Service not configured: %v\n", err)
			return
		}

		fmt.Printf("Service configured with provider: %s, sync paths: %v\n", cfg.Provider, cfg.SyncPaths)

		// Check if running
		var provider sync.CloudProvider
		switch cfg.Provider {
		case "gdrive":
			p, err := sync.NewGDriveProvider()
			if err != nil {
				fmt.Printf("Service not ready: %v\n", err)
				return
			}
			provider = p
		default:
			fmt.Printf("Unsupported provider: %s\n", cfg.Provider)
			return
		}

		svc := service.NewService(cfg, provider)
		if svc.IsRunning() {
			fmt.Println("Service status: running")
		} else {
			fmt.Println("Service status: stopped")
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
