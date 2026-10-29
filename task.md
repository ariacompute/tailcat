# Task list — aria-tailcat

Implement a thin Go launcher (module `ariacompute/tailcat`) that builds the pinned
`github.com/tailscale/tailcat` binary from the module graph and runs it in `serve`
mode, wrapping it as a systemd service. Reference pattern: `aria-tailscale`
(cobra CLI + debian/systemd + config precedence + emSign trust-store handling).

## Steps

- [x] Scaffold module: `go.mod` (Go 1.27.3, `github.com/tailscale/tailcat` v0.7.0,
      cobra, yaml) and `cmd/aria-tailcat/main.go` (cobra root + `run`, signal
      shutdown), `tools.go` to pin the tailcat module without compiling it into the
      launcher.
- [x] `internal/cat/config.go`: `Config`/`ServeConfig`, YAML load, defaults, precedence
      merge (defaults → YAML → flags → env), `validate` (ports required for listen,
      command required for exec).
- [x] `internal/cat/serve.go`: `BuildServer` constructs a `tailcat.Server` in-process
      (A/B/C serve shapes) from `ServeConfig`; `Run`/`runServe` starts it and
      forwards context cancellation to `DrainTCP`/`Close`.
- [x] `internal/cat/server.go`: `Options`, `RegisterFlags`/`FlagsFromCmd`, `Run`,
      precedence (built-in → YAML → CLI flags → env).
- [x] `internal/cat/` tests: `config_test.go` (LoadConfig/applyConfig/BuildServeArgs/
      validate), `server_test.go` (FlagsFromCmd defaults/precedence/CLI override).
- [x] Deployment: `debian/` (control, rules, install, changelog, service, default,
      source/format), `debian/aria-tailcat-service` launcher, `config/config.yaml`,
      `scripts/build-deb.sh`, `scripts/test-cat.sh`.
- [x] `.github/workflows/release.yml`: linux amd64/arm64, windows x64, macOS universal,
      Debian package on `aria-tailcat/vX.Y.Z` tags.
- [x] Docs: `README.md` (EN), `README_cn.md` (CN), `AGENTS.md`, `requirements.md`,
      `task.md`, `LICENSE`.

## Acceptance criteria

- `go build ./...` and `go test ./...` pass.
- `./scripts/test-cat.sh` passes: `aria-tailcat run` starts the tailcat server in-process
  and prints a `tc...` address (requires DERP network access).
- `./scripts/build-deb.sh` produces a `.deb` that installs `aria-tailcat`, the systemd
  unit, the launcher, and the
  default/config templates.
- Installed service joins via tailcat (no control plane) and prints a `tc...` address;
  `aria-tailcat run` with no config errors out (nothing to serve).

## eMudhra / emSign 证书私钥泄露响应（2026-10）

背景：eMudhra(emSign) 8 张中间 CA 私钥极弱（P-384 私钥标量 <256），可被反推伪造任意域名证书；
Go 等自带信任库需升级。tailcat 是 Go 静态二进制（经 `tailscale.com` 客户端 + `tailcat`
模块），以 Debian 打包（`.deb`）部署到宿主，无 Dockerfile。

### 1. Go 二进制（决定性）
- `go.mod`：`go 1.27.3`，由 `crypto/x509` 硬编码封禁 8 张 emSign 中间证书。tailcat 连接
  DERP 的出向 TLS 受该封禁保护，与宿主 `ca-certificates` 版本无关。

### 2. 宿主部署信任库
- `debian/control`：`Depends` 显式加入 `ca-certificates`，保证安装宿主具备系统信任库（Go 二进制
  出向 TLS 所需）。宿主仍须 `apt-get install ca-certificates` 保持最新（事件后发布）。

### 3. 检测与验证
- 复用 `scripts/check-emudhra.sh`：离线按 emSign 根（信任锚）指纹扫描宿主 CA bundle，规避代理假阳性。
