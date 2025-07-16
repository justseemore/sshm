package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// Connection represents an SSH connection configuration
type Connection struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	User         string `yaml:"user"`
	Password     string `yaml:"password,omitempty"`
	IdentityFile string `yaml:"identity_file,omitempty"`
	Timeout      string `yaml:"timeout,omitempty"`

	// 新增代理配置字段
	ProxyType     string `yaml:"proxy_type,omitempty"` // 支持 "http", "socks5", "none"
	ProxyHost     string `yaml:"proxy_host,omitempty"`
	ProxyPort     int    `yaml:"proxy_port,omitempty"`
	ProxyUser     string `yaml:"proxy_user,omitempty"`
	ProxyPassword string `yaml:"proxy_password,omitempty"`
}

// Config represents the structure of the config file
type Config struct {
	Connections map[string]Connection `yaml:"connections"`
}

// GetConfigPath returns the path to the config file
func GetConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}
	return filepath.Join(homeDir, ".config", "sshm", "ssh.yaml")
}

// LoadConfig loads the configuration from the config file
func LoadConfig() (*Config, error) {
	configPath := GetConfigPath()

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config
		return &Config{
			Connections: make(map[string]Connection),
		}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return &config, nil
}

// SaveConfig saves the configuration to the config file
func SaveConfig(config *Config) error {
	configPath := GetConfigPath()

	// Ensure directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error serializing config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}

// GetConnection gets a connection by alias
func GetConnection(alias string) (*Connection, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	conn, exists := config.Connections[alias]
	if !exists {
		return nil, fmt.Errorf("connection alias '%s' not found", alias)
	}

	return &conn, nil
}
