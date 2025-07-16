package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/justseemore/sshm/pkg/config"
	"github.com/spf13/cobra"
)

var (
	host         string
	port         int
	user         string
	password     string
	identityFile string
	timeout      string
)

var addCmd = &cobra.Command{
	Use:   "add [alias]",
	Short: "Add a new SSH connection",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias := args[0]

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("error loading config: %w", err)
		}

		if _, exists := cfg.Connections[alias]; exists {
			return fmt.Errorf("connection with alias '%s' already exists", alias)
		}

		// Expand identity file path
		if identityFile != "" {
			if identityFile[0] == '~' {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("error getting home directory: %w", err)
				}
				identityFile = filepath.Join(homeDir, identityFile[1:])
			}
		}

		// Create new connection
		cfg.Connections[alias] = config.Connection{
			Host:         host,
			Port:         port,
			User:         user,
			Password:     password,
			IdentityFile: identityFile,
			Timeout:      timeout,
		}

		// Save config
		if err := config.SaveConfig(cfg); err != nil {
			return fmt.Errorf("error saving config: %w", err)
		}

		fmt.Printf("Connection '%s' added successfully.\n", alias)
		return nil
	},
}

func init() {
	addCmd.Flags().StringVarP(&host, "host", "H", "", "Host address (required)")
	addCmd.Flags().IntVarP(&port, "port", "p", 22, "Port number")
	addCmd.Flags().StringVarP(&user, "user", "u", "", "Username (required)")
	addCmd.Flags().StringVarP(&password, "password", "P", "", "Password (not recommended, use identity file instead)")
	addCmd.Flags().StringVarP(&identityFile, "identity-file", "i", "", "Identity file path")
	addCmd.Flags().StringVarP(&timeout, "timeout", "t", "10s", "Connection timeout")

	addCmd.MarkFlagRequired("host")
	addCmd.MarkFlagRequired("user")

	rootCmd.AddCommand(addCmd)
}
