package cat

import (
	"os"
	"path/filepath"
	"testing"
)

func boolPtr(b bool) *bool { return &b }

func TestDefaultConfig(t *testing.T) {
	c := defaultConfig()
	if c.Serve.Mode != "listen" {
		t.Errorf("default mode = %q, want listen", c.Serve.Mode)
	}
}

func TestLoadConfig(t *testing.T) {
	yaml := `
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
		Serve: ServeConfig{
			Mode: "ssh",
			SSH: struct {
				AuthorizedKeys []string `yaml:"authorized_keys"`
			}{AuthorizedKeys: []string{"alice@github"}},
			PSK: boolPtr(true),
		},
	}
	applyConfig(&dst, src)
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

func TestBuildServer(t *testing.T) {
	t.Run("listen ports", func(t *testing.T) {
		s, err := BuildServer(ServeConfig{Mode: "listen", Key: "new", Ports: []string{"8080", "9090"}})
		if err != nil {
			t.Fatal(err)
		}
		if len(s.ServedTCPPorts) != 2 {
			t.Errorf("ServedTCPPorts = %v", s.ServedTCPPorts)
		}
		if s.OnTCP == nil {
			t.Error("OnTCP handler not set")
		}
		if s.OnTCP(8080) == nil {
			t.Error("OnTCP(8080) should return a forward handler")
		}
	})
	t.Run("forward", func(t *testing.T) {
		s, err := BuildServer(ServeConfig{Mode: "forward", Key: "new", Forward: "8080"})
		if err != nil {
			t.Fatal(err)
		}
		if len(s.ServedTCPPorts) != 1 || s.ServedTCPPorts[0].First != 8080 {
			t.Errorf("ServedTCPPorts = %v", s.ServedTCPPorts)
		}
	})
	t.Run("ssh", func(t *testing.T) {
		s, err := BuildServer(ServeConfig{Mode: "ssh", Key: "new"})
		if err != nil {
			t.Fatal(err)
		}
		if s.OnTCP == nil || s.OnTCP(22) == nil {
			t.Error("ssh mode must handle port 22")
		}
	})
	t.Run("exec", func(t *testing.T) {
		s, err := BuildServer(ServeConfig{Mode: "exec", Key: "new", Exec: struct {
			Command string `yaml:"command"`
		}{Command: "/bin/sh -c echo hi"}})
		if err != nil {
			t.Fatal(err)
		}
		if s.OnTCP == nil || s.OnTCP(1234) == nil {
			t.Error("exec mode must handle connections on any port")
		}
	})
	t.Run("psk disabled", func(t *testing.T) {
		s, err := BuildServer(ServeConfig{Mode: "listen", Key: "new", Ports: []string{"80"}, PSK: boolPtr(false)})
		if err != nil {
			t.Fatal(err)
		}
		if !s.DisablePresharedKey {
			t.Error("DisablePresharedKey should be true when psk=false")
		}
	})
	t.Run("listen needs ports", func(t *testing.T) {
		if _, err := BuildServer(ServeConfig{Mode: "listen", Key: "new"}); err == nil {
			t.Error("expected error for listen without ports")
		}
	})
	t.Run("exec needs command", func(t *testing.T) {
		if _, err := BuildServer(ServeConfig{Mode: "exec", Key: "new"}); err == nil {
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
