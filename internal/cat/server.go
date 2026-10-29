package cat

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
)

// Options is the fully-resolved runtime configuration.
type Options struct {
	ConfigFile string
	Bin        string
	Serve      ServeConfig
}

// RegisterFlags registers all CLI flags on the given command.
func RegisterFlags(cmd *cobra.Command) {
	cmd.Flags().String("config", "", "Path to YAML config file")
	cmd.Flags().String("bin", "", "Path to the tailcat binary (default packaged path, then PATH)")
	cmd.Flags().String("serve-mode", "", "Serve mode: listen | forward | ssh | exec (or a raw tailcat service name)")
	cmd.Flags().StringArray("serve-ports", nil, "Ports/ranges to serve (listen/forward mode)")
	cmd.Flags().String("serve-forward", "", "Internal port to expose via the tunnel (forward mode)")
	cmd.Flags().String("serve-key", "", "tailcat key name/path; 'new' for ephemeral; empty = default saved key")
	cmd.Flags().String("serve-derpmap-url", "", "Custom DERP map URL (also TAILCAT_DERPMAP_URL)")
	cmd.Flags().StringArray("serve-allow", nil, "nodekey allowlist (nodekey:...); empty = allow all")
	cmd.Flags().Bool("serve-psk", true, "Include WireGuard preshared key in the address")
	cmd.Flags().Bool("serve-full-address", false, "Embed DERP server info in the address")
	cmd.Flags().Bool("serve-verbose", false, "Enable verbose logging")
	cmd.Flags().Bool("serve-json", false, "Emit {\"listenAddr\":...} JSON to stdout")
	cmd.Flags().StringArray("ssh-authorized-keys", nil, "SSH authorized-key sources (file/literal/name@github)")
	cmd.Flags().String("exec-command", "", "Command run per connection (exec mode), space-separated")
}

// FlagsFromCmd resolves Options from CLI flags, environment, and an optional
// YAML config file. Precedence (lowest to highest):
//
//	built-in defaults -> YAML file -> CLI flags
//
// tailcat itself reads TAILCAT_DERPMAP_URL from the environment for the DERP map.
func FlagsFromCmd(cmd *cobra.Command) (Options, error) {
	var o Options

	o.ConfigFile, _ = cmd.Flags().GetString("config")

	cfg := defaultConfig()
	if o.ConfigFile != "" {
		loaded, err := LoadConfig(o.ConfigFile)
		if err != nil {
			return o, fmt.Errorf("load config %q: %w", o.ConfigFile, err)
		}
		applyConfig(&cfg, loaded)
	}

	setStr := func(flag, cur string) string {
		if cmd.Flags().Changed(flag) {
			if v, err := cmd.Flags().GetString(flag); err == nil && strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		}
		return cur
	}
	setStrArray := func(flag string, cur []string) []string {
		if cmd.Flags().Changed(flag) {
			if v, err := cmd.Flags().GetStringArray(flag); err == nil && len(v) > 0 {
				return v
			}
		}
		return cur
	}
	changedBool := func(flag string) bool {
		if !cmd.Flags().Changed(flag) {
			return false
		}
		v, err := cmd.Flags().GetBool(flag)
		return err == nil && v
	}

	o.Bin = setStr("bin", cfg.Bin)

	sv := cfg.Serve
	sv.Mode = setStr("serve-mode", sv.Mode)
	if cmd.Flags().Changed("serve-ports") {
		if v, err := cmd.Flags().GetStringArray("serve-ports"); err == nil {
			sv.Ports = v
		}
	}
	sv.Forward = setStr("serve-forward", sv.Forward)
	sv.Key = setStr("serve-key", sv.Key)
	sv.DerpMapURL = setStr("serve-derpmap-url", sv.DerpMapURL)
	sv.Allow = setStrArray("serve-allow", sv.Allow)
	sv.SSH.AuthorizedKeys = setStrArray("ssh-authorized-keys", sv.SSH.AuthorizedKeys)
	if cmd.Flags().Changed("serve-psk") {
		if v, err := cmd.Flags().GetBool("serve-psk"); err == nil {
			b := v
			sv.PSK = &b
		}
	}
	sv.FullAddress = sv.FullAddress || changedBool("serve-full-address")
	sv.Verbose = sv.Verbose || changedBool("serve-verbose")
	sv.JSON = sv.JSON || changedBool("serve-json")
	if cmd.Flags().Changed("exec-command") {
		if v, err := cmd.Flags().GetString("exec-command"); err == nil {
			sv.Exec.Command = strings.TrimSpace(v)
		}
	}
	o.Serve = sv

	if strings.TrimSpace(o.Serve.Mode) == "" {
		o.Serve.Mode = "listen"
	}
	if err := o.Serve.validate(); err != nil {
		return o, err
	}
	return o, nil
}

// Run builds the tailcat serve argument vector and launches it until ctx is
// cancelled.
func Run(ctx context.Context, o Options) error {
	args, err := BuildServeArgs(o.Serve)
	if err != nil {
		return err
	}
	log.Printf("aria-tailcat: serve mode=%s", o.Serve.Mode)
	return Launch(ctx, o.Bin, args)
}
