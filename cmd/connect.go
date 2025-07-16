package cmd

import (
	"fmt"

	"github.com/justseemore/sshm/pkg/config"
	"github.com/justseemore/sshm/pkg/ssh"
	"github.com/spf13/cobra"
)

var connectCmd = &cobra.Command{
	Use:   "connect [alias]",
	Short: "Connect to a server using an alias",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias := args[0]

		conn, err := config.GetConnection(alias)
		if err != nil {
			return fmt.Errorf("error getting connection: %w", err)
		}

		fmt.Printf("Connecting to %s (%s@%s:%d)...\n", alias, conn.User, conn.Host, conn.Port)
		return ssh.Connect(conn)
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
}
