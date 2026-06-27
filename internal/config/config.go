package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Title      string  `yaml:"title"`
	Subtitle   string  `yaml:"subtitle"`
	ListenAddr string  `yaml:"listen_addr"`
	Groups     []Group `yaml:"groups"`
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
		cfg.Title = "Links"
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":8080"
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if len(c.Groups) == 0 {
		return errors.New("config requires at least one group")
	}

	for gi, group := range c.Groups {
		if strings.TrimSpace(group.Name) == "" {
			return fmt.Errorf("groups[%d].name is required", gi)
		}
		if len(group.Links) == 0 {
			return fmt.Errorf("groups[%d].links requires at least one link", gi)
		}
		for li, link := range group.Links {
			if strings.TrimSpace(link.Name) == "" {
				return fmt.Errorf("groups[%d].links[%d].name is required", gi, li)
			}
			if strings.TrimSpace(link.URL) == "" {
				return fmt.Errorf("groups[%d].links[%d].url is required", gi, li)
			}
		}
	}

	return nil
}
