package cat

import (
	"os"
	"path/filepath"
	"testing"
)

func boolPtr(b bool) *bool { return &b }

func equalArgs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestDefaultConfig(t *testing.T) {
	c := defaultConfig()
	if c.Serve.Mode != "listen" {
		t.Errorf("default mode = %q, want listen", c.Serve.Mode)
	}
}

func TestLoadConfig(t *testing.T) {
	yaml := `
bin: /opt/tailcat
serve:
  mode: forward
  forward: "8080"
  key: default
  allow:
    - nodekey:abc
  psk: false
`
	dir := t.TempDir()
	p := filepath.Join(dir, "c.yaml")
	if err := os.WriteFile(p, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.Bin != "/opt/tailcat" {
		t.Errorf("bin = %q", c.Bin)
	}
	if c.Serve.Mode != "forward" {
		t.Errorf("mode = %q", c.Serve.Mode)
	}
	if c.Serve.Forward != "8080" {
		t.Errorf("forward = %q", c.Serve.Forward)
	}
	if c.Serve.Key != "default" {
		t.Errorf("key = %q", c.Serve.Key)
	}
	if len(c.Serve.Allow) != 1 || c.Serve.Allow[0] != "nodekey:abc" {
		t.Errorf("allow = %v", c.Serve.Allow)
	}
	if c.Serve.PSK == nil || *c.Serve.PSK != false {
		t.Errorf("psk = %v", c.Serve.PSK)
	}
}

func TestApplyConfig(t *testing.T) {
	dst := defaultConfig()
	src := &Config{
		Bin: "/x/tailcat",
		Serve: ServeConfig{
			Mode: "ssh",
			SSH: struct {
				AuthorizedKeys []string `yaml:"authorized_keys"`
			}{AuthorizedKeys: []string{"alice@github"}},
			PSK: boolPtr(true),
		},
	}
	applyConfig(&dst, src)
	if dst.Bin != "/x/tailcat" {
		t.Errorf("bin = %q", dst.Bin)
	}
	if dst.Serve.Mode != "ssh" {
		t.Errorf("mode = %q", dst.Serve.Mode)
	}
	if len(dst.Serve.SSH.AuthorizedKeys) != 1 {
		t.Errorf("ssh keys = %v", dst.Serve.SSH.AuthorizedKeys)
	}
	if dst.Serve.PSK == nil || !*dst.Serve.PSK {
		t.Errorf("psk = %v", dst.Serve.PSK)
	}
}

func TestBuildServeArgs(t *testing.T) {
	t.Run("listen", func(t *testing.T) {
		args, err := BuildServeArgs(ServeConfig{Mode: "listen", Ports: []string{"8080", "9090"}})
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"serve", "8080,9090"}
		if !equalArgs(args, want) {
			t.Errorf("got %v want %v", args, want)
		}
	})
	t.Run("forward", func(t *testing.T) {
		args, err := BuildServeArgs(ServeConfig{Mode: "forward", Forward: "8080"})
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"serve", "8080"}
		if !equalArgs(args, want) {
			t.Errorf("got %v want %v", args, want)
		}
	})
	t.Run("ssh", func(t *testing.T) {
		c := ServeConfig{Mode: "ssh"}
		c.SSH.AuthorizedKeys = []string{"alice@github"}
		args, err := BuildServeArgs(c)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"serve", "--ssh-authorized-keys=alice@github", "ssh"}
		if !equalArgs(args, want) {
			t.Errorf("got %v want %v", args, want)
		}
	})
	t.Run("exec", func(t *testing.T) {
		c := ServeConfig{Mode: "exec"}
		c.Exec.Command = "/bin/sh -c echo hi"
		args, err := BuildServeArgs(c)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"serve", "exec", "--", "/bin/sh", "-c", "echo", "hi"}
		if !equalArgs(args, want) {
			t.Errorf("got %v want %v", args, want)
		}
	})
	t.Run("with flags", func(t *testing.T) {
		args, err := BuildServeArgs(ServeConfig{
			Mode: "listen", Ports: []string{"80"}, Key: "default",
			DerpMapURL: "https://x/derp.json", Allow: []string{"nodekey:a"},
			PSK: boolPtr(false), FullAddress: true, Verbose: true, JSON: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"serve", "--key=default", "--derpmap-url=https://x/derp.json",
			"--allow=nodekey:a", "--psk=false", "--full-address", "--verbose", "--json", "80"}
		if !equalArgs(args, want) {
			t.Errorf("got %v want %v", args, want)
		}
	})
	t.Run("listen needs ports", func(t *testing.T) {
		if _, err := BuildServeArgs(ServeConfig{Mode: "listen"}); err == nil {
			t.Error("expected error for listen without ports")
		}
	})
	t.Run("exec needs command", func(t *testing.T) {
		if _, err := BuildServeArgs(ServeConfig{Mode: "exec"}); err == nil {
			t.Error("expected error for exec without command")
		}
	})
}

func TestValidate(t *testing.T) {
	if err := (ServeConfig{Mode: "listen"}).validate(); err == nil {
		t.Error("listen without ports should error")
	}
	if err := (ServeConfig{Mode: "exec"}).validate(); err == nil {
		t.Error("exec without command should error")
	}
	if err := (ServeConfig{Mode: "all"}).validate(); err != nil {
		t.Errorf("raw service name should be allowed: %v", err)
	}
}
