package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/justseemore/sshm/pkg/config"
	"github.com/spf13/cobra"
)

var (
	// 凭证类型和相关字段
	credType      string
	credUsername  string
	credPassword  string
	credKeyPath   string
	credKeyPasswd string
)

// credCmd 表示管理凭证的命令
var credCmd = &cobra.Command{
	Use:   "cred",
	Short: "Manage SSH credentials",
	Long:  `Manage SSH credentials including SSH keys and username/password pairs.`,
}

// credAddCmd 添加新的凭证
var credAddCmd = &cobra.Command{
	Use:   "add [alias]",
	Short: "Add a new SSH credential",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias := args[0]

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("error loading config: %w", err)
		}

		if _, exists := cfg.Credentials[alias]; exists {
			return fmt.Errorf("credential with alias '%s' already exists", alias)
		}

		// 验证凭证类型
		if credType != "key" && credType != "password" {
			return fmt.Errorf("invalid credential type: must be 'key' or 'password'")
		}

		// 验证必要参数
		if credType == "key" {
			if credKeyPath == "" {
				return fmt.Errorf("key path is required for key type credential")
			}

			// 展开路径
			if credKeyPath[0] == '~' {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("error getting home directory: %w", err)
				}
				credKeyPath = filepath.Join(homeDir, credKeyPath[1:])
			}

			// 检查密钥文件是否存在
			if _, err := os.Stat(credKeyPath); os.IsNotExist(err) {
				return fmt.Errorf("key file does not exist: %s", credKeyPath)
			}
		} else if credType == "password" {
			if credUsername == "" || credPassword == "" {
				return fmt.Errorf("username and password are required for password type credential")
			}
		}

		// 创建新凭证
		cfg.Credentials[alias] = config.Credential{
			Type:        credType,
			Username:    credUsername,
			Password:    credPassword,
			KeyPath:     credKeyPath,
			KeyPassword: credKeyPasswd,
		}

		// 保存配置
		if err := config.SaveConfig(cfg); err != nil {
			return fmt.Errorf("error saving config: %w", err)
		}

		fmt.Printf("Credential '%s' added successfully.\n", alias)
		return nil
	},
}

// credListCmd 列出所有凭证
var credListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all SSH credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("error loading config: %w", err)
		}

		if len(cfg.Credentials) == 0 {
			fmt.Println("No credentials configured. Use 'sshm cred add' to add a credential.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ALIAS\tTYPE\tUSERNAME\tKEY PATH")
		for alias, cred := range cfg.Credentials {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", alias, cred.Type, cred.Username, cred.KeyPath)
		}
		return w.Flush()
	},
}

// credDeleteCmd 删除凭证
var credDeleteCmd = &cobra.Command{
	Use:   "delete [alias]",
	Short: "Delete an SSH credential",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias := args[0]

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("error loading config: %w", err)
		}

		if _, exists := cfg.Credentials[alias]; !exists {
			return fmt.Errorf("credential with alias '%s' not found", alias)
		}

		delete(cfg.Credentials, alias)

		if err := config.SaveConfig(cfg); err != nil {
			return fmt.Errorf("error saving config: %w", err)
		}

		fmt.Printf("Credential '%s' deleted successfully.\n", alias)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(credCmd)
	credCmd.AddCommand(credAddCmd)
	credCmd.AddCommand(credListCmd)
	credCmd.AddCommand(credDeleteCmd)

	credAddCmd.Flags().StringVar(&credType, "type", "", "Credential type: 'key' or 'password' (required)")
	credAddCmd.Flags().StringVar(&credUsername, "username", "", "Username for the credential")
	credAddCmd.Flags().StringVar(&credPassword, "password", "", "Password for the credential or for the key")
	credAddCmd.Flags().StringVar(&credKeyPath, "key-path", "", "Path to the SSH key file")
	credAddCmd.Flags().StringVar(&credKeyPasswd, "key-password", "", "Password for the SSH key file")

	credAddCmd.MarkFlagRequired("type")
}
