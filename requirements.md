# Requirements — aria-tailcat

## Goal

Provide point-to-point access to internal Ubuntu-hosted ariacompute services
through [github.com/tailscale/tailcat](https://github.com/tailscale/tailcat), a
control-plane-free WireGuard/DERP tunnel CLI, using a thin Go launcher
(`aria-tailcat`) that wraps it as a long-running systemd service.

## Functional requirements

1. **tailcat serve ingress.** Run `tailcat` in `serve` mode as a systemd service.
   tailcat prints a `tc...` address the peer uses to connect directly over DERP;
   no Tailscale control plane, tsnet, or Funnel is involved. The launcher resolves
   configuration and execs the `tailcat` binary (compiled from the pinned
   `github.com/tailscale/tailcat` module, not a prebuilt upstream binary).
2. **Serve-shape routing.** Translate a single `serve.mode` setting into the
   correct `tailcat serve` argument vector:
   - A) `listen` — generic listener: `tailcat serve <ports>` (prints the address).
   - B) `forward` — expose an internal port: `tailcat serve <port>`.
   - C) `ssh` — `tailcat serve --ssh-authorized-keys=... ssh`; `exec` —
     `tailcat serve exec -- /cmd`.
   Raw service names (`all`/`exit-node`/`files`) pass through positionally.
3. **Dual-channel configuration.** Support YAML config file and CLI flags. CLI has
   higher priority than YAML, reusing the precedence-merge logic from aria-tailscale.
   Common options: `key`, `derpmap_url`, `allow`, `psk`, `full_address`, `verbose`,
   `json`.
4. **Transparent wrapper.** `aria-tailcat` must not reimplement tunnel logic or add
   authentication; it only maps config → `tailcat serve` flags. All tunnel/auth
   policy (allowed nodekeys, PSK, DERP region) is tailcat's.
5. **Module-based delivery (in-process).** `github.com/tailscale/tailcat` is a pinned
   Go module dependency (v0.7.0). `aria-tailcat` imports the `tailcat` package and runs
   the server in-process (constructs a `tailcat.Server` and calls `Start()`); no
   separate `tailcat` binary is built, bundled, or executed.
6. **Ubuntu packaging.** Ship the launcher binary, a systemd unit, an `/etc/default`
   template, a config template, and a `.deb` produced via `dpkg-buildpackage`.
7. **CLI via cobra.** The `aria-tailcat` binary uses cobra with a `run` subcommand,
   `--config`, `--serve-*` flags, and signal-based graceful shutdown.
8. **Pinned tailcat.** Build against the pinned `github.com/tailscale/tailcat` v0.7.0
   Go module; keep the Go toolchain aligned (1.27.3) for the emSign x509 ban.
9. **Release pipeline.** A GitHub Actions workflow builds linux amd64 / linux arm64 /
   windows x64 / macOS universal binaries and a Debian package on tagged releases
   (`aria-tailcat/vX.Y.Z`).
10. **Unit tests.** Cover config loading/precedence, launcher argv construction for
    A/B/C, and `FlagsFromCmd` precedence/edge cases. tailcat-dependent runtime paths
    are excluded (covered by `scripts/test-cat.sh` against a live DERP).

## Non-functional requirements

- Graceful shutdown on SIGINT/SIGTERM: the launcher forwards the signal to the
  `tailcat` child and waits for it to exit.
- Secrets (key path, PSK) are passed via config file or env, never on the command
  line.
- The wrapper must not add authentication or alter tailcat's tunnel behavior.

## Out of scope

- Managing tailcat keys, DERP servers, or nodekey allowlists (operator task).
- Authentication of internal services (delegated to those services).
- Service discovery or dynamic route registration at runtime.
