package cat

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tailscale/tailcat"
	"tailscale.com/types/key"
	"tailscale.com/wgengine/filter"
)

// BuildServer constructs an in-process tailcat.Server from a ServeConfig. It does
// not start the server; call Start on the returned server (see Run). This is the
// "import the tailcat module" integration: no separate tailcat binary is exec'd.
func BuildServer(s ServeConfig) (*tailcat.Server, error) {
	if strings.TrimSpace(s.Mode) == "" {
		s.Mode = "listen"
	}
	if err := s.validate(); err != nil {
		return nil, err
	}

	priv, err := loadKey(s.Key)
	if err != nil {
		return nil, err
	}

	srv := &tailcat.Server{
		Key:        priv,
		Logf:       log.Printf,
		DERPMapURL: s.DerpMapURL,
	}
	if s.PSK != nil && !*s.PSK {
		srv.DisablePresharedKey = true
	}
	for _, ks := range s.Allow {
		if ks == "none" {
			srv.AddAllowedClient(key.NodePublic{})
			continue
		}
		var k key.NodePublic
		if err := k.UnmarshalText([]byte(ks)); err != nil {
			return nil, fmt.Errorf("invalid allow key %q: %w", ks, err)
		}
		srv.AddAllowedClient(k)
	}

	ports := s.Ports
	if s.Mode == "forward" && s.Forward != "" {
		ports = []string{s.Forward}
	}
	portSet, err := parsePorts(ports)
	if err != nil {
		return nil, err
	}

	var sshHandler, execHandler func(net.Conn)
	switch s.Mode {
	case "ssh":
		opts := tailcat.SSHOptions{Shell: true, AuthorizedKeys: resolveAuthorizedKeys(s.SSH.AuthorizedKeys)}
		sshHandler = srv.SSHConnHandler(opts)
	case "exec":
		cmd := strings.Fields(s.Exec.Command)
		if len(cmd) > 0 {
			execHandler = srv.ExecConnHandler(cmd)
		}
	}

	// Ports the packet filter admits. exec mode accepts any port (handled by
	// execHandler), so it leaves ServedTCPPorts empty.
	admit := make(map[uint16]bool, len(portSet))
	if s.Mode != "exec" {
		for p := range portSet {
			admit[p] = true
		}
		if sshHandler != nil {
			admit[22] = true
		}
	}
	var ranges []filter.PortRange
	for p := range admit {
		ranges = append(ranges, filter.PortRange{First: p, Last: p})
	}
	srv.ServedTCPPorts = ranges

	srv.OnTCP = func(port uint16) func(net.Conn) {
		if port == 22 && sshHandler != nil {
			return sshHandler
		}
		if execHandler != nil {
			return execHandler
		}
		if admit[port] {
			return tcpForwardTo(fmt.Sprintf("localhost:%d", port))
		}
		return nil
	}
	return srv, nil
}

// runServe builds the in-process tailcat server and serves until ctx is
// cancelled, then drains and closes the server gracefully.
func runServe(ctx context.Context, o Options) error {
	srv, err := BuildServer(o.Serve)
	if err != nil {
		return err
	}
	if err := srv.Start(); err != nil {
		return fmt.Errorf("start tailcat server: %w", err)
	}
	addr := srv.TailcatAddr()
	log.Printf("aria-tailcat: serving tailcat address: %s", addr)
	if o.Serve.JSON {
		fmt.Printf("{\"listenAddr\":%q}\n", string(addr))
	}

	<-ctx.Done()
	_ = srv.DrainTCP(ctx)
	_ = srv.Close()
	return nil
}

// loadKey resolves the tailcat node private key. The empty string or "default"
// loads (and persists) a key under STATE_DIRECTORY (set by systemd) or the user
// cache dir; "new" generates an ephemeral key; any other value is treated as a
// file path to a key in tailscale's NodePrivate text format.
func loadKey(spec string) (key.NodePrivate, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" || spec == "default" {
		path := defaultKeyPath()
		if b, err := os.ReadFile(path); err == nil {
			var k key.NodePrivate
			if err := k.UnmarshalText(b); err != nil {
				return k, fmt.Errorf("read key %s: %w", path, err)
			}
			return k, nil
		}
		k := key.NewNode()
		if b, err := k.MarshalText(); err == nil {
			_ = os.MkdirAll(filepath.Dir(path), 0700)
			_ = os.WriteFile(path, b, 0600)
		}
		return k, nil
	}
	if spec == "new" {
		return key.NewNode(), nil
	}
	b, err := os.ReadFile(spec)
	if err != nil {
		return key.NodePrivate{}, err
	}
	var k key.NodePrivate
	if err := k.UnmarshalText(b); err != nil {
		return k, fmt.Errorf("read key %s: %w", spec, err)
	}
	return k, nil
}

func defaultKeyPath() string {
	if d := os.Getenv("STATE_DIRECTORY"); d != "" {
		return filepath.Join(d, "tailcat.key")
	}
	if d, err := os.UserCacheDir(); err == nil {
		return filepath.Join(d, "aria-tailcat", "tailcat.key")
	}
	return "tailcat.key"
}

// resolveAuthorizedKeys turns each configured source into authorized_keys text.
// A source that is an existing file is read; otherwise it is treated as literal
// authorized_keys text. GitHub (name@github) lookup is not resolved locally.
func resolveAuthorizedKeys(sources []string) []string {
	out := make([]string, 0, len(sources))
	for _, src := range sources {
		if b, err := os.ReadFile(src); err == nil {
			out = append(out, string(b))
			continue
		}
		out = append(out, src)
	}
	return out
}

// parsePorts expands a list of port tokens (single ports and "lo-hi" ranges,
// comma-separated) into a set of uint16 ports.
func parsePorts(ports []string) (map[uint16]bool, error) {
	set := make(map[uint16]bool)
	for _, raw := range ports {
		for _, tok := range strings.Split(raw, ",") {
			tok = strings.TrimSpace(tok)
			if tok == "" {
				continue
			}
			if i := strings.IndexByte(tok, '-'); i >= 0 {
				lo, err1 := strconv.ParseUint(tok[:i], 10, 16)
				hi, err2 := strconv.ParseUint(tok[i+1:], 10, 16)
				if err1 != nil || err2 != nil || lo > hi {
					return nil, fmt.Errorf("invalid port range %q", tok)
				}
				for p := uint16(lo); p <= uint16(hi); p++ {
					set[p] = true
				}
				continue
			}
			p, err := strconv.ParseUint(tok, 10, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid port %q", tok)
			}
			set[uint16(p)] = true
		}
	}
	return set, nil
}

// tcpForwardTo returns an OnTCP handler that proxies the accepted connection to
// the given localhost address.
func tcpForwardTo(addr string) func(net.Conn) {
	return func(c net.Conn) {
		defer c.Close()
		up, err := net.Dial("tcp", addr)
		if err != nil {
			return
		}
		defer up.Close()
		done := make(chan struct{}, 2)
		go func() { io.Copy(up, c); done <- struct{}{} }()
		go func() { io.Copy(c, up); done <- struct{}{} }()
		<-done
	}
}
