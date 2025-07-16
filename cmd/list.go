package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/justseemore/sshm/pkg/config"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured SSH connections",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("error loading config: %w", err)
		}

		if len(cfg.Connections) == 0 {
			fmt.Println("No connections configured. Use 'sshm add' to add a connection.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ALIAS\tHOST\tPORT\tUSER")
		for alias, conn := range cfg.Connections {
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", alias, conn.Host, conn.Port, conn.User)
		}
		return w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
