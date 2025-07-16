package sftp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/justseemore/sshm/pkg/config"
	"github.com/pkg/sftp"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/crypto/ssh"
)

// SftpClient 表示SFTP客户端
type SftpClient struct {
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

// NewSftpClient 创建新的SFTP客户端
func NewSftpClient(conn *config.Connection, cred *config.Credential) (*SftpClient, error) {
	// 创建SSH客户端配置
	clientConfig := &ssh.ClientConfig{
		User:            conn.User,
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 使用凭证配置身份验证
	if cred != nil {
		if cred.Username != "" {
			clientConfig.User = cred.Username
		}

		if cred.Type == "key" {
			expandedPath := os.ExpandEnv(cred.KeyPath)
			key, err := os.ReadFile(expandedPath)
			if err != nil {
				return nil, fmt.Errorf("unable to read private key: %w", err)
			}

			var signer ssh.Signer
			if cred.KeyPassword != "" {
				signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(cred.KeyPassword))
			} else {
				signer, err = ssh.ParsePrivateKey(key)
			}

			if err != nil {
				return nil, fmt.Errorf("unable to parse private key: %w", err)
			}

			clientConfig.Auth = append(clientConfig.Auth, ssh.PublicKeys(signer))
		} else if cred.Type == "password" {
			clientConfig.Auth = append(clientConfig.Auth, ssh.Password(cred.Password))
		}
	} else {
		// 使用连接配置
		if conn.Password != "" {
			clientConfig.Auth = append(clientConfig.Auth, ssh.Password(conn.Password))
		}

		if conn.IdentityFile != "" {
			expandedPath := os.ExpandEnv(conn.IdentityFile)
			key, err := os.ReadFile(expandedPath)
			if err != nil {
				return nil, fmt.Errorf("unable to read private key: %w", err)
			}

			signer, err := ssh.ParsePrivateKey(key)
			if err != nil {
				return nil, fmt.Errorf("unable to parse private key: %w", err)
			}

			clientConfig.Auth = append(clientConfig.Auth, ssh.PublicKeys(signer))
		}
	}

	// 设置超时
	if conn.Timeout != "" {
		timeout, err := time.ParseDuration(conn.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout value: %w", err)
		}
		clientConfig.Timeout = timeout
	} else {
		clientConfig.Timeout = 10 * time.Second
	}

	// 连接到SSH服务器
	addr := fmt.Sprintf("%s:%d", conn.Host, conn.Port)
	sshClient, err := ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to SSH server: %w", err)
	}

	// 创建SFTP客户端
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		return nil, fmt.Errorf("unable to create SFTP client: %w", err)
	}

	return &SftpClient{
		sshClient:  sshClient,
		sftpClient: sftpClient,
	}, nil
}

// Close 关闭SFTP和SSH连接
func (c *SftpClient) Close() error {
	if c.sftpClient != nil {
		c.sftpClient.Close()
	}
	if c.sshClient != nil {
		c.sshClient.Close()
	}
	return nil
}

// UploadFile 上传文件并显示进度条
func (c *SftpClient) UploadFile(localPath, remotePath string) error {
	// 打开本地文件
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer localFile.Close()

	// 获取文件信息用于进度条
	fileInfo, err := localFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get local file info: %w", err)
	}
	fileSize := fileInfo.Size()

	// 创建远程文件
	remoteFile, err := c.sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file: %w", err)
	}
	defer remoteFile.Close()

	// 创建进度条
	bar := progressbar.NewOptions(
		int(fileSize),
		progressbar.OptionSetDescription(fmt.Sprintf("Uploading %s", filepath.Base(localPath))),
		progressbar.OptionSetWriter(os.Stdout),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(50),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() { fmt.Println() }),
	)

	// 复制文件并更新进度
	_, err = io.Copy(remoteFile, io.TeeReader(localFile, bar))
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	return nil
}

// DownloadFile 下载文件并显示进度条
func (c *SftpClient) DownloadFile(remotePath, localPath string) error {
	// 打开远程文件
	remoteFile, err := c.sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file: %w", err)
	}
	defer remoteFile.Close()

	// 获取文件信息用于进度条
	fileInfo, err := remoteFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get remote file info: %w", err)
	}
	fileSize := fileInfo.Size()

	// 创建本地文件
	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer localFile.Close()

	// 创建进度条
	bar := progressbar.NewOptions(
		int(fileSize),
		progressbar.OptionSetDescription(fmt.Sprintf("Downloading %s", filepath.Base(remotePath))),
		progressbar.OptionSetWriter(os.Stdout),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(50),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() { fmt.Println() }),
	)

	// 复制文件并更新进度
	_, err = io.Copy(localFile, io.TeeReader(remoteFile, bar))
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	return nil
}

// ListFiles 列出远程目录中的文件
func (c *SftpClient) ListFiles(remotePath string) ([]os.FileInfo, error) {
	return c.sftpClient.ReadDir(remotePath)
}

// MakeDir 在远程服务器上创建目录
func (c *SftpClient) MakeDir(remotePath string) error {
	return c.sftpClient.MkdirAll(remotePath)
}

// GetSftpClient 返回底层sftp.Client
func (c *SftpClient) GetSftpClient() *sftp.Client {
	return c.sftpClient
}
