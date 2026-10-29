# aria-tailcat

[English](README.md) | [中文](README_cn.md)

Deployment launcher for [`github.com/tailscale/tailcat`](https://github.com/tailscale/tailcat)
on ariacompute hosts.

`tailcat` is a point-to-point WireGuard/DERP tunnel that runs **without** the
Tailscale control plane, `tsnet`, or Funnel — it prints a `tc...` address you can
hand to a peer, who connects directly over DERP.

## Install (Debian)

```bash
sudo dpkg -i aria-tailcat_*.deb
sudo apt-get install -f  # pull ca-certificates if missing

# Edit the serve shape, then enable:
sudoedit /etc/aria-tailcat/config.yaml
sudo systemctl enable --now aria-tailcat
sudo journalctl -u aria-tailcat -f  # look for the printed tc... address
```

## Configure

`/etc/aria-tailcat/config.yaml` (template installed by the package):

```yaml
serve:
  # A) Generic listener (print tc address):
  # mode: listen
  # ports: ["8080"]

  # B) Forward an internal port through the tunnel:
  # mode: forward
  # forward: "8080"

  # C) SSH server (needs authorized keys):
  # mode: ssh
  # ssh:
  #   authorized_keys: ["alice@github", "/etc/aria-tailcat/id_ed25519.pub"]

  # C) Exec server (run a command per connection):
  # mode: exec
  # exec:
  #   command: /usr/bin/fortune

  mode: listen
  ports: ["8080"]
  # key: default     # tailcat key name/path; 'new' = ephemeral
  # derpmap_url: ""  # custom DERP map URL
  # allow: []        # nodekey allowlist; empty = allow all
  # psk: true        # include WireGuard PSK in address
```

Env overrides live in `/etc/default/aria-tailcat` (`ARIA_TAILCAT_*`); in particular
`ARIA_TAILCAT_CONFIG`, `ARIA_TAILCAT_KEY`, and
`ARIA_TAILCAT_DERPMAP_URL`.

## Run locally

```bash
go build -o build/aria-tailcat ./cmd/aria-tailcat
./build/aria-tailcat run --config config/config.yaml
```

## Build the .deb

```bash
./scripts/build-deb.sh      # needs dpkg-buildpackage + golang
./scripts/test-cat.sh       # smoke test (needs DERP network access)
./scripts/check-emudhra.sh  # offline eMudhra/emSign CA audit
```

## License

MIT.

## Engineering Conventions

This repository follows the Harness Engineering philosophy:

- [`AGENTS.md`](AGENTS.md): Agent engineering context entry and directory index
- [`requirements.md`](requirements.md): Requirements spec (feature boundaries/exceptions/acceptance criteria, human-review-gated)
- [`task.md`](task.md): Implementation task checklist
