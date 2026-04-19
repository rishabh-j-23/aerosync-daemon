package cmd

import (
	"aerosync-service/internal/config"
	"aerosync-service/internal/service"
	"aerosync-service/internal/sync"
	"aerosync-service/internal/tui"
	"fmt"

	"github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Interactive menu to manage your backups",
	Long:  `A clean, menu-based TUI to browse labels and restore entire backup sets.`,
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

		svc := service.NewService(cfg, p)

		// Start Main Menu
		runMainMenu(cfg, p, svc)
	},
}

func runMainMenu(cfg *config.Config, p sync.CloudProvider, svc *service.Service) {
	for {
		m := tui.NewMenu("Aerosync Main Menu")
		
		exitApp := false
		m.AddItem("Backups", func() error {
			runBackupMenu(cfg, p, svc)
			return nil
		})

		m.AddItem("Settings", func() error {
			runSettingsMenu(cfg)
			return nil
		})
		
		m.AddItem("Exit", func() error {
			exitApp = true
			return nil
		})

		selected, _ := m.Display()
		if !selected || exitApp {
			return
		}
	}
}

func runSettingsMenu(cfg *config.Config) {
	for {
		m := tui.NewMenu("Aerosync Settings")

		m.AddItem(fmt.Sprintf("Sync Interval: %s", cfg.SyncInterval), func() error {
			newVal := tui.Prompt("\nEnter new sync interval (e.g., 1h, 30m): ")
			if newVal != "" {
				cfg.SyncInterval = newVal
				cfg.Save()
				fmt.Println("Setting updated.")
				tui.WaitForEnter()
			}
			return nil
		})

		m.AddItem(fmt.Sprintf("Versioning: %v", cfg.Versioning), func() error {
			cfg.Versioning = !cfg.Versioning
			cfg.Save()
			fmt.Printf("Versioning is now %v\n", cfg.Versioning)
			tui.WaitForEnter()
			return nil
		})

		m.AddItem(fmt.Sprintf("Max Versions: %d", cfg.MaxVersions), func() error {
			newVal := tui.Prompt("\nEnter max versions to keep: ")
			if newVal != "" {
				var count int
				fmt.Sscanf(newVal, "%d", &count)
				if count > 0 {
					cfg.MaxVersions = count
					cfg.Save()
					fmt.Println("Setting updated.")
					tui.WaitForEnter()
				}
			}
			return nil
		})

		m.AddItem("Back to Main Menu", func() error {
			return nil
		})

		selected, _ := m.Display()
		if !selected {
			return
		}
	}
}

func runBackupMenu(cfg *config.Config, p sync.CloudProvider, svc *service.Service) {
	for {
		m := tui.NewMenu("Select a Backup to Manage")
		
		exitMenu := false
		for _, sp := range cfg.SyncPaths {
			currentPath := sp
			m.AddItem(fmt.Sprintf("%s (%s)", sp.Label, sp.Path), func() error {
				runActionMenu(cfg, p, svc, currentPath)
				return nil
			})
		}
		
		m.AddItem("Back to Main Menu", func() error {
			exitMenu = true
			return nil
		})

		selected, _ := m.Display()
		if !selected || exitMenu {
			return
		}
	}
}

func runActionMenu(cfg *config.Config, p sync.CloudProvider, svc *service.Service, sp config.SyncPath) {
	m := tui.NewMenu(fmt.Sprintf("Actions for %s", sp.Label))
	
	m.AddItem(fmt.Sprintf("Restore '%s' to Original Location", sp.Label), func() error {
		fmt.Printf("\nRestoring ALL files for %s...\n", sp.Label)
		if err := svc.Restore(sp.Path, ""); err != nil {
			return err
		}
		fmt.Println("Success: All files restored.")
		tui.WaitForEnter()
		return nil
	})
	
	m.AddItem(fmt.Sprintf("Restore '%s' to Custom Location...", sp.Label), func() error {
		target := tui.Prompt("\nEnter target folder path: ")
		if target == "" {
			return nil
		}
		fmt.Printf("Restoring ALL files for %s to %s...\n", sp.Label, target)
		if err := svc.Restore(sp.Path, target); err != nil {
			return err
		}
		fmt.Println("Success: All files restored to custom mapping.")
		tui.WaitForEnter()
		return nil
	})
	
	m.AddItem("Back to Backup List", func() error {
		return nil
	})

	m.Display()
}

func init() {
	rootCmd.AddCommand(uiCmd)
}
