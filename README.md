# LocalTUN

<p align="right">
    <b>English</b> | <a href="./README_zh.md">简体中文</a>
</p>

[![GitHub last commit](https://img.shields.io/github/last-commit/fishwww-ww/LocalTUN)](https://github.com/fishwww-ww/LocalTUN/commits/main/)
[![GitHub License](https://img.shields.io/github/license/fishwww-ww/LocalTUN)](https://github.com/fishwww-ww/LocalTUN/blob/main/LICENSE)
[![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/fishwww-ww/LocalTUN/total)](https://github.com/fishwww-ww/LocalTUN/releases)

`LocalTUN` is a CLI tool that forwards proxy traffic from a remote server to your local proxy port over an SSH reverse tunnel.

It is useful when your local machine already runs Clash, Mihomo, Surge, V2Ray, or another proxy client, and you want a cloud server to reuse that local proxy instead of maintaining a separate proxy stack on the server.

## Features

- **Interactive setup**: Generate the config file with `localtun init`.
- **Remote bootstrap**: Configure `sshd_config` and remote `~/.bashrc` with `localtun setup`.
- **SSH reverse tunneling**: Forward a remote port to your local proxy port.
- **Keepalive and reconnect**: Built-in keepalive checks with exponential backoff reconnect.
- **Foreground and daemon modes**: Run directly in the terminal or in the background.
- **Connectivity test**: Verify the full proxy path from the remote server.

## How It Works

```text
Remote server (:1080) -> SSH reverse tunnel -> Local machine (:7897) -> Local proxy client -> Internet
```

Typical flow:

1. Run `localtun start` on your local machine.
2. The program establishes an SSH connection to the remote server.
3. A remote port, `1080` by default, is exposed on the server side.
4. Traffic sent to that remote port is forwarded to your local proxy port, `7897` by default.
5. Your local proxy handles the outbound request and the response travels back through the SSH tunnel.

## Prerequisites

- **A running local proxy**: An HTTP or mixed proxy port is available on your machine, default `7897`.
- **SSH key access**: Your machine can log in to the remote server using a private key.
- **Remote privileges**: `localtun setup` usually requires permission to edit `/etc/ssh/sshd_config` and restart SSH.
- **Go is optional**: Only needed if you want to build from source.

## Installation

### Homebrew

```bash
brew tap fishwww-ww/tap
brew install fishwww-ww/tap/localtun
```

### Build from source

```bash
go build -o localtun .
```

To make it available globally:

```bash
sudo mv localtun /usr/local/bin/
```

### Run without installing

```bash
go run . --help
```

## Quick Start

### 1. Initialize configuration

```bash
localtun init
```

You will be prompted for:

- One or more server names
- Server IP or hostname
- SSH username
- SSH port
- SSH private key path
- Remote proxy port
- Local proxy port

The default config path is:

```text
~/.localtun/config.yaml
```

### 2. Configure the remote server

For first-time setup on a target server:

```bash
localtun setup
```

This command connects to the server over SSH and, with confirmation prompts, can:

- back up and modify `/etc/ssh/sshd_config`
- enable `AllowTcpForwarding yes`
- enable `GatewayPorts yes`
- enable `PermitTunnel yes`
- optionally restart `sshd` or `ssh`
- back up and update remote `~/.bashrc`
- add proxy environment variables and helper functions: `proxy_on`, `proxy_off`, `proxy_test`

Some managed or container-based providers do not allow restarting SSH from inside the instance. If the restart step fails, you can continue the `.bashrc` setup and then test the tunnel directly.

After setup, reload the shell on the remote server:

```bash
source ~/.bashrc
```

### 3. Start the tunnel

Foreground mode:

```bash
localtun start
```

By default, this starts tunnels for all configured servers. To start only one server:

```bash
localtun start --server west
```

Daemon mode:

```bash
localtun start -d
```

Runtime files:

- `~/.localtun/run/<server>.pid` prevents duplicate tunnel processes per server.
- `~/.localtun/logs/<server>.log` is used in daemon mode.

### 4. Check status

```bash
localtun status
```

This shows:

- whether the tunnel is running
- process PID
- server address and user
- tunnel mapping
- keepalive settings
- log file path

### 5. Test connectivity

```bash
localtun test
```

The command tests outbound access from the remote server through the tunnel against:

- `https://www.baidu.com`
- `https://www.google.com`

This helps verify:

- the tunnel is up
- the local proxy is reachable
- the remote proxy environment works as expected

### 6. Stop the tunnel

```bash
localtun stop
```

## Configuration

The default config file is `~/.localtun/config.yaml`:

```yaml
servers:
  west:
    host: 1.2.3.4
    port: 22
    user: root
    key_path: ~/.ssh/id_rsa
    remote_port: 1080
    local_port: 7897
  east:
    host: example.com
    port: 22
    user: ubuntu
    key_path: ~/.ssh/id_ed25519
    remote_port: 1080
    local_port: 7897

keepalive:
  interval: 30
  max_count: 3
```

Field reference:

| Field | Description |
|------|------|
| `servers.<name>.host` | Remote server IP or hostname |
| `servers.<name>.port` | SSH port, default `22` |
| `servers.<name>.user` | SSH login user, default `root` |
| `servers.<name>.key_path` | SSH private key path, supports `~/` |
| `servers.<name>.remote_port` | Proxy port exposed on the remote server, default `1080` |
| `servers.<name>.local_port` | Local proxy port, default `7897` |
| `keepalive.interval` | Keepalive interval in seconds, default `30` |
| `keepalive.max_count` | Max keepalive failures before reconnect, default `3` |

## Commands

### `localtun init`

Interactively generate a multi-server config file. If the target file already exists, the command asks before overwriting it.

### `localtun server`

Manage server profiles:

- `localtun server list`: list configured servers
- `localtun server add [name]`: add or replace a server profile
- `localtun server remove [name]`: remove a server profile

### `localtun setup`

Configure selected remote servers with confirmation prompts before applying changes.

It handles:

- SSH forwarding settings in `sshd_config`
- proxy environment variables in `~/.bashrc`
- helper functions `proxy_on`, `proxy_off`, and `proxy_test`

### `localtun start`

Start the SSH reverse tunnel and forward the remote port to your local proxy port.

Default behavior:

- runs in the foreground
- starts all configured servers unless `--server` is provided
- exits gracefully on `Ctrl+C`
- reconnects automatically when the connection drops

Flags:

- `-d`, `--daemon`: run in the background
- `-s`, `--server`: only process the named server; can be passed multiple times

### `localtun status`

Show tunnel status, PID, config summary, and log path for each selected server.

### `localtun stop`

Stop selected background tunnel processes and remove PID files.

### `localtun test`

Connect to the remote server over SSH and run proxy tests with `curl --proxy`.

## Global Flag

All commands support:

| Flag | Description |
|------|------|
| `-c`, `--config` | Custom config path, default `~/.localtun/config.yaml` |

Example:

```bash
localtun --config /path/to/config.yaml start -d
localtun start --server west --server east
```

## What `setup` Changes on the Remote Server

### `sshd_config`

The command ensures these options are set to `yes`:

```text
AllowTcpForwarding yes
GatewayPorts yes
PermitTunnel yes
```

### `~/.bashrc`

It injects a `LocalTUN`-managed proxy block that includes:

- `http_proxy`
- `https_proxy`
- `HTTP_PROXY`
- `HTTPS_PROXY`
- `proxy_on`
- `proxy_off`
- `proxy_test`

Run `proxy_on` in the remote shell when you want to enable the proxy environment.

## Common Usage

### Start with the default config

```bash
localtun start
```

### Start in background and inspect status

```bash
localtun start -d
localtun status
```

### Use a custom config file

```bash
localtun -c ./config.yaml start -d
```

### Test the proxy manually on the remote shell

```bash
proxy_test
curl --proxy http://127.0.0.1:1080 -I -s https://www.google.com
```

## Files

| Path | Description |
|------|------|
| `~/.localtun/config.yaml` | Main config file |
| `~/.localtun/run/<server>.pid` | PID file used to prevent duplicate tunnel processes per server |
| `~/.localtun/logs/<server>.log` | Runtime log file used in daemon mode |

## Troubleshooting

### 1. Config file not found

Run:

```bash
localtun init
```

### 2. SSH connection fails

Check:

- server address and port
- SSH username
- private key path
- whether the key is authorized on the server

### 3. Tunnel setup fails

Common causes:

- remote port `1080` is already in use
- SSH port forwarding is not enabled on the server
- SSH service was not restarted after config changes
- firewall or security group rules block access

Try:

```bash
localtun setup
```

### 4. Remote proxy is unavailable

If `localtun test` fails on external sites, the issue is usually one of:

- local proxy client is not running
- wrong local proxy port
- local proxy does not support the forwarding mode
- SSH tunnel is down or has not been established yet

Suggested checks:

```bash
localtun status
localtun test
```

### 5. No output in daemon mode

Inspect the log:

```bash
cat ~/.localtun/logs/west.log
```

You will usually see whether:

- SSH connected successfully
- keepalive is failing repeatedly
- the local proxy port cannot be reached
- the program is reconnecting

## Notes

- The current implementation uses SSH private key authentication and does not support interactive password login.
- The remote environment is driven by HTTP proxy environment variables, so a local HTTP or mixed proxy port is recommended.
- SSH host key verification is permissive for a smoother first-run experience, so assess the security implications before using this against sensitive servers.
- `localtun setup` modifies remote system files. In production environments, review backup and rollback procedures first.

## License

This project is licensed under the MIT License. See [`LICENSE`](./LICENSE).
