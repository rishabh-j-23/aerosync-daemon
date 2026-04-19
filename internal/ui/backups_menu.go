package ui

import (
	"aerosync-service/internal/config"
	"aerosync-service/internal/sync"
	"aerosync-service/internal/tui"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// BackupMenu displays all configured labels for restoration
func (ui *AerosyncUI) BackupMenu() {
	tui.RunMenu(func() *tui.Menu {
		m := tui.NewMenu("Select a Backup to Manage")

		for _, sp := range ui.Config.SyncPaths {
			currentPath := sp
			m.AddItem(fmt.Sprintf("%s (%s)", sp.Label, sp.Path), func() error {
				ui.ActionMenu(currentPath)
				return nil
			})
		}

		m.AddItem("Back to Main Menu", func() error {
			return tui.ErrExit
		})

		return m
	})
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
	
	m.AddItem("Browse Files & Restore Versions...", func() error {
		ui.BrowseVersionsMenu(sp)
		return nil
	})

	m.AddItem("Back to Backup List", func() error {
		return nil
	})

	m.Display()
}

func (ui *AerosyncUI) BrowseVersionsMenu(sp config.SyncPath) {
	files, err := ui.Provider.ListRemote(context.Background(), sp.Label)
	if err != nil {
		fmt.Printf("Error listing files: %v\n", err)
		tui.WaitForEnter()
		return
	}

	tui.RunMenu(func() *tui.Menu {
		m := tui.NewMenu(fmt.Sprintf("Files in %s", sp.Label))
		for _, f := range files {
			currentFile := f
			m.AddItem(f.Path, func() error {
				ui.FileVersionsMenu(sp.Label, currentFile)
				return nil
			})
		}
		m.AddItem("Back", func() error { return tui.ErrExit })
		return m
	})
}

func (ui *AerosyncUI) FileVersionsMenu(label string, file sync.RemoteFile) {
	versions, err := ui.Provider.GetFileVersions(context.Background(), label, file.Path)
	if err != nil {
		fmt.Printf("Error getting versions: %v\n", err)
		tui.WaitForEnter()
		return
	}

	tui.RunMenu(func() *tui.Menu {
		m := tui.NewMenu(fmt.Sprintf("Versions for %s", file.Path))
		
		// Latest version
		m.AddItem("Latest Version (Current)", func() error {
			target := tui.Prompt("Enter restore destination (leave blank for local sync path): ")
			if target == "" {
				target = filepath.Join(ui.Config.GetSyncPath(label), file.Path)
			}
			fmt.Println("Restoring latest...")
			if err := ui.Provider.RestoreSpecific(context.Background(), file.DriveID, target); err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("Restored to: %s\n", target)
			}
			tui.WaitForEnter()
			return nil
		})

		for _, v := range versions {
			currentVer := v
			timeStr := strings.TrimSpace(strings.ReplaceAll(fmt.Sprintf("%v", (time.Unix(v.Timestamp, 0))), "+0000 UTC", ""))
			m.AddItem(fmt.Sprintf("Version %d (%s)", v.Number, timeStr), func() error {
				target := tui.Prompt("Enter restore destination: ")
				if target == "" {
					fmt.Println("Destination required for historical versions.")
					tui.WaitForEnter()
					return nil
				}
				fmt.Println("Restoring version...")
				if err := ui.Provider.RestoreSpecific(context.Background(), currentVer.DriveID, target); err != nil {
					fmt.Printf("Error: %v\n", err)
				} else {
					fmt.Printf("Restored version %d to: %s\n", currentVer.Number, target)
				}
				tui.WaitForEnter()
				return nil
			})
		}
		m.AddItem("Back", func() error { return tui.ErrExit })
		return m
	})
}
