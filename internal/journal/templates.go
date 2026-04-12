// Package journal provides the embedded journal template catalog.
// Templates are defined in YAML files alongside this file; no Go code
// changes are required to add or edit templates.
package journal

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed *.yaml
var templateFiles embed.FS

// Template is a single pre-defined journal entry template.
type Template struct {
	Category  string `yaml:"category" json:"category"`
	Sentiment string `yaml:"sentiment" json:"sentiment"`
	Title     string `yaml:"title" json:"title"`
}

// templateFile is the top-level structure of a templates YAML file.
type templateFile struct {
	Templates []Template `yaml:"templates"`
}

// Catalog holds all loaded templates, indexed by category.
type Catalog struct {
	all        []Template
	byCategory map[string][]Template
}

// Load reads all embedded YAML template files and returns a populated Catalog.
func Load() (*Catalog, error) {
	c := &Catalog{
		byCategory: make(map[string][]Template),
	}

	entries, err := fs.ReadDir(templateFiles, ".")
	if err != nil {
		return nil, fmt.Errorf("reading embedded template directory: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".yaml") {
			continue
		}

		data, err := templateFiles.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("reading template file %s: %w", name, err)
		}

		var tf templateFile
		if err := yaml.Unmarshal(data, &tf); err != nil {
			return nil, fmt.Errorf("parsing template file %s: %w", name, err)
		}

		for _, t := range tf.Templates {
			c.all = append(c.all, t)
			c.byCategory[t.Category] = append(c.byCategory[t.Category], t)
		}
	}

	return c, nil
}

// All returns every loaded template in file order.
func (c *Catalog) All() []Template {
	return c.all
}

// ByCategory returns all templates for the given category.
// Returns nil if no templates exist for that category.
func (c *Catalog) ByCategory(category string) []Template {
	return c.byCategory[category]
}

// Grouped returns a map of category → templates, suitable for API responses.
func (c *Catalog) Grouped() map[string][]Template {
	out := make(map[string][]Template, len(c.byCategory))
	for k, v := range c.byCategory {
		out[k] = v
	}
	return out
}

// NewEmptyCatalog returns a Catalog with no templates (used as a safe nil substitute).
func NewEmptyCatalog() *Catalog {
	return &Catalog{byCategory: make(map[string][]Template)}
}
