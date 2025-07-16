package cmd

import (
	"fmt"

	"github.com/justseemore/sshm/pkg/config"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [alias]",
	Short: "Delete an SSH connection",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias := args[0]

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("error loading config: %w", err)
		}

		if _, exists := cfg.Connections[alias]; !exists {
			return fmt.Errorf("connection with alias '%s' not found", alias)
		}

		// Delete connection
		delete(cfg.Connections, alias)

		// Save config
		if err := config.SaveConfig(cfg); err != nil {
			return fmt.Errorf("error saving config: %w", err)
		}

		fmt.Printf("Connection '%s' deleted successfully.\n", alias)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
