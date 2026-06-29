package config

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Title      string      `yaml:"title"`
	Subtitle   string      `yaml:"subtitle"`
	ListenAddr string      `yaml:"listen_addr"`
	Dashboards []Dashboard `yaml:"dashboards"`
}

type Dashboard struct {
	Path     string  `yaml:"path"`
	Title    string  `yaml:"title"`
	Subtitle string  `yaml:"subtitle"`
	Groups   []Group `yaml:"groups"`
}

type Group struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Links       []Link `yaml:"links"`
}

type Link struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	URL         string `yaml:"url"`
	Icon        string `yaml:"icon"`
}

func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}

	if cfg.Title == "" {
		cfg.Title = "Dashboards"
	}
	if cfg.Subtitle == "" {
		cfg.Subtitle = "Configured dashboards"
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":8080"
	}
	for i := range cfg.Dashboards {
		cfg.Dashboards[i].Path = normalizeDashboardPath(cfg.Dashboards[i].Path)
		if cfg.Dashboards[i].Title == "" {
			cfg.Dashboards[i].Title = "Links"
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if len(c.Dashboards) == 0 {
		return errors.New("config requires at least one dashboard")
	}

	seenPaths := make(map[string]struct{}, len(c.Dashboards))
	for di, dashboard := range c.Dashboards {
		if dashboard.Path == "" {
			return fmt.Errorf("dashboards[%d].path is required", di)
		}
		if dashboard.Path == "/" {
			return fmt.Errorf("dashboards[%d].path cannot be /", di)
		}
		if !strings.HasPrefix(dashboard.Path, "/") {
			return fmt.Errorf("dashboards[%d].path must start with /", di)
		}
		if _, ok := seenPaths[dashboard.Path]; ok {
			return fmt.Errorf("dashboards[%d].path duplicates another dashboard path", di)
		}
		seenPaths[dashboard.Path] = struct{}{}

		if len(dashboard.Groups) == 0 {
			return fmt.Errorf("dashboards[%d].groups requires at least one group", di)
		}

		for gi, group := range dashboard.Groups {
			if strings.TrimSpace(group.Name) == "" {
				return fmt.Errorf("dashboards[%d].groups[%d].name is required", di, gi)
			}
			if len(group.Links) == 0 {
				return fmt.Errorf("dashboards[%d].groups[%d].links requires at least one link", di, gi)
			}
			for li, link := range group.Links {
				if strings.TrimSpace(link.Name) == "" {
					return fmt.Errorf("dashboards[%d].groups[%d].links[%d].name is required", di, gi, li)
				}
				if strings.TrimSpace(link.URL) == "" {
					return fmt.Errorf("dashboards[%d].groups[%d].links[%d].url is required", di, gi, li)
				}
			}
		}
	}

	return nil
}

func normalizeDashboardPath(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	cleaned := path.Clean(raw)
	if cleaned == "." {
		return ""
	}
	return cleaned
}
