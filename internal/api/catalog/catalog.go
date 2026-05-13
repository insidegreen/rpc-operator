package catalog

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"sync"
)

//go:embed data/*/*.json
var dataFS embed.FS

// Component describes one Redpanda Connect component available in the catalog.
type Component struct {
	Name          string          `json:"name"`
	Category      string          `json:"category"`
	Status        string          `json:"status"`
	Summary       string          `json:"summary"`
	BodyKind      string          `json:"bodyKind"`
	ReplicaSafety string          `json:"replicaSafety"`
	ConfigSchema  json.RawMessage `json:"configSchema"`
}

// Catalog holds the loaded component entries indexed for fast lookup.
type Catalog struct {
	byKey map[string]*Component // key: "<category>/<name>"
	all   []*Component
}

// Load parses all embedded JSON files and returns a populated Catalog.
func Load() (*Catalog, error) {
	c := &Catalog{byKey: map[string]*Component{}}
	err := fs.WalkDir(dataFS, "data", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".json") {
			return err
		}
		b, err := dataFS.ReadFile(p)
		if err != nil {
			return fmt.Errorf("read %s: %w", p, err)
		}
		var comp Component
		if err := json.Unmarshal(b, &comp); err != nil {
			return fmt.Errorf("parse %s: %w", p, err)
		}
		// Verify the file's directory matches the component's category.
		wantCategory := path.Base(path.Dir(p))
		if comp.Category != wantCategory {
			return fmt.Errorf("%s: category %q != directory %q", p, comp.Category, wantCategory)
		}
		c.byKey[comp.Category+"/"+comp.Name] = &comp
		c.all = append(c.all, &comp)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

// Get returns a component by category and name, or false if not found.
func (c *Catalog) Get(category, name string) (*Component, bool) {
	comp, ok := c.byKey[category+"/"+name]
	return comp, ok
}

// All returns a copy of all components in the catalog.
func (c *Catalog) All() []*Component {
	out := make([]*Component, len(c.all))
	copy(out, c.all)
	return out
}

var (
	once    sync.Once
	cached  *Catalog
	loadErr error
)

// Default returns the singleton catalog, loading it on first call.
func Default() (*Catalog, error) {
	once.Do(func() { cached, loadErr = Load() })
	return cached, loadErr
}
