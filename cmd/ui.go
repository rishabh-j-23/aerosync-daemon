package cmd

import (
	"aerosync-service/internal/config"
	"aerosync-service/internal/sync"
	"aerosync-service/internal/ui"
	"fmt"

	"github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Interactive menu to manage your backups",
	Long:  `A modular, menu-based TUI to manage your backups, sync paths, and settings.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		p, err := sync.NewGDriveProvider()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		// Initialize and start the modular UI
		app := ui.NewAerosyncUI(cfg, p)
		app.MainMenu()
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}
