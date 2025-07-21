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

	// 单行代理配置，格式："http://user:pass@host:port" 或 "socks5://host:port"
	Proxy            string `yaml:"proxy,omitempty"`
	
	// 默认使用的凭证别名
	DefaultCredential string `yaml:"default_credential,omitempty"`
}

// Credential represents a credential for SSH authentication
type Credential struct {
	Type        string `yaml:"type"` // "key" 或 "password"
	Username    string `yaml:"username,omitempty"`
	Password    string `yaml:"password,omitempty"`
	KeyPath     string `yaml:"key_path,omitempty"`
	KeyPassword string `yaml:"key_password,omitempty"` // 私钥密码
}

// Config represents the structure of the config file
type Config struct {
	Connections map[string]Connection `yaml:"connections"`
	Credentials map[string]Credential `yaml:"credentials"`
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

// / LoadConfig loads the configuration from the config file
func LoadConfig() (*Config, error) {
	configPath := GetConfigPath()

	// 检查文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 创建默认配置
		return &Config{
			Connections: make(map[string]Connection),
			Credentials: make(map[string]Credential),
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

	// 初始化maps
	if config.Connections == nil {
		config.Connections = make(map[string]Connection)
	}
	if config.Credentials == nil {
		config.Credentials = make(map[string]Credential)
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

// GetCredential gets a credential by alias
func GetCredential(alias string) (*Credential, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	cred, exists := config.Credentials[alias]
	if !exists {
		return nil, fmt.Errorf("credential alias '%s' not found", alias)
	}

	return &cred, nil
}
