package ui

import (
	"aerosync-service/internal/autostart"
	"aerosync-service/internal/tui"
	"fmt"
	"strings"
	"time"
)

// StatusMenu provides a live dashboard of service health and synchronization metrics
func (ui *AerosyncUI) StatusMenu() {
	tui.RunMenu(func() *tui.Menu {
		m := tui.NewMenu("System Health & Status")

		// 1. Background Process Status
		svcStatus := "🔴 STOPPED"
		if ui.Service.IsRunning() {
			svcStatus = "🟢 RUNNING"
		}
		m.AddItem(fmt.Sprintf("Background Service: %s", svcStatus), func() error {
			fmt.Printf("\nService is currently %s.\n", svcStatus)
			tui.WaitForEnter()
			return nil
		})

		// 2. Scheduler Alignment
		autoStatus := "⚪ DISABLED"
		if autostart.IsEnabled() {
			autoStatus = "🔵 ENABLED"
		}
		m.AddItem(fmt.Sprintf("Auto-start on Login: %s", autoStatus), func() error {
			fmt.Printf("\nOS Scheduler status: %s\n", autoStatus)
			tui.WaitForEnter()
			return nil
		})

		// 3. Sync Timestamps
		lastStart, _ := ui.Provider.GetStatus("last_sync_start")
		lastSuccess, _ := ui.Provider.GetStatus("last_sync_success")

		m.AddItem(fmt.Sprintf("Last Sync Attempt: %s", formatTime(lastStart)), func() error { return nil })
		m.AddItem(fmt.Sprintf("Last Sync Success: %s", formatTime(lastSuccess)), func() error { return nil })

		m.AddItem("Refresh Dashboard", func() error {
			return nil // Just triggers a rebuild of the menu
		})

		m.AddItem("Back to Main Menu", func() error {
			return tui.ErrExit
		})

		return m
	})
}

func formatTime(raw string) string {
	if raw == "" {
		return "Never"
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	
	// Pretty format: "Apr 19, 21:44:20"
	return strings.ReplaceAll(t.Format("Jan 02, 15:04:05"), "UTC", "")
}
