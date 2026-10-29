package cat

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
)

func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "x"}
	RegisterFlags(cmd)
	return cmd
}

func TestFlagsFromCmdDefaults(t *testing.T) {
	cmd := newTestCmd()
	cmd.SetArgs([]string{})
	if err := cmd.ParseFlags([]string{}); err != nil {
		t.Fatal(err)
	}
	// With no config and no flags there is nothing to serve, so FlagsFromCmd
	// must reject the empty "listen" mode (ports required).
	if _, err := FlagsFromCmd(cmd); err == nil {
		t.Fatal("expected error for empty config with no ports")
	}
}

func TestFlagsFromCmdExplicitPorts(t *testing.T) {
	cmd := newTestCmd()
	cmd.SetArgs([]string{})
	if err := cmd.ParseFlags([]string{"--serve-ports", "8080"}); err != nil {
		t.Fatal(err)
	}
	o, err := FlagsFromCmd(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if o.Serve.Mode != "listen" {
		t.Errorf("mode = %q, want listen", o.Serve.Mode)
	}
	if len(o.Serve.Ports) != 1 || o.Serve.Ports[0] != "8080" {
		t.Errorf("ports = %v", o.Serve.Ports)
	}
}

func TestFlagsFromCmdConfigOverride(t *testing.T) {
	dir := t.TempDir()
	p := dir + "/c.yaml"
	if err := os.WriteFile(p, []byte("serve:\n  mode: forward\n  forward: \"9000\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := newTestCmd()
	cmd.SetArgs([]string{})
	if err := cmd.ParseFlags([]string{"--config", p}); err != nil {
		t.Fatal(err)
	}
	o, err := FlagsFromCmd(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if o.Serve.Mode != "forward" {
		t.Errorf("mode = %q", o.Serve.Mode)
	}
	if o.Serve.Forward != "9000" {
		t.Errorf("forward = %q", o.Serve.Forward)
	}
}

func TestFlagsFromCmdCLIPrecedence(t *testing.T) {
	cmd := newTestCmd()
	cmd.SetArgs([]string{})
	if err := cmd.ParseFlags([]string{"--serve-mode", "ssh", "--ssh-authorized-keys", "bob@github"}); err != nil {
		t.Fatal(err)
	}
	o, err := FlagsFromCmd(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if o.Serve.Mode != "ssh" {
		t.Errorf("mode = %q", o.Serve.Mode)
	}
	if len(o.Serve.SSH.AuthorizedKeys) != 1 || o.Serve.SSH.AuthorizedKeys[0] != "bob@github" {
		t.Errorf("keys = %v", o.Serve.SSH.AuthorizedKeys)
	}
}

func TestFlagsFromCmdCLIOverridesConfig(t *testing.T) {
	dir := t.TempDir()
	p := dir + "/c.yaml"
	if err := os.WriteFile(p, []byte("serve:\n  mode: listen\n  ports:\n    - \"1111\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := newTestCmd()
	cmd.SetArgs([]string{})
	// CLI --serve-mode should override the YAML mode.
	if err := cmd.ParseFlags([]string{"--config", p, "--serve-mode", "forward", "--serve-forward", "2222"}); err != nil {
		t.Fatal(err)
	}
	o, err := FlagsFromCmd(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if o.Serve.Mode != "forward" {
		t.Errorf("mode = %q, want forward (CLI override)", o.Serve.Mode)
	}
	if o.Serve.Forward != "2222" {
		t.Errorf("forward = %q", o.Serve.Forward)
	}
}
