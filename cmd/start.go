package cmd

import (
	"aerosync-service/internal/config"
	"aerosync-service/internal/service"
	"aerosync-service/internal/sync"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the aerosync service",
	Long:  `Start the background sync service.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		var provider sync.CloudProvider
		switch cfg.Provider {
		case "gdrive":
			p, err := sync.NewGDriveProvider()
			if err != nil {
				log.Fatalf("Failed to create GDrive provider: %v", err)
			}
			provider = p
		default:
			log.Fatalf("Unsupported provider: %s", cfg.Provider)
		}

		svc := service.NewService(cfg, provider)
		svc.Start()

		// Wait for interrupt signal
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		svc.Stop()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
