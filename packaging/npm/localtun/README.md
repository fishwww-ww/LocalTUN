# @fishwww-ww/localtun

`localtun` 是一个 CLI 工具，用于创建自带外网能力的 SSH Session。

它会通过 SSH 反向隧道把远端临时代理端口连接到本机代理，并且只给当前远端 shell 注入代理环境变量。退出 session 后，临时隧道会自动销毁。

安装：

```bash
npm install -g @fishwww-ww/localtun
```

验证：

```bash
localtun --help
```

使用：

```bash
localtun connect root@gpu01
localtun connect root@gpu01 --identity ~/.ssh/id_ed25519
localtun connect --detach root@gpu01
```

更多说明见项目主页：

- <https://github.com/fishwww-ww/LocalTUN>
