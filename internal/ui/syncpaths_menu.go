package ui

import (
	"aerosync-service/internal/config"
	"aerosync-service/internal/tui"
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// SyncPathsMenu provides a list of all current sync paths for editing
func (ui *AerosyncUI) SyncPathsMenu() {
	tui.RunMenu(func() *tui.Menu {
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
			path = filepath.ToSlash(path)
			label := tui.Prompt("Enter label for this backup: ")
			if label == "" {
				return nil
			}
			ui.Config.SyncPaths = append(ui.Config.SyncPaths, config.SyncPath{Path: path, Label: label})
			ui.Config.Save()
			fmt.Printf("Path added successfully: %s\n", path)
			tui.WaitForEnter()
			return nil
		})

		m.AddItem("Back to Settings", func() error {
			return tui.ErrExit
		})

		return m
	})
}

// EditPathMenu allows modifying specific details of a single sync path
func (ui *AerosyncUI) EditPathMenu(index int) {
	tui.RunMenu(func() *tui.Menu {
		// Ensure index is still valid (in case skip/delete happened)
		if index >= len(ui.Config.SyncPaths) {
			return tui.NewMenu("Invalid State (Redirected)") // Will exit
		}

		sp := ui.Config.SyncPaths[index]
		m := tui.NewMenu(fmt.Sprintf("Editing: %s", sp.Label))

		m.AddItem("Sync now", func() error {
			fmt.Printf("\nTriggering sync for '%s'...\n", sp.Label)
			if err := ui.Service.SyncLabel(sp.Label); err != nil {
				fmt.Printf("Sync failed: %v\n", err)
			} else {
				fmt.Println("Sync completed successfully.")
			}
			tui.WaitForEnter()
			return nil
		})

		m.AddItem(fmt.Sprintf("Local Path: %s", sp.Path), func() error {
			fmt.Printf("\nCurrent path: %s\n", sp.Path)
			newPath := tui.Prompt("Enter new local path: ")
			if newPath != "" {
				ui.Config.SyncPaths[index].Path = filepath.ToSlash(newPath)
				ui.Config.Save()
				fmt.Println("Path updated.")
				tui.WaitForEnter()
			}
			return nil
		})

		m.AddItem(fmt.Sprintf("Label: %s", sp.Label), func() error {
			oldLabel := sp.Label
			fmt.Printf("\nCurrent label: %s\n", oldLabel)
			newLabel := tui.Prompt("Enter new label: ")
			if newLabel != "" && newLabel != oldLabel {
				fmt.Printf("Renaming folder on cloud from '%s' to '%s'...\n", oldLabel, newLabel)
				if err := ui.Provider.RenameLabel(context.Background(), oldLabel, newLabel); err != nil {
					fmt.Printf("Warning: Failed to rename folder on cloud: %v\n", err)
				} else {
					fmt.Println("Cloud folder renamed successfully.")
				}

				ui.Config.SyncPaths[index].Label = newLabel
				ui.Config.Save()
				fmt.Println("Label updated in config.")
				tui.WaitForEnter()
			}
			return nil
		})

		m.AddItem("Delete this sync path", func() error {
			confirm := tui.Prompt("\nAre you sure you want to delete this sync path? (y/N): ")
			if strings.ToLower(confirm) == "y" {
				ui.Config.SyncPaths = append(ui.Config.SyncPaths[:index], ui.Config.SyncPaths[index+1:]...)
				ui.Config.Save()
				fmt.Println("Path deleted.")
				tui.WaitForEnter()
				return tui.ErrExit // Go back to list
			}
			return nil
		})

		m.AddItem("Back to Path List", func() error {
			return tui.ErrExit
		})

		return m
	})
}
