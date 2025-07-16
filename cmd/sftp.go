package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/justseemore/sshm/pkg/config"
	"github.com/justseemore/sshm/pkg/sftp"
	"github.com/spf13/cobra"
)

var (
	// SFTP命令标志
	sftpRemotePath string
	sftpLocalPath  string
	sftpRecursive  bool
)

// sftpCmd 表示SFTP命令
var sftpCmd = &cobra.Command{
	Use:   "sftp",
	Short: "SFTP operations (upload/download files)",
	Long:  `Perform SFTP operations to upload and download files with progress tracking.`,
}

// uploadCmd 表示上传文件命令
var uploadCmd = &cobra.Command{
	Use:   "upload [alias|host] [local_path] [remote_path]",
	Short: "Upload files to remote server",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		// 确定本地路径
		localPath := sftpLocalPath
		if localPath == "" && len(args) > 1 {
			localPath = args[1]
		}
		if localPath == "" {
			return fmt.Errorf("local path is required")
		}

		// 确定远程路径
		remotePath := sftpRemotePath
		if remotePath == "" && len(args) > 2 {
			remotePath = args[2]
		}
		if remotePath == "" {
			// 如果未指定远程路径，使用本地文件名
			remotePath = filepath.Base(localPath)
		}

		// 确定连接和凭证
		conn, cred, err := resolveConnectionAndCredential(target)
		if err != nil {
			return err
		}

		// 创建SFTP客户端
		client, err := sftp.NewSftpClient(conn, cred)
		if err != nil {
			return fmt.Errorf("failed to create SFTP client: %w", err)
		}
		defer client.Close()

		// 检查本地路径是否是目录
		localInfo, err := os.Stat(localPath)
		if err != nil {
			return fmt.Errorf("failed to access local path: %w", err)
		}

		if localInfo.IsDir() {
			if !sftpRecursive {
				return fmt.Errorf("local path is a directory, use --recursive to upload directories")
			}

			// 递归上传目录
			return uploadDirectory(client, localPath, remotePath)
		} else {
			// 上传单个文件
			fmt.Printf("Uploading %s to %s:%s\n", localPath, conn.Host, remotePath)
			return client.UploadFile(localPath, remotePath)
		}
	},
}

// downloadCmd 表示下载文件命令
var downloadCmd = &cobra.Command{
	Use:   "download [alias|host] [remote_path] [local_path]",
	Short: "Download files from remote server",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		// 确定远程路径
		remotePath := sftpRemotePath
		if remotePath == "" && len(args) > 1 {
			remotePath = args[1]
		}
		if remotePath == "" {
			return fmt.Errorf("remote path is required")
		}

		// 确定本地路径
		localPath := sftpLocalPath
		if localPath == "" && len(args) > 2 {
			localPath = args[2]
		}
		if localPath == "" {
			// 如果未指定本地路径，使用远程文件名
			localPath = filepath.Base(remotePath)
		}

		// 确定连接和凭证
		conn, cred, err := resolveConnectionAndCredential(target)
		if err != nil {
			return err
		}

		// 创建SFTP客户端
		client, err := sftp.NewSftpClient(conn, cred)
		if err != nil {
			return fmt.Errorf("failed to create SFTP client: %w", err)
		}
		defer client.Close()

		// 检查远程路径是否是目录
		remoteInfo, err := client.GetSftpClient().Stat(remotePath)
		if err != nil {
			return fmt.Errorf("failed to access remote path: %w", err)
		}

		if remoteInfo.IsDir() {
			if !sftpRecursive {
				return fmt.Errorf("remote path is a directory, use --recursive to download directories")
			}

			// 递归下载目录
			return downloadDirectory(client, remotePath, localPath)
		} else {
			// 下载单个文件
			fmt.Printf("Downloading %s from %s to %s\n", remotePath, conn.Host, localPath)
			return client.DownloadFile(remotePath, localPath)
		}
	},
}

// lsCmd 表示列出远程文件命令
var lsCmd = &cobra.Command{
	Use:   "ls [alias|host] [remote_path]",
	Short: "List files on remote server",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		// 确定远程路径
		remotePath := "."
		if len(args) > 1 {
			remotePath = args[1]
		}

		// 确定连接和凭证
		conn, cred, err := resolveConnectionAndCredential(target)
		if err != nil {
			return err
		}

		// 创建SFTP客户端
		client, err := sftp.NewSftpClient(conn, cred)
		if err != nil {
			return fmt.Errorf("failed to create SFTP client: %w", err)
		}
		defer client.Close()

		// 列出远程目录内容
		files, err := client.ListFiles(remotePath)
		if err != nil {
			return fmt.Errorf("failed to list files: %w", err)
		}

		// 打印文件列表
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "MODE\tSIZE\tMODIFIED\tNAME")

		for _, file := range files {
			mode := file.Mode().String()
			size := file.Size()
			modified := file.ModTime().Format("Jan 02 15:04")
			name := file.Name()

			fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", mode, size, modified, name)
		}

		return w.Flush()
	},
}

