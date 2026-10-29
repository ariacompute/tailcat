package cat

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ServeConfig describes how the imported github.com/tailscale/tailcat library
// should serve, in-process (no separate tailcat binary is executed).
//
// The three ariacompute serve shapes map onto tailcat like this:
//   - A) listen (generic listener): Mode=listen  + Ports=[...]         -> tailcat serve <ports>
//   - B) forward (expose internal port): Mode=forward + Forward=<port> -> tailcat serve <port>
//   - C) ssh / exec server: Mode=ssh | exec                             -> tailcat serve [--ssh-authorized-keys=...] ssh / tailcat serve exec -- <cmd>
type ServeConfig struct {
	Mode    string   `yaml:"mode"`    // "listen" | "forward" | "ssh" | "exec" (or a raw tailcat service name like "all"/"exit-node"/"files")
	Ports   []string `yaml:"ports"`   // A: list of ports / ranges to serve
	Forward string   `yaml:"forward"` // B: internal port to expose via the tunnel
	SSH     struct {
		AuthorizedKeys []string `yaml:"authorized_keys"` // C: SSH authorized-key sources (file / literal)
	} `yaml:"ssh"`
	Exec struct {
		Command string `yaml:"command"` // C: command run per connection, space-separated
	} `yaml:"exec"`
	Key         string   `yaml:"key"`          // tailcat key name/path/"new"; empty = default saved key
	DerpMapURL  string   `yaml:"derpmap_url"`  // custom DERP map URL (also read from TAILCAT_DERPMAP_URL by tailcat)
	Allow       []string `yaml:"allow"`        // nodekey allowlist (`nodekey:...`); empty = allow all
	PSK         *bool    `yaml:"psk"`          // include WireGuard preshared key in address (tailcat default true)
	FullAddress bool     `yaml:"full_address"` // embed DERP server info in the address
	Verbose     bool     `yaml:"verbose"`
	JSON        bool     `yaml:"json"` // emit {"listenAddr":...} JSON to stdout
}

// Config is the parsed aria-tailcat configuration.
type Config struct {
	Serve ServeConfig `yaml:"serve"`
}

// defaultConfig returns the built-in defaults used before any YAML or CLI input.
func defaultConfig() Config {
	return Config{
		Serve: ServeConfig{
			Mode: "listen",
		},
	}
}

// LoadConfig reads and parses a YAML configuration file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// applyConfig overlays non-empty scalar fields and non-empty slices from src onto
// dst. A non-nil PSK pointer always wins. Boolean flags OR together so an explicit
// true in either source is preserved.
func applyConfig(dst *Config, src *Config) {
	applyServe(&dst.Serve, &src.Serve)
}

func applyServe(dst *ServeConfig, src *ServeConfig) {
	if src.Mode != "" {
		dst.Mode = src.Mode
	}
	if len(src.Ports) > 0 {
		dst.Ports = src.Ports
	}
	if src.Forward != "" {
		dst.Forward = src.Forward
	}
	if len(src.SSH.AuthorizedKeys) > 0 {
		dst.SSH.AuthorizedKeys = src.SSH.AuthorizedKeys
	}
	if src.Exec.Command != "" {
		dst.Exec.Command = src.Exec.Command
	}
	if src.Key != "" {
		dst.Key = src.Key
	}
	if src.DerpMapURL != "" {
		dst.DerpMapURL = src.DerpMapURL
	}
	if len(src.Allow) > 0 {
		dst.Allow = src.Allow
	}
	if src.PSK != nil {
		dst.PSK = src.PSK
	}
	dst.FullAddress = dst.FullAddress || src.FullAddress
	dst.Verbose = dst.Verbose || src.Verbose
	dst.JSON = dst.JSON || src.JSON
}

// validate checks that the serve configuration is internally consistent.
func (s ServeConfig) validate() error {
	switch s.Mode {
	case "", "listen", "forward", "ssh", "exec":
		// handled below
	default:
		// raw tailcat service name (e.g. all / exit-node / files): accepted
		return nil
	}
	switch s.Mode {
	case "listen", "forward":
		ports := s.Ports
		if s.Mode == "forward" && s.Forward != "" {
			ports = []string{s.Forward}
		}
		if len(ports) == 0 {
			return fmt.Errorf("serve mode %q requires at least one port (serve.ports / serve.forward)", s.Mode)
		}
	case "exec":
		if strings.TrimSpace(s.Exec.Command) == "" {
			return fmt.Errorf("serve mode %q requires serve.exec.command", s.Mode)
		}
	}
	return nil
}
