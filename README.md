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

  mode: forward
  forward: "8080"
  full_address: true
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

## Connect to a node (client)

A peer connects to an aria-tailcat node purely through DERP using the `tcp...`
address the server prints (look for `aria-tailcat: serving tailcat address:
tcp...` in logs — the prefix is `tcp`, not `tc`). The launcher only runs the
**server**; use the upstream `tailcat` CLI as the client. The client script
downloads the upstream tailcat release binary (and falls back to a local
`go build` on macOS, where no release asset is published):

```bash
# ping the node over DERP (verifies reachability + allowlist)
TC_ADDR=tcp... ./scripts/test-client.sh

# optionally forward a local port to the server's forwarded port and probe it
TC_ADDR=tcp... PORT=8080 ./scripts/test-client.sh

# expose the forward on all interfaces (default is loopback 127.0.0.1)
BIND=0.0.0.0 TC_ADDR=tcp... PORT=8080 ./scripts/test-client.sh
```

The script passes `BIND` through to `tailcat forward --bind` (default loopback),
so the forwarded listener is only reachable locally unless you set `BIND` and open
the port in your firewall.

Other client subcommands (the server must serve the matching mode):

```bash
tailcat forward 8080 tcp...    # forward local :8080 to the server's :8080
tailcat socks tcp...           # SOCKS5 proxy through the tunnel
tailcat ssh tcp...             # SSH (server in ssh mode)
echo hi | tailcat tcp... 8080  # raw send to server :8080
```

The client also needs DERP map access (default
`https://tailcat.dev/derpmap.json`); set `TAILCAT_DERPMAP_URL` if the server
uses a custom DERP map. With `full_address: true` in the server config the
`tcp...` address embeds the DERP info, so clients connect without a DERP map
fetch.

## Build the .deb

```bash
./scripts/build-deb.sh      # needs dpkg-buildpackage + golang
./scripts/test-cat.sh       # smoke test (needs DERP network access)
./scripts/test-client.sh    # client DERP connectivity test (needs a tc... address)
./scripts/check-emudhra.sh  # offline eMudhra/emSign CA audit
```

## License

MIT.

## Engineering Conventions

This repository follows the Harness Engineering philosophy:

- [`AGENTS.md`](AGENTS.md): Agent engineering context entry and directory index
- [`requirements.md`](requirements.md): Requirements spec (feature boundaries/exceptions/acceptance criteria, human-review-gated)
- [`task.md`](task.md): Implementation task checklist
