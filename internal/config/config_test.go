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
dashboards:
  - path: /ops
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

	if cfg.Title != "Dashboards" {
		t.Fatalf("title = %q, want %q", cfg.Title, "Dashboards")
	}
	if cfg.Subtitle != "Configured dashboards" {
		t.Fatalf("subtitle = %q, want %q", cfg.Subtitle, "Configured dashboards")
	}
	if cfg.ListenAddr != ":8080" {
		t.Fatalf("listen addr = %q, want %q", cfg.ListenAddr, ":8080")
	}
	if len(cfg.Dashboards) != 1 {
		t.Fatalf("dashboards = %d, want 1", len(cfg.Dashboards))
	}
	if cfg.Dashboards[0].Path != "/ops" {
		t.Fatalf("dashboard path = %q, want %q", cfg.Dashboards[0].Path, "/ops")
	}
}

func TestValidateRejectsMissingDashboards(t *testing.T) {
	t.Parallel()

	cfg := &Config{}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "at least one dashboard") {
		t.Fatalf("validate error = %v, want missing dashboard error", err)
	}
}

func TestValidateRejectsMissingLinkURL(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Dashboards: []Dashboard{
			{
				Path: "/ops",
				Groups: []Group{
					{
						Name: "Ops",
						Links: []Link{
							{Name: "Grafana"},
						},
					},
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "url is required") {
		t.Fatalf("validate error = %v, want missing url error", err)
	}
}

func TestValidateRejectsDuplicateDashboardPath(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Dashboards: []Dashboard{
			{Path: "/ops", Groups: []Group{{Name: "Ops", Links: []Link{{Name: "Grafana", URL: "https://grafana.example.com"}}}}},
			{Path: "/ops", Groups: []Group{{Name: "Ops 2", Links: []Link{{Name: "Argo", URL: "https://argo.example.com"}}}}},
		},
	}

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "duplicates another dashboard path") {
		t.Fatalf("validate error = %v, want duplicate path error", err)
	}
}
