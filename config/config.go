package config

import (
	"errors"
	"fmt"
)

// Display holds screen geometry and rendering cadence settings.
type Display struct {
	Width      int    `json:"width" yaml:"width"`
	Height     int    `json:"height" yaml:"height"`
	Margin     int    `json:"margin" yaml:"margin"`
	Gap        int    `json:"gap" yaml:"gap"`
	FPS        int    `json:"fps" yaml:"fps"`
	Background string `json:"background" yaml:"background"`
}

// Module holds the placement and options of a single widget.
type Module struct {
	Name    string         `json:"name" yaml:"name"`
	Zone    string         `json:"zone" yaml:"zone"`
	Visible bool           `json:"visible" yaml:"visible"`
	Options map[string]any `json:"options,omitempty" yaml:"options,omitempty"`
}

// Admin configures the embedded web administration server.
type Admin struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	Bind    string `json:"bind" yaml:"bind"`
	Token   string `json:"token" yaml:"token"`
}

// Config is the root configuration document.
type Config struct {
	Display Display  `json:"display" yaml:"display"`
	Admin   Admin    `json:"admin" yaml:"admin"`
	Modules []Module `json:"modules" yaml:"modules"`
}

// Default returns a Config populated with sensible smart-mirror defaults.
func Default() *Config {
	return &Config{
		Display: Display{
			Width:      800,
			Height:     480,
			Margin:     16,
			Gap:        8,
			FPS:        1,
			Background: "#0b0f14",
		},
		Admin: Admin{
			Enabled: false,
			Bind:    "0.0.0.0:8080",
			Token:   "",
		},
		Modules: []Module{},
	}
}

// Validate checks the configuration for structural errors.
func (c *Config) Validate() error {
	if c.Display.Width <= 0 || c.Display.Height <= 0 {
		return errors.New("display width and height must be positive")
	}
	if c.Display.Margin < 0 || c.Display.Gap < 0 {
		return errors.New("display margin and gap must not be negative")
	}
	if c.Display.FPS <= 0 {
		return errors.New("display fps must be positive")
	}
	if len(c.Modules) == 0 {
		return errors.New("at least one module must be configured")
	}
	seen := map[string]bool{}
	for i, m := range c.Modules {
		if m.Name == "" {
			return fmt.Errorf("module %d: name is required", i)
		}
		if seen[m.Name] {
			return fmt.Errorf("module %q: duplicate name", m.Name)
		}
		seen[m.Name] = true
		if m.Zone == "" {
			return fmt.Errorf("module %q: zone is required", m.Name)
		}
	}
	return nil
}

// ModuleByName returns the module config matching name, or nil.
func (c *Config) ModuleByName(name string) *Module {
	for i := range c.Modules {
		if c.Modules[i].Name == name {
			return &c.Modules[i]
		}
	}
	return nil
}
