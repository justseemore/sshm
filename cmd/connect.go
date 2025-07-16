package cmd

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/justseemore/sshm/pkg/config"
	"github.com/justseemore/sshm/pkg/ssh"
	"github.com/spf13/cobra"
)

var (
	credentialAlias string // 连接时使用的凭证别名
	connectPort     int    // 直接连接时的端口
	connectUser     string // 直接连接时的用户名
)

var connectCmd = &cobra.Command{
	Use:     "connect [alias|host]",
	Aliases: []string{"login", "l"},
	Short:   "Connect to a server using an alias or directly via IP/hostname",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		// 检查是配置别名还是直接IP/主机名
		isDirectConnect := isIPorHostname(target)

		var conn *config.Connection
		var err error

		if isDirectConnect {
			// 直接使用IP/主机名连接
			port := connectPort
			if port == 0 {
				port = 22 // 默认SSH端口
			}

			// 创建临时连接配置
			conn = &config.Connection{
				Host: target,
				Port: port,
				User: connectUser,
			}

			fmt.Println("Direct connection mode - using IP/hostname without saved configuration")
		} else {
			// 从配置加载连接别名
			conn, err = config.GetConnection(target)
			if err != nil {
				return fmt.Errorf("error getting connection: %w", err)
			}
		}

		// 确定使用哪个凭证
		var cred *config.Credential
		if credentialAlias != "" {
			// 使用命令行指定的凭证
			cred, err = config.GetCredential(credentialAlias)
			if err != nil {
				return fmt.Errorf("error getting credential: %w", err)
			}
		} else if !isDirectConnect && conn.DefaultCredential != "" {
			// 使用连接配置中的默认凭证（仅当使用别名时）
			cred, err = config.GetCredential(conn.DefaultCredential)
			if err != nil {
				return fmt.Errorf("error getting default credential: %w", err)
			}
		}

		// 确定显示的用户名
		username := conn.User
		if cred != nil && cred.Username != "" {
			username = cred.Username
		} else if connectUser != "" {
			username = connectUser
		} else {
			username = "root"
		}

		// 验证是否有可用的用户名
		if username == "" {
			return fmt.Errorf("no username provided, please specify with --user or use a credential with username")
		}

		fmt.Printf("Connecting to %s (%s@%s:%d)...\n",
			target, username, conn.Host, conn.Port)

		return ssh.ConnectWithCredential(conn, cred)
	},
}

// isIPorHostname 检查给定的字符串是否像是IP地址或主机名
func isIPorHostname(s string) bool {
	// 检查是否是有效的IP地址
	if net.ParseIP(s) != nil {
		return true
	}

	// 检查是否可能是主机名（包含点号但不是纯数字）
	if strings.Contains(s, ".") {
		// 尝试看是否可以解析为数字（如192.168.1.1）
		parts := strings.Split(s, ".")
		allNumbers := true

		for _, part := range parts {
			if _, err := strconv.Atoi(part); err != nil {
				allNumbers = false
				break
			}
		}

		// 如果不全是数字，则可能是主机名
		if !allNumbers {
			return true
		}
	}

	// 最后检查是否是已存在的配置别名
	cfg, err := config.LoadConfig()
	if err == nil {
		if _, exists := cfg.Connections[s]; exists {
			return false // 是已配置的别名
		}
	}

	return true // 当作IP/主机名处理
}

func init() {
	connectCmd.Flags().StringVarP(&credentialAlias, "credential", "c", "",
		"Use specific credential alias for connection")
	connectCmd.Flags().IntVarP(&connectPort, "port", "p", 0,
		"Port to use when connecting directly to IP/hostname (default: 22)")
	connectCmd.Flags().StringVarP(&connectUser, "user", "u", "",
		"Username to use when connecting directly to IP/hostname")
	rootCmd.AddCommand(connectCmd)
}
