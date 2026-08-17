package engine

// FieldKind describes the UI control type for a schema field.
type FieldKind string

const (
	FieldText     FieldKind = "text"
	FieldNumber   FieldKind = "number"
	FieldBoolean  FieldKind = "boolean"
	FieldSelect   FieldKind = "select"
	FieldColor    FieldKind = "color"
	FieldDuration FieldKind = "duration"
	FieldPassword FieldKind = "password"
	FieldFile     FieldKind = "file"
)

// Field describes a single configurable option for a module so the admin UI
// can render a form instead of a raw JSON editor.
type Field struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Hint        string   `json:"hint,omitempty"`
	Kind        FieldKind `json:"kind"`
	Default     string   `json:"default,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Options     []string `json:"options,omitempty"`
	Min         float64  `json:"min,omitempty"`
	Max         float64  `json:"max,omitempty"`
	Required    bool     `json:"required,omitempty"`
}

// Schema describes the configurable surface of a module.
type Schema struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Fields      []Field `json:"fields"`
}

// SchemaModule is optionally implemented by modules that expose structured
// option metadata for the admin UI form generator.
type SchemaModule interface {
	Module
	Schema() *Schema
}
