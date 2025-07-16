package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sshm",
	Short: "SSH Manager - manage your SSH connections",
	Long: `SSH Manager (sshm) is a simple tool to manage your SSH connections.
It allows you to store your SSH connection details in a YAML file and connect to them using aliases.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Here you will define your flags and configuration settings.
}
