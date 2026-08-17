// Package modules provides the in-tree module registry. Community widgets
// register a factory via Register and are imported (blank import) by main.
package modules

import (
	"sort"

	"framego/engine"
)

type factory func() engine.Module

var registry = map[string]factory{}

// Register associates name with a module constructor.
func Register(name string, f factory) {
	registry[name] = f
}

// Create instantiates a module by name.
func Create(name string) (engine.Module, bool) {
	f, ok := registry[name]
	if !ok {
		return nil, false
	}
	return f(), true
}

// Names lists registered module names in sorted order.
func Names() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
