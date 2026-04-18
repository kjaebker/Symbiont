// Package kits provides embedded test kit specifications for aquarium water chemistry measurements.
// Kit definitions live in YAML files alongside this file. To add a new manufacturer or kit,
// add or edit a YAML file — no Go code changes required.
package kits

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed *.yaml
var kitFiles embed.FS

// KitSpec is the top-level structure of a kit YAML file.
type KitSpec struct {
	Manufacturer string   `yaml:"manufacturer"`
	Slug         string   `yaml:"slug"`
	Kits         []KitDef `yaml:"kits"`
}

// KitDef defines a single test kit or checker.
type KitDef struct {
	// Slug-qualified ref is "manufacturer_slug/id" — e.g. "hanna/hi772".
	// ID is the kit's unique identifier within a manufacturer's file.
	ID         string     `yaml:"id"`
	Name       string     `yaml:"name"`
	Parameter  string     `yaml:"parameter"` // must match a measurement_parameters.name
	Input      KitInput   `yaml:"input"`
	Conversion Conversion `yaml:"conversion"`

	// Ref is populated by Load and is the fully-qualified reference string ("slug/id").
	Ref string `yaml:"-"`
}

// KitInput describes what the user reads off the instrument or test kit.
type KitInput struct {
	Label string `yaml:"label"`
	Unit  string `yaml:"unit"`
}

// Conversion is a linear transformation from raw input to canonical value:
//
//	result = RawValue * Scale + Offset
//
// For kits that read directly in the canonical unit, Scale=1 and Offset=0.
type Conversion struct {
	Scale  float64 `yaml:"scale"`
	Offset float64 `yaml:"offset"`
}

// Apply computes the canonical value from a raw instrument reading.
func (c Conversion) Apply(rawValue float64) float64 {
	return rawValue*c.Scale + c.Offset
}

// Catalog holds all loaded kit definitions, indexed by their ref string ("slug/id").
type Catalog struct {
	kits    map[string]*KitDef
	byParam map[string][]*KitDef // parameter name → kits
}

// Load reads all embedded YAML kit files and returns a populated Catalog.
//
// validParams, if non-nil, is the set of known parameter names (from
// measurement_parameters). Each kit's Parameter field is validated against
// this set. Pass nil to skip validation.
func Load(validParams map[string]bool) (*Catalog, error) {
	c := &Catalog{
		kits:    make(map[string]*KitDef),
		byParam: make(map[string][]*KitDef),
	}

	entries, err := fs.ReadDir(kitFiles, ".")
	if err != nil {
		return nil, fmt.Errorf("reading embedded kit directory: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".yaml") {
			continue
		}

		data, err := kitFiles.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("reading kit file %s: %w", name, err)
		}

		var spec KitSpec
		if err := yaml.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("parsing kit file %s: %w", name, err)
		}

		// Derive slug from filename if not set in the YAML.
		if spec.Slug == "" {
			spec.Slug = strings.TrimSuffix(name, ".yaml")
		}

		for i := range spec.Kits {
			kit := &spec.Kits[i]
			ref := spec.Slug + "/" + kit.ID
			kit.Ref = ref

			if validParams != nil && !validParams[kit.Parameter] {
				return nil, fmt.Errorf("kit %q references unknown parameter %q (check %s)", ref, kit.Parameter, name)
			}

			c.kits[ref] = kit
			c.byParam[kit.Parameter] = append(c.byParam[kit.Parameter], kit)
		}
	}

	return c, nil
}

// Get returns the KitDef for the given ref ("slug/id"), or false if not found.
func (c *Catalog) Get(ref string) (*KitDef, bool) {
	k, ok := c.kits[ref]
	return k, ok
}

// ListByParameter returns all kits that measure the named parameter.
func (c *Catalog) ListByParameter(param string) []*KitDef {
	return c.byParam[param]
}

// All returns every loaded kit definition.
func (c *Catalog) All() []*KitDef {
	out := make([]*KitDef, 0, len(c.kits))
	for _, k := range c.kits {
		out = append(out, k)
	}
	return out
}

// GroupedByParameter returns all kits grouped by their parameter name.
func (c *Catalog) GroupedByParameter() map[string][]*KitDef {
	out := make(map[string][]*KitDef, len(c.byParam))
	for param, kits := range c.byParam {
		out[param] = kits
	}
	return out
}

// Apply computes the canonical value from a raw reading for the given kit ref.
func (c *Catalog) Apply(ref string, rawValue float64) (float64, error) {
	kit, ok := c.kits[ref]
	if !ok {
		return 0, fmt.Errorf("unknown kit ref %q", ref)
	}
	return kit.Conversion.Apply(rawValue), nil
}
