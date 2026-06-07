<h1 align="center">LocalTUN Next</h1>

<p align="center">
    <a href="./README.md">English</a> · <b>简体中文</b>
</p>

<p align="center">
    <img src="https://img.shields.io/github/go-mod/go-version/fishwww-ww/LocalTUN?style=flat-square" alt="Go version">
    <img src="https://img.shields.io/github/license/fishwww-ww/LocalTUN?style=flat-square" alt="License">
    <img src="https://img.shields.io/github/actions/workflow/status/fishwww-ww/LocalTUN/release.yml?branch=main&style=flat-square" alt="Release workflow status">
    <img src="https://img.shields.io/github/v/release/fishwww-ww/LocalTUN?color=red&style=flat-square" alt="Latest release">
    <img src="https://img.shields.io/github/downloads/fishwww-ww/LocalTUN/total?style=flat-square" alt="Total downloads">
</p>

<p align="center">
    <b>让新服务器立即可用。</b>
</p>

`LocalTUN` 用来创建一个自带外网能力的 SSH Session，适合短期使用的云服务器和 GPU 服务器。

你不需要在远端配置镜像源、修改 shell 配置、安装代理软件或维护订阅。只要本机代理已经可用，执行：

```bash
localtun connect root@gpu01
```

LocalTUN 会建立一个临时 SSH 反向隧道，只给当前远端 shell 注入代理环境变量，并在 session 退出时自动销毁隧道。

## 工作方式

```text
远端 shell 环境变量
  HTTP_PROXY=http://127.0.0.1:<临时端口>
  HTTPS_PROXY=http://127.0.0.1:<临时端口>
  ALL_PROXY=http://127.0.0.1:<临时端口>
        |
        v
远端 127.0.0.1:<临时端口>
        |
        v
SSH 反向隧道
        |
        v
本机代理，例如 127.0.0.1:7897
```

LocalTUN 不会修改远端 `.bashrc`、`.zshrc`、Docker、Conda、系统代理或 SSH 配置。

## 快速开始

先启动本机代理客户端，然后连接服务器：

```bash
localtun connect root@gpu01
```

常用参数：

```bash
localtun connect ubuntu@gpu01:2222
localtun connect gpu01 --identity ~/.ssh/id_ed25519
localtun connect root@gpu01 --local-proxy 7897
localtun connect root@gpu01 --remote-port 46327
localtun connect root@gpu01 --shell /bin/bash
```

进入远端 shell 后，支持代理环境变量的工具可以直接访问外网：

```bash
pip install transformers
git clone https://github.com/huggingface/transformers
huggingface-cli download bert-base-uncased
```

正常退出 shell：

```bash
exit
```

隧道会随 SSH Session 一起关闭。

## 本地代理探测

未传入 `--local-proxy` 时，LocalTUN 默认扫描：

```text
7890, 7897, 1080, 20170
```

也可以通过 `--local-proxy host:port` 或 `--local-proxy port` 显式指定。

## 后台模式

当远端任务需要在启动 LocalTUN 的终端退出后继续下载时，使用：

```bash
localtun connect --detach root@gpu01
```

LocalTUN 会输出临时远端代理 URL 和可复制的环境变量。后台 session 元数据保存在：

```text
~/.localtun-next/sessions/
```

查看后台 session：

```bash
localtun sessions
```

停止某个 session：

```bash
localtun disconnect <session-id>
```

## 前置条件

- 本机已有可用的 HTTP、SOCKS 或 mixed 代理端口。
- 当前机器可以通过 SSH 私钥登录目标服务器。
- 远端 SSH 服务允许 TCP forwarding。如果远端禁用转发，LocalTUN 会提示需要开启 `AllowTcpForwarding`。

## 构建

```bash
go build -o localtun .
```

从源码运行：

```bash
go run . --help
```
