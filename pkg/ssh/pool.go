package ssh

import (
	"context"
	"fmt"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/justseemore/sshm/pkg/config"
	"golang.org/x/crypto/ssh"
)

// ConnectionPool 管理SSH连接池
type ConnectionPool struct {
	connections map[string]*ssh.Client
	mutex       sync.RWMutex
	// 连接最后使用时间，用于清理过期连接
	lastUsed map[string]time.Time
}

// 全局连接池实例
var (
	globalPool     *ConnectionPool
	globalPoolOnce sync.Once
)

// GetConnectionPool 返回全局连接池实例
func GetConnectionPool() *ConnectionPool {
	globalPoolOnce.Do(func() {
		globalPool = &ConnectionPool{
			connections: make(map[string]*ssh.Client),
			lastUsed:    make(map[string]time.Time),
		}
		// 启动定期清理过期连接的goroutine
		go globalPool.cleanupExpiredConnections()
	})
	return globalPool
}

// 生成连接的唯一键
func generateConnectionKey(conn *config.Connection, cred *config.Credential) string {
	user := conn.User
	if cred != nil && cred.Username != "" {
		user = cred.Username
	}
	return fmt.Sprintf("%s@%s:%d", user, conn.Host, conn.Port)
}

// GetClient 从连接池获取客户端，如果不存在则创建新的
func (p *ConnectionPool) GetClient(conn *config.Connection, cred *config.Credential) (*ssh.Client, error) {
	key := generateConnectionKey(conn, cred)

	// 先尝试从池中获取现有连接
	p.mutex.RLock()
	client, exists := p.connections[key]
	p.mutex.RUnlock()

	if exists {
		// 测试连接是否仍然有效
		_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
		if err == nil {
			// 更新最后使用时间
			p.mutex.Lock()
			p.lastUsed[key] = time.Now()
			p.mutex.Unlock()
			return client, nil
		}
		// 连接已失效，从池中移除
		p.mutex.Lock()
		delete(p.connections, key)
		delete(p.lastUsed, key)
		p.mutex.Unlock()
	}

	// 创建新的SSH连接
	client, err := createSSHClient(conn, cred)
	if err != nil {
		return nil, err
	}

	// 将新连接添加到池中
	p.mutex.Lock()
	p.connections[key] = client
	p.lastUsed[key] = time.Now()
	p.mutex.Unlock()

	// 启动心跳保持连接活跃
	go p.keepAlive(client, key)

	return client, nil
}

// 保持连接活跃的心跳
func (p *ConnectionPool) keepAlive(client *ssh.Client, key string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 添加一个检测客户端是否已关闭的通道
	closed := make(chan struct{})
	go func() {
		// 这个goroutine会在客户端关闭时退出
		_, _, _ = client.SendRequest("keepalive@openssh.com", true, nil)
		close(closed)
	}()

	for {
		select {
		case <-ticker.C:
			// 检查连接是否仍在池中
			p.mutex.RLock()
			_, exists := p.connections[key]
			p.mutex.RUnlock()
			if !exists {
				return
			}

			// 发送心跳
			_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				// 连接已断开，从池中移除
				p.mutex.Lock()
				delete(p.connections, key)
				delete(p.lastUsed, key)
				p.mutex.Unlock()
				return
			}

			// 更新最后使用时间
			p.mutex.Lock()
			p.lastUsed[key] = time.Now()
			p.mutex.Unlock()
		case <-closed:
			// 客户端已关闭，从池中移除
			p.mutex.Lock()
			delete(p.connections, key)
			delete(p.lastUsed, key)
			p.mutex.Unlock()
			return
		}
	}
}

// 清理过期连接
func (p *ConnectionPool) cleanupExpiredConnections() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		<-ticker.C
		now := time.Now()

		p.mutex.Lock()
		for key, lastUsed := range p.lastUsed {
			// 如果连接超过30分钟未使用，关闭并移除
			if now.Sub(lastUsed) > 30*time.Minute {
				if client, exists := p.connections[key]; exists {
					client.Close()
					delete(p.connections, key)
					delete(p.lastUsed, key)
				}
			}
		}
		p.mutex.Unlock()
	}
}

// 创建新的SSH客户端连接
func createSSHClient(conn *config.Connection, cred *config.Credential) (*ssh.Client, error) {
	// 这里复用现有的SSH客户端创建逻辑，但不包括交互式会话部分
	// 创建SSH客户端配置
	clientConfig := &ssh.ClientConfig{
		User:            conn.User,
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
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
		clientConfig.Timeout = 10 * time.Second // 默认超时
	}

	var client *ssh.Client
	var err error
	addr := fmt.Sprintf("%s:%d", conn.Host, conn.Port)

	// 使用代理或直接连接
	if conn.Proxy != "" {
		// 解析代理URL
		proxyURL, err := url.Parse(conn.Proxy)
		if err != nil {
			return nil, fmt.Errorf("unable to parse proxy URL: %w", err)
		}
		
		proxyType := proxyURL.Scheme
		proxyHost := proxyURL.Hostname()
		proxyPort, err := strconv.Atoi(proxyURL.Port())
		if err != nil {
			return nil, fmt.Errorf("invalid proxy port: %w", err)
		}
		
		proxyUser := ""
		proxyPassword := ""
		if proxyURL.User != nil {
			proxyUser = proxyURL.User.Username()
			proxyPassword, _ = proxyURL.User.Password()
		}
		
		switch proxyType {
		case "http":
			// HTTP代理连接
			httpProxyURL := &url.URL{
				Scheme: "http",
				Host:   fmt.Sprintf("%s:%d", proxyHost, proxyPort),
			}
			
			if proxyUser != "" {
				httpProxyURL.User = url.UserPassword(proxyUser, proxyPassword)
			}
			
			httpClient := &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyURL(httpProxyURL),
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
				return nil, fmt.Errorf("unable to connect through HTTP proxy: %w", err)
			}
			
			// 使用建立的连接创建SSH客户端
			conn, chans, reqs, err := ssh.NewClientConn(netConn, addr, clientConfig)
			if err != nil {
				netConn.Close()
				return nil, fmt.Errorf("unable to create SSH client connection: %w", err)
			}
			client = ssh.NewClient(conn, chans, reqs)
			
		case "socks5":
			// SOCKS5代理连接
			proxyAddr := fmt.Sprintf("%s:%d", proxyHost, proxyPort)
			var auth *proxy.Auth
			
			if proxyUser != "" {
				auth = &proxy.Auth{
					User:     proxyUser,
					Password: proxyPassword,
				}
			}
			
			dialer, err := proxy.SOCKS5("tcp", proxyAddr, auth, proxy.Direct)
			if err != nil {
				return nil, fmt.Errorf("unable to create SOCKS5 proxy dialer: %w", err)
			}
			
			netConn, err := dialer.Dial("tcp", addr)
			if err != nil {
				return nil, fmt.Errorf("unable to connect through SOCKS5 proxy: %w", err)
			}
			
			// 使用建立的连接创建SSH客户端
			conn, chans, reqs, err := ssh.NewClientConn(netConn, addr, clientConfig)
			if err != nil {
				netConn.Close()
				return nil, fmt.Errorf("unable to create SSH client connection: %w", err)
			}
			client = ssh.NewClient(conn, chans, reqs)
			
		default:
			return nil, fmt.Errorf("unsupported proxy type: %s", proxyType)
		}
	} else {
		// 直接连接（不使用代理）
		client, err = ssh.Dial("tcp", addr, clientConfig)
		if err != nil {
			return nil, fmt.Errorf("unable to connect to SSH server: %w", err)
		}
	}

	return client, nil
}
