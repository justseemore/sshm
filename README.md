# SSHM - SSH Manager

SSHM 是一个功能强大的 SSH 连接管理工具，使用 Go 语言开发，基于 Cobra 命令行框架。它可以帮助您管理 SSH 连接、凭证和执行文件传输操作，提高运维效率。

## 功能特点

- **连接管理**：存储和管理多个 SSH 连接配置
- **凭证管理**：安全存储 SSH 密钥和密码凭证
- **代理支持**：通过 HTTP 或 SOCKS5 代理连接到服务器
- **文件传输**：内置 SFTP 功能，支持 rz/sz 风格的文件上传下载
- **进度显示**：文件传输时显示详细进度条
- **直接连接**：支持直接使用 IP 地址连接，无需预先配置
- **多平台支持**：兼容 Linux、macOS 和 Windows

## 安装方法

### 从源码安装

```bash
# 克隆仓库
git clone https://github.com/yourusername/sshm.git
cd sshm

# 安装依赖并编译
go mod tidy
go build -o sshm

# 移动到系统路径
sudo mv sshm /usr/local/bin/
```
### 使用预编译二进制文件

从 [Releases](https://github.com/yourusername/sshm/releases) 页面下载适合您系统的二进制文件，然后添加到系统路径中。

## 基本使用

### 管理 SSH 连接
```bash
# 添加新连接
sshm add my-server --host 192.168.1.100 --user admin --identity-file ~/.ssh/id_rsa

# 添加带代理的连接
sshm add proxy-server --host example.com --port 2222 --user john \
  --proxy-type socks5 --proxy-host 127.0.0.1 --proxy-port 1080

# 列出所有连接
sshm list

# 删除连接
sshm delete my-server

# 连接到服务器
sshm connect my-server
```
### 凭证管理
```bash
# 添加 SSH 密钥凭证
sshm cred add prod-key --type key --username admin --key-path ~/.ssh/prod_rsa

# 添加密码凭证
sshm cred add dev-user --type password --username developer --password "secret123"

# 列出所有凭证
sshm cred list

# 删除凭证
sshm cred delete dev-user
```
### 直接 IP 连接
```bash
# 直接连接到 IP，使用已存储的凭证
sshm connect 192.168.1.100 --credential prod-key

# 直接连接并指定用户和端口
sshm connect example.com --user admin --port 2222 --credential my-key
```
### 文件传输
```bash
# 上传文件（rz - Receive to Zmodem）
sshm sftp rz my-server ./local-file.txt /remote/path/file.txt

# 下载文件（sz - Send from Zmodem）
sshm sftp sz my-server /remote/path/file.txt ./local-file.txt

# 递归上传目录
sshm sftp rz my-server -l ./local-folder -r /remote/path -R

# 递归下载目录
sshm sftp sz my-server -r /remote/folder -l ./local-path -R

# 列出远程目录内容
sshm sftp ls my-server /remote/path
```
## 配置文件

SSHM 使用 YAML 格式的配置文件存储连接和凭证信息，默认位于 `~/.config/sshm/ssh.yaml`。
### 配置文件示例
```yaml
connections:
  prod-server:
    host: 192.168.1.100
    port: 22
    timeout: 10s
    default_credential: prod-key
  
  dev-server:
    host: dev.example.com
    port: 2222
    proxy_type: socks5
    proxy_host: 127.0.0.1
    proxy_port: 1080
    default_credential: dev-account

credentials:
  prod-key:
    type: key
    username: admin
    key_path: ~/.ssh/prod_rsa
  
  dev-account:
    type: password
    username: developer
    password: dev-password
```
## 代理支持

SSHM 支持以下代理类型：

- **HTTP**: 通过 HTTP 代理连接
- **SOCKS5**: 通过 SOCKS5 代理连接

可以为每个连接单独配置代理，也可以在直接连接时指定代理。

## 使用技巧

### 创建快捷命令

将以下别名添加到 `.bashrc` 或 `.zshrc` 文件中：
```bash
# 快速连接
alias s="sshm connect"

# 快速文件传输
alias rz="sshm sftp rz"
alias sz="sshm sftp sz"
```
### 安全建议

- 避免在配置文件中存储明文密码
- 优先使用 SSH 密钥认证
- 定期轮换凭证
- 对配置文件设置适当的权限：`chmod 600 ~/.config/sshm/ssh.yaml`

## 常见问题

**Q: 如何在使用代理的同时指定特定凭证？**

A: 可以同时使用 `--credential` 和 `--proxy-*` 参数，例如：
```bash
sshm connect example.com --credential my-key --proxy-type http --proxy-host proxy.example.com --proxy-port 8080
```
**Q: 如何避免每次都输入代理密码？**

A: 使用 `add` 命令创建带有代理配置的连接：
```bash
sshm add my-server --host example.com --proxy-type socks5 --proxy-host localhost --proxy-port 1080
```
## 依赖项

- github.com/spf13/cobra
- golang.org/x/crypto/ssh
- golang.org/x/crypto/ssh/terminal
- gopkg.in/yaml.v2
- github.com/pkg/sftp
- github.com/schollz/progressbar/v3
- golang.org/x/net/proxy

## 许可证

本项目采用 MIT 许可证。详情请参阅 [LICENSE](https://chat.geemii.com/LICENSE) 文件。

## 贡献

欢迎提交 Pull Request 或创建 Issue 来帮助改进这个项目。在提交代码前，请确保测试通过并遵循项目的代码风格。

该工具旨在简化 SSH 连接和文件传输操作，提高开发和运维工作效率。如有任何问题或建议，请通过 GitHub Issues 反馈。
