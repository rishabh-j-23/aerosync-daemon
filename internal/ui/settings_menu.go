package ui

import (
	"aerosync-service/internal/tui"
	"fmt"
)

// SettingsMenu handles global configuration adjustments
func (ui *AerosyncUI) SettingsMenu() {
	for {
		m := tui.NewMenu("Aerosync Settings")

		m.AddItem(fmt.Sprintf("Sync Interval: %s", ui.Config.SyncInterval), func() error {
			newVal := tui.Prompt("\nEnter new sync interval (e.g., 1h, 30m): ")
			if newVal != "" {
				ui.Config.SyncInterval = newVal
				ui.Config.Save()
				fmt.Println("Setting updated.")
				tui.WaitForEnter()
			}
			return nil
		})

		m.AddItem(fmt.Sprintf("Versioning: %v", ui.Config.Versioning), func() error {
			ui.Config.Versioning = !ui.Config.Versioning
			ui.Config.Save()
			fmt.Printf("Versioning is now %v\n", ui.Config.Versioning)
			tui.WaitForEnter()
			return nil
		})

		m.AddItem(fmt.Sprintf("Max Versions: %d", ui.Config.MaxVersions), func() error {
			newVal := tui.Prompt("\nEnter max versions to keep: ")
			if newVal != "" {
				var count int
				fmt.Sscanf(newVal, "%d", &count)
				if count > 0 {
					ui.Config.MaxVersions = count
					ui.Config.Save()
					fmt.Println("Setting updated.")
					tui.WaitForEnter()
				}
			}
			return nil
		})

		m.AddItem("Manage Sync Paths", func() error {
			ui.SyncPathsMenu()
			return nil
		})

		exitMenu := false
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
