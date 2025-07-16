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

	// 新增代理相关变量
	proxyType     string
	proxyHost     string
	proxyPort     int
	proxyUser     string
	proxyPassword string

	defaultCredential string
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

		// 展开身份文件路径
		if identityFile != "" {
			if identityFile[0] == '~' {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("error getting home directory: %w", err)
				}
				identityFile = filepath.Join(homeDir, identityFile[1:])
			}
		}

		// 在RunE函数中添加验证代码
		if defaultCredential != "" {
			if _, exists := cfg.Credentials[defaultCredential]; !exists {
				return fmt.Errorf("default credential '%s' not found", defaultCredential)
			}
		}
		// 创建新连接配置
		cfg.Connections[alias] = config.Connection{
			Host:         host,
			Port:         port,
			User:         user,
			Password:     password,
			IdentityFile: identityFile,
			Timeout:      timeout,

			// 添加代理配置
			ProxyType:     proxyType,
			ProxyHost:     proxyHost,
			ProxyPort:     proxyPort,
			ProxyUser:     proxyUser,
			ProxyPassword: proxyPassword,

			DefaultCredential: defaultCredential,
		}

		// 保存配置
		if err := config.SaveConfig(cfg); err != nil {
			return fmt.Errorf("error saving config: %w", err)
		}

		fmt.Printf("Connection '%s' added successfully.\n", alias)
		return nil
	},
}

func init() {
	// 现有选项
	addCmd.Flags().StringVarP(&host, "host", "H", "", "Host address (required)")
	addCmd.Flags().IntVarP(&port, "port", "p", 22, "Port number")
	addCmd.Flags().StringVarP(&user, "user", "u", "", "Username (required)")
	addCmd.Flags().StringVarP(&password, "password", "P", "", "Password (not recommended, use identity file instead)")
	addCmd.Flags().StringVarP(&identityFile, "identity-file", "i", "", "Identity file path")
	addCmd.Flags().StringVarP(&timeout, "timeout", "t", "10s", "Connection timeout")
	// 添加默认凭证选项
	addCmd.Flags().StringVar(&defaultCredential, "default-credential", "",
		"Default credential to use for this connection")

	// 添加代理相关选项
	addCmd.Flags().StringVar(&proxyType, "proxy-type", "", "Proxy type (http, socks5, none)")
	addCmd.Flags().StringVar(&proxyHost, "proxy-host", "", "Proxy host")
	addCmd.Flags().IntVar(&proxyPort, "proxy-port", 0, "Proxy port")
	addCmd.Flags().StringVar(&proxyUser, "proxy-user", "", "Proxy username")
	addCmd.Flags().StringVar(&proxyPassword, "proxy-password", "", "Proxy password")

	addCmd.MarkFlagRequired("host")
	addCmd.MarkFlagRequired("user")

	rootCmd.AddCommand(addCmd)
}
