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
		_, _ = fmt.Fprintln(w, "ALIAS\tHOST\tPORT\tPROXY")
		for alias, conn := range cfg.Connections {
			proxyInfo := "none"
			if conn.ProxyType != "" && conn.ProxyType != "none" {
				proxyInfo = fmt.Sprintf("%s://%s:%d", conn.ProxyType, conn.ProxyHost, conn.ProxyPort)
			}
			// 修改输出行
			_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", alias, conn.Host, conn.Port, proxyInfo)
		}
		return w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
