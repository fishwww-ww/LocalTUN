<h1 align="center">LocalTUN Next</h1>

<p align="center">
    <b>English</b> · <a href="./README_zh.md">简体中文</a>
</p>

<p align="center">
    <img src="https://img.shields.io/github/go-mod/go-version/fishwww-ww/LocalTUN?style=flat-square" alt="Go version">
    <img src="https://img.shields.io/github/license/fishwww-ww/LocalTUN?style=flat-square" alt="License">
    <img src="https://img.shields.io/github/actions/workflow/status/fishwww-ww/LocalTUN/release.yml?branch=main&style=flat-square" alt="Release workflow status">
    <img src="https://img.shields.io/github/v/release/fishwww-ww/LocalTUN?color=red&style=flat-square" alt="Latest release">
    <img src="https://img.shields.io/github/downloads/fishwww-ww/LocalTUN/total?style=flat-square" alt="Total downloads">
</p>

<p align="center">
    <b>SSH in. Download models. Done.</b>
</p>

`LocalTUN` creates an Internet-enabled SSH session for short-lived cloud and GPU servers.

Instead of configuring mirrors, editing shell profiles, or installing a proxy stack on the
remote server, run one command from the machine where your proxy already works:

```bash
localtun connect root@gpu01
```

LocalTUN opens a temporary SSH reverse tunnel to your local proxy, injects proxy
environment variables into the current remote shell only, and tears everything down when
the session exits.

## What It Does

```text
Remote shell env
  HTTP_PROXY=http://127.0.0.1:<temporary-port>
  HTTPS_PROXY=http://127.0.0.1:<temporary-port>
  ALL_PROXY=http://127.0.0.1:<temporary-port>
        |
        v
Remote 127.0.0.1:<temporary-port>
        |
        v
SSH reverse tunnel
        |
        v
Local proxy, for example 127.0.0.1:7897
```

LocalTUN does not modify remote `.bashrc`, `.zshrc`, Docker, Conda, system proxy settings,
or SSH config.

## Quick Start

Start your local proxy client first, then connect:

```bash
localtun connect root@gpu01
```

Common options:

```bash
localtun connect ubuntu@gpu01:2222
localtun connect gpu01 --identity ~/.ssh/id_ed25519
localtun connect root@gpu01 --local-proxy 7897
localtun connect root@gpu01 --remote-port 46327
localtun connect root@gpu01 --shell /bin/bash
```

Inside the remote shell, tools that respect proxy environment variables can immediately
use the Internet:

```bash
pip install transformers
git clone https://github.com/huggingface/transformers
huggingface-cli download bert-base-uncased
```

Exit the shell normally:

```bash
exit
```

The tunnel closes with the SSH session.

## Local Proxy Detection

When `--local-proxy` is not set, LocalTUN scans these local ports:

```text
7890, 7897, 1080, 20170
```

Use `--local-proxy host:port` or `--local-proxy port` to choose explicitly.

## Detached Tunnels

Use detached mode when a remote task should keep downloading after the terminal that
started LocalTUN exits:

```bash
localtun connect --detach root@gpu01
```

LocalTUN prints the temporary remote proxy URL and export commands. Detached session
metadata is stored under:

```text
~/.localtun-next/sessions/
```

List detached sessions:

```bash
localtun sessions
```

Stop one:

```bash
localtun disconnect <session-id>
```

## Requirements

- A local HTTP, SOCKS, or mixed proxy port already running.
- SSH key access to the target server.
- Remote SSH server allows TCP forwarding. If forwarding is disabled, LocalTUN reports
  that `AllowTcpForwarding` needs to be enabled.

## Build

```bash
go build -o localtun .
```

Run from source:

```bash
go run . --help
```
