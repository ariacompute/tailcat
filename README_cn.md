# aria-tailcat

用于在 ariacompute 宿主上部署
[`github.com/tailscale/tailcat`](https://github.com/tailscale/tailcat) 的启动器。

`tailcat` 是一个**不经过** Tailscale 控制面、`tsnet` 或 Funnel 的点对点
WireGuard/DERP 隧道工具——它会打印一个 `tc...` 地址交给对端，对端直接通过
DERP 连入。

`aria-tailcat` 是一个薄包装程序：它解析配置（YAML / flags / env），拼装
`tailcat serve` 的参数，并以 systemd 服务方式运行随包捆绑的 `tailcat` 二进制。
它**不**重写隧道逻辑。

## 安装（Debian）

```bash
sudo dpkg -i aria-tailcat_*.deb
sudo apt-get install -f   # 缺少时拉取 ca-certificates

# 修改服务形态后启用：
sudoedit /etc/aria-tailcat/config.yaml
sudo systemctl enable --now aria-tailcat
sudo journalctl -u aria-tailcat -f   # 观察打印出的 tc... 地址
```

## 配置

`/etc/aria-tailcat/config.yaml`（包安装时写入模板）：

```yaml
serve:
  # A) 通用监听（打印 tc 地址）：
  # mode: listen
  # ports: ["8080"]

  # B) 转发内网端口到隧道：
  # mode: forward
  # forward: "8080"

  # C) SSH 服务器（需授权密钥）：
  # mode: ssh
  # ssh:
  #   authorized_keys: ["alice@github", "/etc/aria-tailcat/id_ed25519.pub"]

  # C) Exec 服务器（每连接执行命令）：
  # mode: exec
  # exec:
  #   command: /usr/bin/fortune

  mode: listen
  ports: ["8080"]
  # key: default          # tailcat 密钥名/路径；'new' = 临时密钥
  # derpmap_url: ""       # 自定义 DERP map URL
  # allow: []             # nodekey 白名单；空 = 允许所有
  # psk: true             # 地址中包含 WireGuard PSK
```

环境变量覆盖写在 `/etc/default/aria-tailcat`（`ARIA_TAILCAT_*`），主要是
`ARIA_TAILCAT_CONFIG`、`ARIA_TAILCAT_BIN`、`ARIA_TAILCAT_KEY`、
`ARIA_TAILCAT_DERPMAP_URL`。

## 本地运行

```bash
go build -o build/aria-tailcat ./cmd/aria-tailcat
go build -o build/tailcat github.com/tailscale/tailcat/cmd/tailcat

./build/aria-tailcat run --bin ./build/tailcat --config config/config.yaml
```

## 构建 .deb

```bash
./scripts/build-deb.sh          # 需要 dpkg-buildpackage 与 golang
./scripts/test-cat.sh           # smoke 测试（需能访问 DERP 网络）
./scripts/check-emudhra.sh      # 离线 eMudhra/emSign 证书审计
```

## 信任库（eMudhra/emSign 证书泄露，2026-10）

静态 Go 二进制（Go 1.27.3）在 `crypto/x509` 中硬编码封禁了泄露的 emSign 中间证书，
可保护 tailcat 连接 DERP 的 TLS。请保持宿主 `ca-certificates` 包为最新，并运行
`scripts/check-emudhra.sh` 审计宿主 CA bundle。
