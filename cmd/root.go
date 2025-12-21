package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "aerosync",
	Short: "Aerosync background sync service with versioning",
	Long:  `Aerosync is a background service for syncing local paths to cloud storage with automatic versioning and restore capabilities.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
