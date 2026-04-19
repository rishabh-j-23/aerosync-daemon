package ui

import (
	"aerosync-service/internal/config"
	"aerosync-service/internal/tui"
	"fmt"
)

// BackupMenu displays all configured labels for restoration
func (ui *AerosyncUI) BackupMenu() {
	for {
		m := tui.NewMenu("Select a Backup to Manage")
		
		exitMenu := false
		for _, sp := range ui.Config.SyncPaths {
			currentPath := sp
			m.AddItem(fmt.Sprintf("%s (%s)", sp.Label, sp.Path), func() error {
				ui.ActionMenu(currentPath)
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

// ActionMenu handles individual actions for a specific backup label
func (ui *AerosyncUI) ActionMenu(sp config.SyncPath) {
	m := tui.NewMenu(fmt.Sprintf("Actions for %s", sp.Label))
	
	m.AddItem(fmt.Sprintf("Restore '%s' to Original Location", sp.Label), func() error {
		fmt.Printf("\nRestoring ALL files for %s...\n", sp.Label)
		if err := ui.Service.Restore(sp.Path, ""); err != nil {
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
		if err := ui.Service.Restore(sp.Path, target); err != nil {
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
