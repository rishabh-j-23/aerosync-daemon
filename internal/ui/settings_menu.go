package ui

import (
	"aerosync-service/internal/autostart"
	"aerosync-service/internal/tui"
	"fmt"
)

// SettingsMenu handles global configuration adjustments
func (ui *AerosyncUI) SettingsMenu() {
	tui.RunMenu(func() *tui.Menu {
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

		status := "Disabled"
		if autostart.IsEnabled() {
			status = "Enabled"
		}
		m.AddItem(fmt.Sprintf("Auto-start on Login: [%s]", status), func() error {
			if autostart.IsEnabled() {
				fmt.Println("\nDisabling auto-start...")
				if err := autostart.Disable(); err != nil {
					fmt.Printf("Error: %v\n", err)
				} else {
					fmt.Println("Auto-start disabled.")
				}
			} else {
				fmt.Println("\nEnabling auto-start...")
				if err := autostart.Enable(); err != nil {
					fmt.Printf("Error: %v\n", err)
				} else {
					fmt.Println("Auto-start enabled.")
				}
			}
			tui.WaitForEnter()
			return nil
		})

		m.AddItem("Back to Main Menu", func() error {
			return tui.ErrExit
		})

		return m
	})
}
