package ssh

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/justseemore/sshm/pkg/config"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

// 保留原有的Connect函数
func Connect(conn *config.Connection) error {
	return ConnectWithCredential(conn, nil)
}

// Connect connects to an SSH server using the given configuration
func ConnectWithCredential(conn *config.Connection, cred *config.Credential) error {
	// 从连接池获取或创建SSH客户端
	pool := GetConnectionPool()
	client, err := pool.GetClient(conn, cred)
	if err != nil {
		return fmt.Errorf("unable to establish SSH connection: %w", err)
	}

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