// 辅助函数：上传目录
func uploadDirectory(client *sftp.SftpClient, localPath, remotePath string) error {
	// 确保远程目录存在
	if err := client.MakeDir(remotePath); err != nil {
		return fmt.Errorf("failed to create remote directory: %w", err)
	}

	// 递归处理本地目录
	return filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(localPath, path)
		if err != nil {
			return err
		}

		// 跳过根目录
		if relPath == "." {
			return nil
		}

		// 构建远程路径
		remPath := filepath.Join(remotePath, relPath)

		// 处理文件或目录
		if info.IsDir() {
			// 创建远程目录
			return client.MakeDir(remPath)
		} else {
			// 上传文件
			fmt.Printf("Uploading %s to %s\n", path, remPath)
			return client.UploadFile(path, remPath)
		}
	})
}

// 辅助函数：下载目录
func downloadDirectory(client *sftp.SftpClient, remotePath, localPath string) error {
	// 确保本地目录存在
	if err := os.MkdirAll(localPath, 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}

	// 列出远程目录中的文件
	files, err := client.ListFiles(remotePath)
	if err != nil {
		return fmt.Errorf("failed to list remote directory: %w", err)
	}

	for _, file := range files {
		remPath := filepath.Join(remotePath, file.Name())
		locPath := filepath.Join(localPath, file.Name())

		if file.IsDir() {
			// 为子目录创建本地目录
			if err := os.MkdirAll(locPath, 0755); err != nil {
				return fmt.Errorf("failed to create local directory: %w", err)
			}

			// 递归处理子目录
			if err := downloadDirectory(client, remPath, locPath); err != nil {
				return err
			}
		} else {
			// 下载文件
			fmt.Printf("Downloading %s to %s\n", remPath, locPath)
			if err := client.DownloadFile(remPath, locPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// 辅助函数：解析连接和凭证
func resolveConnectionAndCredential(target string) (*config.Connection, *config.Credential, error) {
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
			return nil, nil, fmt.Errorf("error getting connection: %w", err)
		}
	}

	// 确定使用哪个凭证
	var cred *config.Credential
	if credentialAlias != "" {
		// 使用命令行指定的凭证
		cred, err = config.GetCredential(credentialAlias)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting credential: %w", err)
		}
	} else if !isDirectConnect && conn.DefaultCredential != "" {
		// 使用连接配置中的默认凭证（仅当使用别名时）
		cred, err = config.GetCredential(conn.DefaultCredential)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting default credential: %w", err)
		}
	}

	// 确定显示的用户名
	username := conn.User
	if cred != nil && cred.Username != "" {
		username = cred.Username
	} else if connectUser != "" {
		username = connectUser
	}

	// 验证是否有可用的用户名
	if username == "" {
		return nil, nil, fmt.Errorf("no username provided, please specify with --user or use a credential with username")
	}

	// 更新连接用户名
	conn.User = username

	return conn, cred, nil
}

func init() {
	rootCmd.AddCommand(sftpCmd)
	sftpCmd.AddCommand(uploadCmd)
	sftpCmd.AddCommand(downloadCmd)
	sftpCmd.AddCommand(lsCmd)

	// 将凭证、用户名和端口标志添加到SFTP命令
	for _, cmd := range []*cobra.Command{uploadCmd, downloadCmd, lsCmd} {
		cmd.Flags().StringVarP(&credentialAlias, "credential", "c", "",
			"Use specific credential alias for connection")
		cmd.Flags().IntVarP(&connectPort, "port", "p", 0,
			"Port to use when connecting directly to IP/hostname (default: 22)")
		cmd.Flags().StringVarP(&connectUser, "user", "u", "",
			"Username to use when connecting directly to IP/hostname")
	}

	// 添加SFTP特定标志
	for _, cmd := range []*cobra.Command{uploadCmd, downloadCmd} {
		cmd.Flags().StringVarP(&sftpLocalPath, "local", "l", "",
			"Local file or directory path")
		cmd.Flags().StringVarP(&sftpRemotePath, "remote", "r", "",
			"Remote file or directory path")
		cmd.Flags().BoolVarP(&sftpRecursive, "recursive", "R", false,
			"Recursively upload/download directories")
	}
}
