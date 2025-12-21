package cmd

import (
	"aerosync-service/internal/config"
	"aerosync-service/internal/service"
	"aerosync-service/internal/sync"
	"fmt"

	"github.com/spf13/cobra"
)

var restoreCmd = &cobra.Command{
	Use:   "restore [path]",
	Short: "Restore a file or folder from backup",
	Long:  `Restore a specific file or folder from the latest backup version.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]

		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("Service not configured: %v\n", err)
			return
		}

		var provider sync.CloudProvider
		switch cfg.Provider {
		case "gdrive":
			p, err := sync.NewGDriveProvider()
			if err != nil {
				fmt.Printf("Failed to create GDrive provider: %v\n", err)
				return
			}
			provider = p
		default:
			fmt.Printf("Unsupported provider: %s\n", cfg.Provider)
			return
		}

		svc := service.NewService(cfg, provider)
		if err := svc.Restore(path); err != nil {
			fmt.Printf("Restore failed: %v\n", err)
			return
		}

		fmt.Printf("Successfully restored: %s\n", path)
	},
}

func init() {
	rootCmd.AddCommand(restoreCmd)
}
