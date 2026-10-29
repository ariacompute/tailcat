package cat

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// defaultBin is the packaged tailcat binary path on ariacompute Debian hosts.
const defaultBin = "/usr/lib/aria-tailcat/tailcat"

// resolveBin returns the tailcat binary to exec. An explicit path wins; otherwise
// we prefer the packaged path, then fall back to PATH lookup.
func resolveBin(bin string) (string, error) {
	bin = strings.TrimSpace(bin)
	if bin != "" {
		return bin, nil
	}
	if _, err := os.Stat(defaultBin); err == nil {
		return defaultBin, nil
	}
	return "tailcat", nil
}

// BuildServeArgs translates a ServeConfig into the argument vector for
// `tailcat serve` (excluding the program name). It returns an error if the
// configuration cannot form a valid serve invocation.
func BuildServeArgs(s ServeConfig) ([]string, error) {
	if strings.TrimSpace(s.Mode) == "" {
		s.Mode = "listen"
	}
	if err := s.validate(); err != nil {
		return nil, err
	}

	args := []string{"serve"}

	if s.Key != "" {
		args = append(args, "--key="+s.Key)
	}
	if s.DerpMapURL != "" {
		args = append(args, "--derpmap-url="+s.DerpMapURL)
	}
	if len(s.Allow) > 0 {
		args = append(args, "--allow="+strings.Join(s.Allow, ","))
	}
	if s.PSK != nil {
		args = append(args, fmt.Sprintf("--psk=%t", *s.PSK))
	}
	if s.FullAddress {
		args = append(args, "--full-address")
	}
	if s.Verbose {
		args = append(args, "--verbose")
	}
	if s.JSON {
		args = append(args, "--json")
	}

	switch s.Mode {
	case "listen", "forward":
		ports := s.Ports
		if s.Mode == "forward" {
			if s.Forward != "" {
				ports = []string{s.Forward}
			} else if len(ports) == 1 {
				ports = []string{ports[0]}
			}
		}
		if len(ports) == 0 {
			return nil, fmt.Errorf("serve mode %q requires at least one port", s.Mode)
		}
		args = append(args, strings.Join(ports, ","))
	case "ssh":
		if len(s.SSH.AuthorizedKeys) > 0 {
			args = append(args, "--ssh-authorized-keys="+strings.Join(s.SSH.AuthorizedKeys, ","))
		}
		args = append(args, "ssh")
	case "exec":
		cmd := strings.TrimSpace(s.Exec.Command)
		if cmd == "" {
			return nil, fmt.Errorf("serve mode %q requires serve.exec.command", s.Mode)
		}
		args = append(args, "exec", "--")
		args = append(args, strings.Fields(cmd)...)
	default:
		// Raw tailcat service name (all / exit-node / files / ...).
		args = append(args, s.Mode)
	}
	return args, nil
}

// Launch execs the tailcat binary with the given serve args, streaming its
// stdio and forwarding graceful termination (SIGTERM) on context cancellation.
func Launch(ctx context.Context, bin string, args []string) error {
	bin, err := resolveBin(bin)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return cmd.Process.Signal(syscall.SIGTERM)
	}

	log.Printf("aria-tailcat: exec %s %s", bin, strings.Join(args, " "))
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start tailcat: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		// Context cancellation is expected on graceful shutdown.
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	return nil
}
