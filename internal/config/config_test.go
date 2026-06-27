package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAppliesDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
groups:
  - name: Ops
    links:
      - name: Grafana
        url: https://grafana.example.com
`

	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Title != "Links" {
		t.Fatalf("title = %q, want %q", cfg.Title, "Links")
	}
	if cfg.ListenAddr != ":8080" {
		t.Fatalf("listen addr = %q, want %q", cfg.ListenAddr, ":8080")
	}
}

func TestValidateRejectsMissingGroups(t *testing.T) {
	t.Parallel()

	cfg := &Config{}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "at least one group") {
		t.Fatalf("validate error = %v, want missing group error", err)
	}
}

func TestValidateRejectsMissingLinkURL(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Groups: []Group{
			{
				Name: "Ops",
				Links: []Link{
					{Name: "Grafana"},
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "url is required") {
		t.Fatalf("validate error = %v, want missing url error", err)
	}
}
