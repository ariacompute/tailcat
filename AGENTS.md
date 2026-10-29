# AGENTS.md — aria-tailcat

`aria-tailcat` is a Go service that deploys
[`github.com/tailscale/tailcat`](https://github.com/tailscale/tailcat) on
ariacompute Ubuntu hosts. `tailcat` is a point-to-point WireGuard/DERP tunnel CLI
that runs **without** the Tailscale control plane, `tsnet`, or Funnel.

Unlike `aria-tailscale` (a transparent Host-based Funnel reverse proxy),
`aria-tailcat` is a **thin launcher**: it resolves configuration from
YAML/flags/env, builds the `tailcat serve` argument vector, and runs the
bundled `tailcat` binary as a long-running systemd service.

## Repository layout

```
cmd/aria-tailcat/main.go     cobra root + `run` subcommand; signal-based graceful shutdown
internal/cat/
  config.go                  Config/ServeConfig types, YAML loading, precedence merge, validate
  launcher.go                BuildServeArgs (A/B/C -> tailcat serve argv) + exec + signal forwarding
  server.go                  Options, RegisterFlags/FlagsFromCmd/Run
  config_test.go             LoadConfig / applyConfig / BuildServeArgs / validate
  server_test.go             FlagsFromCmd defaults + precedence
config/config.yaml           example YAML config (A/B/C serve shapes, commented)
debian/                      debhelper packaging (control, rules, install, service, default,
                             aria-tailcat-service, postinst, postrm, changelog, source/)
scripts/
  build-deb.sh               runs dpkg-buildpackage -us -uc -b
  test-cat.sh                local smoke test (build + run, check tc address printed)
  check-emudhra.sh           offline host CA bundle scan for eMudhra/emSign roots
.github/workflows/release.yml  multi-platform release (linux amd64/arm64, windows x64, macos, deb)
```

## Architecture

- **Module dependency, not a prebuilt binary**: `github.com/tailscale/tailcat` is a
  pinned Go module dependency (see `go.mod` + `tools.go`). The Debian build compiles
  `github.com/tailscale/tailcat/cmd/tailcat` from that module and bundles it at
  `/usr/lib/aria-tailcat/tailcat`. At runtime `aria-tailcat` `exec`s it. This avoids
  fetching an opaque binary while still using tailcat's full, correct `serve`
  behavior (including ssh/exec).
- **Routing / serve shapes** (`Config.Serve.Mode`):
  - `listen` (A) — generic listener: `tailcat serve <ports>` prints the `tc...` address.
  - `forward` (B) — expose an internal ariacompute port: `tailcat serve <port>`.
  - `ssh` (C) — `tailcat serve --ssh-authorized-keys=... ssh`.
  - `exec` (C) — `tailcat serve exec -- /cmd`.
  - raw service names (`all`/`exit-node`/`files`) pass through positionally.
- **Transparent / no auth at the wrapper**: `aria-tailcat` only translates config into
  `tailcat serve` flags; all tunnel/auth policy (allowed nodekeys, PSK, DERP region) is
  tailcat's. Secrets (key path, PSK) are passed via config file or env, never on the
  command line.

## Conventions

- Module path: `ariacompute/tailcat`. Go 1.27.3+. Key deps: `tailscale.com`
  (transitively via `tailcat`), `github.com/spf13/cobra`, `gopkg.in/yaml.v3`.
- Never modify `prombot-ai/gateway` or `tailscale/tailcat` upstream; this repo only
  wraps deployment.
- Keep `aria-tailcat` thin: do not reimplement tunnel logic; add serve shapes by
  mapping config -> `tailcat serve` argv in `internal/cat/launcher.go`.
- Status dir `/var/lib/aria-tailcat`, config `/etc/aria-tailcat/config.yaml`,
  env overrides in `/etc/default/aria-tailcat` (`ARIA_TAILCAT_*`).

## Trust stores (eMudhra/emSign CA key compromise, 2026-10)

- **Go binary** (static, `CGO_ENABLED=0`, via `tailscale.com` / `tailcat`): `go.mod`
  uses Go 1.27.3, which hard-bans the 8 emSign intermediate certs in `crypto/x509`.
  tailcat's DERP TLS is protected by this ban even on an older host `ca-certificates`.
- **Host deployment trust store**: the `.deb` installs to Debian/Ubuntu hosts; its
  runtime trust store is the host `ca-certificates`. `debian/control` `Depends` on
  `ca-certificates`. Hosts must keep it updated (`apt-get install ca-certificates`).
- `scripts/check-emudhra.sh`: offline scan of the host CA bundle by eMudhra/emSign
  trust-anchor fingerprint (hosts usually have openssl/curl).

## Common commands

```bash
go build -o aria-tailcat ./cmd/aria-tailcat
go test ./...
go build -o build/tailcat github.com/tailscale/tailcat/cmd/tailcat
./scripts/test-cat.sh
./scripts/build-deb.sh
./scripts/check-emudhra.sh
```
