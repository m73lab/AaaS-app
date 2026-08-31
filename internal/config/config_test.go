package config

import (
	"os"
	"testing"
)

// TestLoadConfigs ensures both shipped config files parse and apply defaults.
func TestLoadConfigs(t *testing.T) {
	for _, p := range []string{"../../config.yaml", "../../config.homelab02.yaml.template"} {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("config %s not present", p)
		}
		c, err := Load(p)
		if err != nil {
			t.Fatalf("load %s: %v", p, err)
		}
		if c.Server.Listen == "" {
			t.Fatalf("empty listen in %s", p)
		}
		if c.RuntimePolicy() == nil {
			t.Fatalf("nil policy in %s", p)
		}
	}
}
