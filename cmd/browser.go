package cmd

import (
	"aerosync-service/internal/config"
	"aerosync-service/internal/service"
	"aerosync-service/internal/sync"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var browserCmd = &cobra.Command{
	Use:   "browser",
	Short: "Interactive menu to manage your backups",
	Long:  `A hierarchical menu-based TUI to manage your backups and settings.`,
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

		mainMenu(cfg, p)
	},
}

func mainMenu(cfg *config.Config, p sync.CloudProvider) {
	for {
		items := []string{
			"📦 Backups",
			"🛑 Exit",
		}

		choice, err := runFzf("Aerosync Main Menu", items)
		if err != nil || choice == "🛑 Exit" || choice == "" {
			fmt.Println("Goodbye!")
			return
		}

		switch choice {
		case "📦 Backups":
			backupMenu(cfg, p)
		}
	}
}

func backupMenu(cfg *config.Config, p sync.CloudProvider) {
	for {
		var items []string
		for _, sp := range cfg.SyncPaths {
			items = append(items, fmt.Sprintf("%s (%s)", sp.Label, sp.Path))
		}
		items = append(items, "⬅️ Back to Main Menu")

		choice, err := runFzf("Select a Backup to Restore", items)
		if err != nil || choice == "⬅️ Back to Main Menu" || choice == "" {
			return
		}

		// Extract label (everything before the first space or parenthesis)
		selectedLabel := strings.Split(choice, " (")[0]
		
		var selectedSyncPath config.SyncPath
		for _, sp := range cfg.SyncPaths {
			if sp.Label == selectedLabel {
				selectedSyncPath = sp
				break
			}
		}

		actionMenu(cfg, p, selectedSyncPath)
	}
}

func actionMenu(cfg *config.Config, p sync.CloudProvider, sp config.SyncPath) {
	for {
		items := []string{
			fmt.Sprintf("✅ Restore '%s' to Original Location", sp.Label),
			fmt.Sprintf("📂 Restore '%s' to Custom Location...", sp.Label),
			"⬅️ Back to Backup List",
		}

		choice, err := runFzf(fmt.Sprintf("Actions for %s", sp.Label), items)
		if err != nil || choice == "⬅️ Back to Backup List" || choice == "" {
			return
		}

		svc := service.NewService(cfg, p)

		if strings.Contains(choice, "Original Location") {
			fmt.Printf("\nRestoring ALL files for %s...\n", sp.Label)
			if err := svc.Restore(sp.Path, ""); err != nil {
				fmt.Printf("Restore failed: %v\n", err)
			} else {
				fmt.Println("Success: All files restored to original location.")
			}
			fmt.Println("\nPress Enter to return...")
			fmt.Scanln()
			return // Return to backup list after action
		} else if strings.Contains(choice, "Custom Location") {
			fmt.Print("\nEnter target folder path: ")
			var target string
			fmt.Scanln(&target)
			if target == "" {
				fmt.Println("Cancelled.")
				continue
			}
			fmt.Printf("Restoring ALL files for %s to %s...\n", sp.Label, target)
			if err := svc.Restore(sp.Path, target); err != nil {
				fmt.Printf("Restore failed: %v\n", err)
			} else {
				fmt.Println("Success: All files restored to custom mapping.")
			}
			fmt.Println("\nPress Enter to return...")
			fmt.Scanln()
			return // Return to backup list after action
		}
	}
}

func runFzf(header string, items []string) (string, error) {
	// Using --reverse for top-down and --inline-info for a compact look
	fzfCmd := exec.Command("fzf", "--header", header, "--reverse", "--height", "20%", "--info", "inline")
	fzfCmd.Stdin = strings.NewReader(strings.Join(items, "\n"))
	fzfCmd.Stderr = os.Stderr
	
	out, err := fzfCmd.Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return "", nil // Cancelled with ESC
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func init() {
	rootCmd.AddCommand(browserCmd)
}
