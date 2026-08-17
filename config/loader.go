package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads, parses, defaults, and validates a configuration file.
// The file format is inferred from its extension (.json, .yaml, .yml).
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	cfg, err := Parse(raw, filepath.Ext(path))
	if err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config %q: %w", path, err)
	}
	return cfg, nil
}

// Parse decodes a raw config document into a Config, applying defaults.
func Parse(raw []byte, ext string) (*Config, error) {
	cfg := Default()
	switch strings.ToLower(ext) {
	case ".json":
		if err := json.Unmarshal(raw, cfg); err != nil {
			return nil, err
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(raw, cfg); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported config format %q (want .json, .yaml or .yml)", ext)
	}
	cfg.applyDefaults()
	return cfg, nil
}

// applyDefaults fills zero-valued fields with defaults without clobbering
// explicitly configured values.
func (c *Config) applyDefaults() {
	d := Default()
	if c.Display.Width == 0 {
		c.Display.Width = d.Display.Width
	}
	if c.Display.Height == 0 {
		c.Display.Height = d.Display.Height
	}
	if c.Display.FPS == 0 {
		c.Display.FPS = d.Display.FPS
	}
	if c.Display.Background == "" {
		c.Display.Background = d.Display.Background
	}
	if c.Admin.Bind == "" {
		c.Admin.Bind = d.Admin.Bind
	}
}

// Save writes the config to path atomically (temp file + rename), preserving
// the original file's format based on its extension.
func (c *Config) Save(path string) error {
	ext := strings.ToLower(filepath.Ext(path))
	var raw []byte
	var err error
	switch ext {
	case ".json":
		raw, err = json.MarshalIndent(c, "", "  ")
	case ".yaml", ".yml":
		raw, err = yaml.Marshal(c)
	default:
		return fmt.Errorf("unsupported config format %q", ext)
	}
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".framego-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temp config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp config: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("commit config %q: %w", path, err)
	}
	return nil
}
