package ssh

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/justseemore/sshm/pkg/config"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

// Connect connects to an SSH server using the given configuration
func Connect(conn *config.Connection) error {
	// Create SSH client config
	clientConfig := &ssh.ClientConfig{
		User:            conn.User,
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Note: In production, use ssh.FixedHostKey or ssh.VerifyHostKeyDNS
	}

	// Add password auth if provided
	if conn.Password != "" {
		clientConfig.Auth = append(clientConfig.Auth, ssh.Password(conn.Password))
	}

	// Add identity file auth if provided
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

	// Set timeout if specified
	if conn.Timeout != "" {
		timeout, err := time.ParseDuration(conn.Timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout value: %w", err)
		}
		clientConfig.Timeout = timeout
	}

	// Connect to SSH server
	addr := fmt.Sprintf("%s:%d", conn.Host, conn.Port)
	client, err := ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		return fmt.Errorf("unable to connect to SSH server: %w", err)
	}
	defer client.Close()

	// Create SSH session
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("unable to create SSH session: %w", err)
	}
	defer session.Close()

	// Set up terminal
	fd := int(os.Stdin.Fd())
	oldState, err := terminal.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("unable to set terminal to raw mode: %w", err)
	}
	defer terminal.Restore(fd, oldState)

	// Set up IO
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	// Get terminal dimensions
	width, height, err := terminal.GetSize(fd)
	if err != nil {
		return fmt.Errorf("unable to get terminal size: %w", err)
	}

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	// Request pseudo-terminal
	if err := session.RequestPty("xterm-256color", height, width, modes); err != nil {
		return fmt.Errorf("request for pseudo terminal failed: %w", err)
	}

	// Handle window resize
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

	// Start shell
	if err := session.Shell(); err != nil {
		return fmt.Errorf("failed to start shell: %w", err)
	}

	// Wait for session to finish
	if err := session.Wait(); err != nil {
		if e, ok := err.(*ssh.ExitError); ok {
			return fmt.Errorf("command exited with code %d", e.ExitStatus())
		}
		return fmt.Errorf("session ended with error: %w", err)
	}

	return nil
}
