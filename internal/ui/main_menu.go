package ui

import (
	"aerosync-service/internal/config"
	"aerosync-service/internal/service"
	"aerosync-service/internal/sync"
	"aerosync-service/internal/tui"
)

// AerosyncUI holds the state for the TUI application
type AerosyncUI struct {
	Config   *config.Config
	Provider sync.CloudProvider
	Service  *service.Service
}

// NewAerosyncUI initializes a new UI session
func NewAerosyncUI(cfg *config.Config, p sync.CloudProvider) *AerosyncUI {
	return &AerosyncUI{
		Config:   cfg,
		Provider: p,
		Service:  service.NewService(cfg, p),
	}
}

// MainMenu is the entry point for the hierarchical TUI
func (ui *AerosyncUI) MainMenu() {
	tui.RunMenu(func() *tui.Menu {
		m := tui.NewMenu("Aerosync Main Menu")

		m.AddItem("Backups", func() error {
			ui.BackupMenu()
			return nil
		})

		m.AddItem("Settings", func() error {
			ui.SettingsMenu()
			return nil
		})

		m.AddItem("Exit", func() error {
			return tui.ErrExit
		})

		return m
	})
}
