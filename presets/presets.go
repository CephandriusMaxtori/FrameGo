// Package presets provides ready-made configuration templates for common
// kiosk/mirror setups. Each preset is a complete config.Config that the
// admin UI can apply with a single click.
package presets

import "framego/config"

// Preset is a named configuration template.
type Preset struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Config      config.Config `json:"config"`
}

// All returns the built-in preset library.
func All() []Preset {
	return []Preset{
		minimal(),
		clockDate(),
		weather(),
		smartMirror(),
		infoDashboard(),
		photoFrame(),
	}
}

func minimal() Preset {
	return Preset{
		Name:        "Minimal",
		Description: "Just the clock",
		Config: config.Config{
			Display: config.Display{Width: 800, Height: 480, FPS: 1, Margin: 16, Gap: 8, Background: "#000000"},
			Admin:   config.Admin{Enabled: true, Bind: "0.0.0.0:8080"},
			Modules: []config.Module{
				{Name: "clock", Zone: "middle-center", Visible: true, Options: map[string]any{
					"format":    "15:04",
					"dateFormat": "Mon, Jan 2",
				}},
			},
		},
	}
}

func clockDate() Preset {
	return Preset{
		Name:        "Clock & Date",
		Description: "Clock on top, date below",
		Config: config.Config{
			Display: config.Display{Width: 800, Height: 480, FPS: 1, Margin: 16, Gap: 8, Background: "#000000"},
			Admin:   config.Admin{Enabled: true, Bind: "0.0.0.0:8080"},
			Modules: []config.Module{
				{Name: "clock", Zone: "top-center", Visible: true, Options: map[string]any{
					"format": "15:04",
				}},
				{Name: "date", Zone: "middle-center", Visible: true, Options: map[string]any{}},
			},
		},
	}
}

func weather() Preset {
	return Preset{
		Name:        "Weather",
		Description: "Clock, date, and weather forecast",
		Config: config.Config{
			Display: config.Display{Width: 800, Height: 480, FPS: 1, Margin: 16, Gap: 8, Background: "#000000"},
			Admin:   config.Admin{Enabled: true, Bind: "0.0.0.0:8080"},
			Modules: []config.Module{
				{Name: "clock", Zone: "top-left", Visible: true, Options: map[string]any{
					"format": "15:04",
				}},
				{Name: "date", Zone: "top-left", Visible: true, Options: map[string]any{}},
				{Name: "weather", Zone: "middle-center", Visible: true, Options: map[string]any{
					"city":         "New York",
					"units":        "metric",
					"forecastDays": 5,
				}},
			},
		},
	}
}

func smartMirror() Preset {
	return Preset{
		Name:        "Smart Mirror",
		Description: "Full smart mirror: clock, weather, calendar, system",
		Config: config.Config{
			Display: config.Display{Width: 1080, Height: 1920, FPS: 1, Margin: 24, Gap: 12, Background: "#000000"},
			Admin:   config.Admin{Enabled: true, Bind: "0.0.0.0:8080"},
			Modules: []config.Module{
				{Name: "clock", Zone: "top-center", Visible: true, Options: map[string]any{
					"format": "15:04",
				}},
				{Name: "date", Zone: "top-center", Visible: true, Options: map[string]any{}},
				{Name: "weather", Zone: "upper-left", Visible: true, Options: map[string]any{
					"city":         "New York",
					"units":        "metric",
					"forecastDays": 3,
				}},
				{Name: "moon", Zone: "upper-right", Visible: true, Options: map[string]any{}},
				{Name: "quote", Zone: "middle-center", Visible: true, Options: map[string]any{}},
				{Name: "system", Zone: "lower-left", Visible: true, Options: map[string]any{
					"showCPU": true, "showMem": true, "showDisk": false,
				}},
			},
		},
	}
}

func infoDashboard() Preset {
	return Preset{
		Name:        "Info Dashboard",
		Description: "System stats, weather, and calendar",
		Config: config.Config{
			Display: config.Display{Width: 1280, Height: 720, FPS: 1, Margin: 20, Gap: 10, Background: "#0b0f14"},
			Admin:   config.Admin{Enabled: true, Bind: "0.0.0.0:8080"},
			Modules: []config.Module{
				{Name: "clock", Zone: "top-left", Visible: true, Options: map[string]any{}},
				{Name: "date", Zone: "top-left", Visible: true, Options: map[string]any{}},
				{Name: "system", Zone: "upper-left", Visible: true, Options: map[string]any{
					"showCPU": true, "showMem": true, "showDisk": true,
				}},
				{Name: "weather", Zone: "upper-right", Visible: true, Options: map[string]any{
					"city":  "New York",
					"units": "metric",
				}},
				{Name: "calendar", Zone: "middle-center", Visible: true, Options: map[string]any{
					"url":        "https://calendar.google.com/calendar/ical/standard/public.ics",
					"days":       7,
					"maxEvents": 8,
				}},
				{Name: "quote", Zone: "bottom-center", Visible: true, Options: map[string]any{}},
			},
		},
	}
}

func photoFrame() Preset {
	return Preset{
		Name:        "Photo Frame",
		Description: "Clock with rotating photo slideshow",
		Config: config.Config{
			Display: config.Display{Width: 1920, Height: 1080, FPS: 1, Margin: 0, Gap: 0, Background: "#000000"},
			Admin:   config.Admin{Enabled: true, Bind: "0.0.0.0:8080"},
			Modules: []config.Module{
				{Name: "slideshow", Zone: "middle-center", Visible: true, Options: map[string]any{
					"dir":      "/home/user/photos",
					"interval": 10,
					"fit":      "cover",
				}},
				{Name: "clock", Zone: "bottom-center", Visible: true, Options: map[string]any{
					"format": "15:04",
				}},
			},
		},
	}
}
