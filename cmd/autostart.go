package cmd

import (
	"aerosync-service/internal/autostart"
	"fmt"
	"github.com/spf13/cobra"
)

var autostartCmd = &cobra.Command{
	Use:   "autostart",
	Short: "Manage automatic startup of Aerosync on login",
}

var enableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable Aerosync auto-start on login",
	Run: func(cmd *cobra.Command, args []string) {
		if err := autostart.Enable(); err != nil {
			fmt.Printf("Error enabling auto-start: %v\n", err)
		} else {
			fmt.Println("Auto-start enabled successfully.")
		}
	},
}

var disableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable Aerosync auto-start on login",
	Run: func(cmd *cobra.Command, args []string) {
		if err := autostart.Disable(); err != nil {
			fmt.Printf("Error disabling auto-start: %v\n", err)
		} else {
			fmt.Println("Auto-start disabled successfully.")
		}
	},
}

func init() {
	autostartCmd.AddCommand(enableCmd)
	autostartCmd.AddCommand(disableCmd)
	rootCmd.AddCommand(autostartCmd)
}
