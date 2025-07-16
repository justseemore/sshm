package ssh

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/justseemore/sshm/pkg/config"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/net/proxy"
)

// 保留原有的Connect函数
func Connect(conn *config.Connection) error {
	return ConnectWithCredential(conn, nil)
}

// Connect connects to an SSH server using the given configuration
func ConnectWithCredential(conn *config.Connection, cred *config.Credential) error {
	// 创建SSH客户端配置
	clientConfig := &ssh.ClientConfig{
		User:            conn.User, // 默认使用连接配置中的用户名
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 注意：生产环境应使用更安全的方法
	}

	// 使用凭证中的认证信息（如果提供）
	if cred != nil {
		// 使用凭证中的用户名（如果有）
		if cred.Username != "" {
			clientConfig.User = cred.Username
		}

		// 根据凭证类型添加认证方法
		if cred.Type == "key" {
			// 添加私钥认证
			expandedPath := os.ExpandEnv(cred.KeyPath)
			fmt.Println(expandedPath)
			key, err := os.ReadFile(expandedPath)
			if err != nil {
				return fmt.Errorf("unable to read private key: %w", err)
			}

			var signer ssh.Signer
			if cred.KeyPassword != "" {
				signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(cred.KeyPassword))
			} else {
				signer, err = ssh.ParsePrivateKey(key)
			}

			if err != nil {
				return fmt.Errorf("unable to parse private key: %w", err)
			}

			clientConfig.Auth = append(clientConfig.Auth, ssh.PublicKeys(signer))
		} else if cred.Type == "password" {
			// 添加密码认证
			clientConfig.Auth = append(clientConfig.Auth, ssh.Password(cred.Password))
		}
	} else {
		// 使用连接配置中的认证信息
		if conn.Password != "" {
			clientConfig.Auth = append(clientConfig.Auth, ssh.Password(conn.Password))
		}

		if conn.IdentityFile != "" {
			expandedPath := os.ExpandEnv(conn.IdentityFile)
			key, err := os.ReadFile(expandedPath)
			if err != nil {
				return fmt.Errorf("unable to read private key: %w", err)
			}

			signer, err := ssh.ParsePrivateKey(key)
			if err != nil {
				return fmt.Errorf("unable to parse private key: %w", err)
			}

			clientConfig.Auth = append(clientConfig.Auth, ssh.PublicKeys(signer))
		}
	}

	// 设置超时
	if conn.Timeout != "" {
		timeout, err := time.ParseDuration(conn.Timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout value: %w", err)
		}
		clientConfig.Timeout = timeout
	} else {
		clientConfig.Timeout = 10 * time.Second // 默认超时
	}

	var client *ssh.Client
	var err error
	addr := fmt.Sprintf("%s:%d", conn.Host, conn.Port)

	// 使用代理或直接连接
	if conn.ProxyType != "" && conn.ProxyType != "none" {
		switch conn.ProxyType {
		case "http":
			// HTTP代理连接
			proxyURL := &url.URL{
				Scheme: "http",
				Host:   fmt.Sprintf("%s:%d", conn.ProxyHost, conn.ProxyPort),
			}

			if conn.ProxyUser != "" {
				proxyURL.User = url.UserPassword(conn.ProxyUser, conn.ProxyPassword)
			}

			httpClient := &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyURL(proxyURL),
					DialContext: (&net.Dialer{
						Timeout:   clientConfig.Timeout,
						KeepAlive: 30 * time.Second,
					}).DialContext,
				},
			}

			// 使用HTTP代理拨号
			dialer := httpClient.Transport.(*http.Transport).DialContext
			netConn, err := dialer(context.Background(), "tcp", addr)
			if err != nil {
				return fmt.Errorf("unable to connect through HTTP proxy: %w", err)
			}

			// 使用建立的连接创建SSH客户端
			conn, chans, reqs, err := ssh.NewClientConn(netConn, addr, clientConfig)
			if err != nil {
				netConn.Close()
				return fmt.Errorf("unable to create SSH client connection: %w", err)
			}
			client = ssh.NewClient(conn, chans, reqs)

		case "socks5":
			// SOCKS5代理连接
			proxyAddr := fmt.Sprintf("%s:%d", conn.ProxyHost, conn.ProxyPort)
			var auth *proxy.Auth

			if conn.ProxyUser != "" {
				auth = &proxy.Auth{
					User:     conn.ProxyUser,
					Password: conn.ProxyPassword,
				}
			}

			dialer, err := proxy.SOCKS5("tcp", proxyAddr, auth, proxy.Direct)
			if err != nil {
				return fmt.Errorf("unable to create SOCKS5 proxy dialer: %w", err)
			}

			netConn, err := dialer.Dial("tcp", addr)
			if err != nil {
				return fmt.Errorf("unable to connect through SOCKS5 proxy: %w", err)
			}

			// 使用建立的连接创建SSH客户端
			conn, chans, reqs, err := ssh.NewClientConn(netConn, addr, clientConfig)
			if err != nil {
				netConn.Close()
				return fmt.Errorf("unable to create SSH client connection: %w", err)
			}
			client = ssh.NewClient(conn, chans, reqs)

		default:
			return fmt.Errorf("unsupported proxy type: %s", conn.ProxyType)
		}
	} else {
		// 直接连接（不使用代理）
		client, err = ssh.Dial("tcp", addr, clientConfig)
		if err != nil {
			return fmt.Errorf("unable to connect to SSH server: %w", err)
		}
	}
	defer client.Close()

	// 创建SSH会话
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("unable to create SSH session: %w", err)
	}
	defer session.Close()

	// 设置终端
	fd := int(os.Stdin.Fd())
	oldState, err := terminal.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("unable to set terminal to raw mode: %w", err)
	}
	defer terminal.Restore(fd, oldState)

	// 设置IO
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	// 获取终端尺寸
	width, height, err := terminal.GetSize(fd)
	if err != nil {
		return fmt.Errorf("unable to get terminal size: %w", err)
	}

	// 请求伪终端
	if err := session.RequestPty("xterm-256color", height, width, ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}); err != nil {
		return fmt.Errorf("request for pseudo terminal failed: %w", err)
	}

	// 处理窗口大小变化
	sigwinchCh := make(chan os.Signal, 1)
	signal.Notify(sigwinchCh, syscall.SIGWINCH)
	go func() {
		for range sigwinchCh {
			width, height, err := terminal.GetSize(fd)
			if err != nil {
				continue
			}
			session.WindowChange(height, width)
		}
	}()
	defer func() {
		signal.Stop(sigwinchCh)
		close(sigwinchCh)
	}()

	// 启动shell
	if err := session.Shell(); err != nil {
		return fmt.Errorf("failed to start shell: %w", err)
	}

	// 等待会话结束
	if err := session.Wait(); err != nil {
		if e, ok := err.(*ssh.ExitError); ok {
			return fmt.Errorf("command exited with code %d", e.ExitStatus())
		}
		return fmt.Errorf("session ended with error: %w", err)
	}

	return nil
}
