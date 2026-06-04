# LocalTUN

<p align="right">
    <a href="./README.md">English</a> | <b>简体中文</b>
</p>

[![GitHub last commit](https://img.shields.io/github/last-commit/fishwww-ww/LocalTUN)](https://github.com/fishwww-ww/LocalTUN/commits/main/)
[![GitHub License](https://img.shields.io/github/license/fishwww-ww/LocalTUN)](https://github.com/fishwww-ww/LocalTUN/blob/main/LICENSE)
[![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/fishwww-ww/LocalTUN/total)](https://github.com/fishwww-ww/LocalTUN/releases)

`LocalTUN` 是一个基于 SSH 反向隧道的命令行工具，用于把云服务器的代理流量转发到本地代理端口。

适合这样的场景：你的本地机器已经运行 Clash、Mihomo、Surge、V2Ray 等代理软件，希望云服务器通过本地代理访问外网，而不是在服务器上单独维护一套代理配置。

## 功能特性

- **交互式初始化**：通过 `localtun init` 生成配置文件。
- **自动配置远端服务器**：通过 `localtun setup` 配置 `sshd_config` 和远端 `~/.bashrc`。
- **SSH 反向隧道转发**：将远端端口流量转发到本地代理端口。
- **保活与自动重连**：内置 keepalive 检测和指数退避重连机制。
- **前后台运行**：既支持前台运行，也支持守护进程模式。
- **连通性测试**：支持在远端直接验证完整代理链路。

## 工作原理

```text
云服务器 (:1080) -> SSH 反向隧道 -> 本机 (:7897) -> 本地代理软件 -> 外网
```

典型流程如下：

1. 在本机运行 `localtun start`。
2. 程序建立到远端服务器的 SSH 连接。
3. 在远端暴露一个代理端口，默认是 `1080`。
4. 远端发往该端口的流量会被转发到本地代理端口，默认是 `7897`。
5. 本地代理处理请求后，再通过 SSH 隧道把响应返回给远端服务器。

## 前置条件

- **本地代理已启动**：本机有可用的 HTTP 或混合代理端口，默认 `7897`。
- **SSH 密钥可登录服务器**：当前机器可以使用私钥连接远端服务器。
- **远端具备修改权限**：`localtun setup` 通常需要修改 `/etc/ssh/sshd_config` 并重启 SSH 服务。
- **Go 环境可选**：仅在需要从源码编译时需要安装 Go。

## 安装

### Homebrew

```bash
brew tap fishwww-ww/tap
brew install fishwww-ww/tap/localtun
```

### 从源码编译

```bash
go build -o localtun .
```

如需全局使用，可将其放入 PATH：

```bash
sudo mv localtun /usr/local/bin/
```

### 不安装直接运行

```bash
go run . --help
```

## 快速开始

### 1. 初始化配置

```bash
localtun init
```

程序会交互式提示你输入：

- 服务器 IP 或域名
- SSH 用户名
- SSH 端口
- SSH 私钥路径
- 远程代理端口
- 本地代理端口

默认配置文件路径为：

```text
~/.localtun/config.yaml
```

### 2. 配置远程服务器

首次在目标服务器上启用时，执行：

```bash
localtun setup
```

该命令会通过 SSH 连接远端服务器，并在确认后执行以下操作：

- 备份并修改 `/etc/ssh/sshd_config`
- 启用 `AllowTcpForwarding yes`
- 启用 `GatewayPorts yes`
- 启用 `PermitTunnel yes`
- 可选重启 `sshd` 或 `ssh`
- 备份并更新远端 `~/.bashrc`
- 添加代理环境变量和 `proxy_on`、`proxy_off`、`proxy_test` 函数

某些托管或容器环境不允许在实例内部重启 SSH。如果重启步骤失败，可以继续配置 `.bashrc`，然后直接测试隧道是否可用。

配置完成后，请在远端重新加载 shell：

```bash
source ~/.bashrc
```

### 3. 启动隧道

前台运行：

```bash
localtun start
```

后台运行：

```bash
localtun start -d
```

运行时文件：

- `~/.localtun/localtun.pid`：前台和后台模式都会使用，用于防止重复启动隧道进程。
- `~/.localtun/localtun.log`：后台模式使用的日志文件。

### 4. 查看状态

```bash
localtun status
```

该命令会显示：

- 当前是否运行
- 进程 PID
- 服务器地址与用户
- 隧道映射关系
- keepalive 配置
- 日志文件位置

### 5. 测试连通性

```bash
localtun test
```

程序会在远端服务器上通过代理测试：

- `https://www.baidu.com`
- `https://www.google.com`

这样可以快速判断：

- 隧道是否已经建立
- 本地代理是否可达
- 远端代理环境是否按预期工作

### 6. 停止隧道

```bash
localtun stop
```

## 配置说明

默认配置文件 `~/.localtun/config.yaml` 示例：

```yaml
server:
  host: 1.2.3.4
  port: 22
  user: root
  key_path: ~/.ssh/id_rsa

tunnel:
  remote_port: 1080
  local_port: 7897

keepalive:
  interval: 30
  max_count: 3
```

字段说明：

| 配置项 | 说明 |
|------|------|
| `server.host` | 远程服务器 IP 或域名 |
| `server.port` | SSH 端口，默认 `22` |
| `server.user` | SSH 登录用户名，默认 `root` |
| `server.key_path` | SSH 私钥路径，支持 `~/` |
| `tunnel.remote_port` | 远端暴露的代理端口，默认 `1080` |
| `tunnel.local_port` | 本地代理端口，默认 `7897` |
| `keepalive.interval` | keepalive 间隔秒数，默认 `30` |
| `keepalive.max_count` | keepalive 最大失败次数，默认 `3` |

## 命令说明

### `localtun init`

交互式生成配置文件。如果目标文件已存在，会先询问是否覆盖。

### `localtun setup`

自动配置远程服务器，并在真正修改前逐步确认。

它会处理：

- `sshd_config` 中的转发相关设置
- `~/.bashrc` 中的代理环境变量
- `proxy_on`、`proxy_off`、`proxy_test` 辅助函数

### `localtun start`

启动 SSH 反向隧道，并将远端端口转发到本地代理端口。

默认行为：

- 前台运行
- 按 `Ctrl+C` 优雅退出
- 断线后自动重连

支持参数：

- `-d`, `--daemon`：后台运行

### `localtun status`

查看当前隧道状态，并输出 PID、配置摘要和日志路径。

### `localtun stop`

停止后台运行的隧道进程，并清理 PID 文件。

### `localtun test`

通过 SSH 登录远端服务器，执行基于 `curl --proxy` 的代理测试。

## 全局参数

所有命令都支持：

| 参数 | 说明 |
|------|------|
| `-c`, `--config` | 自定义配置文件路径，默认 `~/.localtun/config.yaml` |

示例：

```bash
localtun --config /path/to/config.yaml start -d
```

## `setup` 会修改什么

### `sshd_config`

会确保以下配置为 `yes`：

```text
AllowTcpForwarding yes
GatewayPorts yes
PermitTunnel yes
```

### `~/.bashrc`

会注入一段由 `LocalTUN` 管理的代理配置，包括：

- `http_proxy`
- `https_proxy`
- `HTTP_PROXY`
- `HTTPS_PROXY`
- `proxy_on`
- `proxy_off`
- `proxy_test`

需要启用代理环境时，在远端 shell 中运行 `proxy_on`。

## 常见用法

### 使用默认配置启动

```bash
localtun start
```

### 后台启动并查看状态

```bash
localtun start -d
localtun status
```

### 使用自定义配置文件

```bash
localtun -c ./config.yaml start -d
```

### 在远端手动测试代理

```bash
proxy_test
curl --proxy http://127.0.0.1:1080 -I -s https://www.google.com
```

## 文件说明

| 路径 | 说明 |
|------|------|
| `~/.localtun/config.yaml` | 主配置文件 |
| `~/.localtun/localtun.pid` | 用于防止重复启动隧道进程的 PID 文件 |
| `~/.localtun/localtun.log` | 后台模式运行日志 |

## 故障排查

### 1. 配置文件不存在

运行：

```bash
localtun init
```

### 2. SSH 无法连接

请检查：

- 服务器地址和端口是否正确
- SSH 用户名是否正确
- 私钥路径是否正确
- 私钥是否已在远端授权

### 3. 隧道建立失败

常见原因：

- 远端 `1080` 端口已被占用
- 服务器未开启 SSH 端口转发
- 修改配置后 SSH 服务未重启
- 防火墙或安全组阻止访问

可以重新执行：

```bash
localtun setup
```

### 4. 远端代理不可用

如果 `localtun test` 对外站点测试失败，常见原因包括：

- 本地代理软件未启动
- 本地代理端口填写错误
- 本地代理不支持当前转发方式
- SSH 隧道已断开或尚未建立

建议检查：

```bash
localtun status
localtun test
```

### 5. 后台模式无输出

查看日志：

```bash
cat ~/.localtun/localtun.log
```

通常可以从日志中看到：

- SSH 是否连接成功
- keepalive 是否持续失败
- 本地代理端口是否不可达
- 程序是否正在自动重连

## 注意事项

- 当前实现使用 SSH 私钥认证，不支持交互式密码登录。
- 远端访问依赖 HTTP 代理环境变量，因此建议本地提供 HTTP 或混合代理端口。
- 当前 SSH 主机密钥校验较宽松，优先保证首次使用顺畅；接入敏感服务器前请自行评估安全风险。
- `localtun setup` 会修改远端系统文件，生产环境建议先确认备份与回滚方案。

## 许可证

本项目基于 MIT License 开源，详见 [`LICENSE`](./LICENSE)。
