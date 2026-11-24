package tiled

import (
	"encoding/json"
	"fmt"
)

// PropertyType is the declared type of a custom property.
type PropertyType string

// Property types Tiled can export.
const (
	PropString PropertyType = "string"
	PropInt    PropertyType = "int"
	PropFloat  PropertyType = "float"
	PropBool   PropertyType = "bool"
	PropColor  PropertyType = "color"
	PropFile   PropertyType = "file"
	PropObject PropertyType = "object"
	PropClass  PropertyType = "class"
)

// Property is one typed custom property. Value holds a string (string, file),
// int64 (int), float64 (float), bool (bool), Color (color), int (object: the
// referenced object's ID, 0 for none) or map[string]any (class members).
type Property struct {
	Name       string
	Type       PropertyType
	CustomType string // the "propertytype" for class and enum properties
	Value      any
}

// Properties maps property names to values. Typed accessors return a
// fallback when a property is missing or has a different type, which keeps
// game code free of type assertions while still never guessing silently:
// use Lookup when "absent" and "wrong type" must be distinguished.
type Properties map[string]Property

// Lookup returns the named property.
func (p Properties) Lookup(name string) (Property, bool) {
	prop, ok := p[name]
	return prop, ok
}

// String returns a string or file property, or fallback.
func (p Properties) String(name, fallback string) string {
	if v, ok := p[name].Value.(string); ok {
		return v
	}
	return fallback
}

// Int returns an int property, or fallback.
func (p Properties) Int(name string, fallback int) int {
	if v, ok := p[name].Value.(int64); ok {
		return int(v)
	}
	return fallback
}

// Float returns a float property, or fallback. Int properties are widened,
// since designers often type "2" where "2.0" was intended.
func (p Properties) Float(name string, fallback float64) float64 {
	switch v := p[name].Value.(type) {
	case float64:
		return v
	case int64:
		return float64(v)
	}
	return fallback
}

// Bool returns a bool property, or fallback.
func (p Properties) Bool(name string, fallback bool) bool {
	if v, ok := p[name].Value.(bool); ok {
		return v
	}
	return fallback
}

// Color returns a color property.
func (p Properties) Color(name string) (Color, bool) {
	c, ok := p[name].Value.(Color)
	return c, ok
}

// ObjectRef returns the ID stored in an object property (0 means unset).
func (p Properties) ObjectRef(name string) (int, bool) {
	id, ok := p[name].Value.(int)
	return id, ok
}

type rawProperty struct {
	Name         string          `json:"name"`
	Type         string          `json:"type"`
	PropertyType string          `json:"propertytype"`
	Value        json.RawMessage `json:"value"`
}

func parseProperties(raw []rawProperty) (Properties, error) {
	props := make(Properties, len(raw))
	for _, rp := range raw {
		prop, err := parseProperty(rp)
		if err != nil {
			return nil, fmt.Errorf("property %q: %w", rp.Name, err)
		}
		props[rp.Name] = prop
	}
	return props, nil
}

func parseProperty(rp rawProperty) (Property, error) {
	typ := PropertyType(rp.Type)
	if typ == "" {
		typ = PropString // very old Tiled versions omitted the type
	}
	prop := Property{Name: rp.Name, Type: typ, CustomType: rp.PropertyType}
	var err error
	switch typ {
	case PropString, PropFile:
		var s string
		err = json.Unmarshal(rp.Value, &s)
		prop.Value = s
	case PropInt:
		var n json.Number
		if err = json.Unmarshal(rp.Value, &n); err == nil {
			prop.Value, err = n.Int64()
		}
	case PropFloat:
		var f float64
		err = json.Unmarshal(rp.Value, &f)
		prop.Value = f
	case PropBool:
		var b bool
		err = json.Unmarshal(rp.Value, &b)
		prop.Value = b
	case PropColor:
		var s string
		if err = json.Unmarshal(rp.Value, &s); err == nil {
			var c Color
			c, _, err = parseOptionalColor(s)
			prop.Value = c
		}
	case PropObject:
		var id int
		err = json.Unmarshal(rp.Value, &id)
		prop.Value = id
	case PropClass:
		members := map[string]any{}
		if len(rp.Value) > 0 {
			err = json.Unmarshal(rp.Value, &members)
		}
		prop.Value = members
	default:
		return Property{}, fmt.Errorf("%w: unknown property type %q", ErrInvalid, rp.Type)
	}
	if err != nil {
		return Property{}, fmt.Errorf("%w: value does not match type %q: %v", ErrInvalid, typ, err)
	}
	return prop, nil
}
