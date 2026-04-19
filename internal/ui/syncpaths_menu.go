package ui

import (
	"aerosync-service/internal/config"
	"aerosync-service/internal/tui"
	"fmt"
	"strings"
)

// SyncPathsMenu provides a list of all current sync paths for editing
func (ui *AerosyncUI) SyncPathsMenu() {
	for {
		m := tui.NewMenu("Manage Sync Paths")

		for i, sp := range ui.Config.SyncPaths {
			idx := i
			m.AddItem(fmt.Sprintf("Edit: %s (%s)", sp.Label, sp.Path), func() error {
				ui.EditPathMenu(idx)
				return nil
			})
		}

		m.AddItem("Add New Sync Path", func() error {
			path := tui.Prompt("\nEnter local folder path: ")
			if path == "" {
				return nil
			}
			label := tui.Prompt("Enter label for this backup: ")
			if label == "" {
				return nil
			}
			ui.Config.SyncPaths = append(ui.Config.SyncPaths, config.SyncPath{Path: path, Label: label})
			ui.Config.Save()
			fmt.Println("Path added successfully.")
			tui.WaitForEnter()
			return nil
		})

		exitMenu := false
		m.AddItem("Back to Settings", func() error {
			exitMenu = true
			return nil
		})

		selected, _ := m.Display()
		if !selected || exitMenu {
			return
		}
	}
}

// EditPathMenu allows modifying specific details of a single sync path
func (ui *AerosyncUI) EditPathMenu(index int) {
	for {
		sp := ui.Config.SyncPaths[index]
		m := tui.NewMenu(fmt.Sprintf("Edit: %s", sp.Label))

		m.AddItem(fmt.Sprintf("Change Path: %s", sp.Path), func() error {
			newPath := tui.Prompt("\nEnter new local path: ")
			if newPath != "" {
				ui.Config.SyncPaths[index].Path = newPath
				ui.Config.Save()
				fmt.Println("Path updated.")
				tui.WaitForEnter()
			}
			return nil
		})

		m.AddItem(fmt.Sprintf("Change Label: %s", sp.Label), func() error {
			newLabel := tui.Prompt("\nEnter new label: ")
			if newLabel != "" {
				ui.Config.SyncPaths[index].Label = newLabel
				ui.Config.Save()
				fmt.Println("Label updated.")
				tui.WaitForEnter()
			}
			return nil
		})

		m.AddItem("Delete this Sync Path", func() error {
			confirm := tui.Prompt("\nAre you sure you want to delete this sync path? (y/N): ")
			if strings.ToLower(confirm) == "y" {
				ui.Config.SyncPaths = append(ui.Config.SyncPaths[:index], ui.Config.SyncPaths[index+1:]...)
				ui.Config.Save()
				fmt.Println("Path deleted.")
				tui.WaitForEnter()
				return fmt.Errorf("EXIT_SUBMENU") // Trigger return from loop
			}
			return nil
		})

		m.AddItem("Back to Path List", func() error {
			return nil
		})

		selected, err := m.Display()
		if !selected || (err != nil && err.Error() == "EXIT_SUBMENU") {
			return
		}
	}
}
