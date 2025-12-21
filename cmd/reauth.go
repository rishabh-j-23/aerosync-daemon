package cmd

import (
	"aerosync-service/internal/sync"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var reauthCmd = &cobra.Command{
	Use:   "reauth",
	Short: "Re-authenticate with Google Drive",
	Long:  `Force re-authentication with Google Drive by obtaining a new OAuth token.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Starting re-authentication with Google Drive...")

		// Force re-auth by removing existing token
		configDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Failed to get home dir: %v\n", err)
			return
		}
		tokenPath := filepath.Join(configDir, ".config", "aerosync", "token.json")
		if err := os.Remove(tokenPath); err != nil && !os.IsNotExist(err) {
			fmt.Printf("Failed to remove existing token: %v\n", err)
			return
		}

		provider, err := sync.NewGDriveProvider()
		if err != nil {
			fmt.Printf("Re-authentication failed: %v\n", err)
			return
		}

		// The NewGDriveProvider will now get a fresh token
		_ = provider

		fmt.Println("Successfully re-authenticated with Google Drive")
	},
}

func init() {
	rootCmd.AddCommand(reauthCmd)
}
