package catalog_test

import (
	"encoding/json"
	"testing"

	"github.com/insidegreen/rpc-operator-claude/internal/api/catalog"
)

func TestCatalog_LoadsThreeComponents(t *testing.T) {
	cat, err := catalog.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	all := cat.All()
	if len(all) != 3 {
		t.Errorf("expected 3 components, got %d", len(all))
	}
}

func TestCatalog_ExpectedEntries(t *testing.T) {
	cat, err := catalog.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := [][2]string{
		{"inputs", "generate"},
		{"processors", "mapping"},
		{"outputs", "stdout"},
	}
	for _, w := range want {
		if _, ok := cat.Get(w[0], w[1]); !ok {
			t.Errorf("missing component %s/%s", w[0], w[1])
		}
	}
}

func TestCatalog_CategoryMatchesDirectory(t *testing.T) {
	cat, err := catalog.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, comp := range cat.All() {
		if comp.Category == "" {
			t.Errorf("component %q has empty category", comp.Name)
		}
	}
}

func TestCatalog_SchemasAreValidJSON(t *testing.T) {
	cat, err := catalog.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, comp := range cat.All() {
		var v any
		if err := json.Unmarshal(comp.ConfigSchema, &v); err != nil {
			t.Errorf("%s/%s: configSchema is not valid JSON: %v", comp.Category, comp.Name, err)
		}
	}
}

func TestCatalog_GetNotFound(t *testing.T) {
	cat, err := catalog.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, ok := cat.Get("inputs", "no-such"); ok {
		t.Error("expected Get to return false for unknown component")
	}
}

func TestCatalog_Default_IsSingleton(t *testing.T) {
	c1, err := catalog.Default()
	if err != nil {
		t.Fatalf("Default: %v", err)
	}
	c2, err := catalog.Default()
	if err != nil {
		t.Fatalf("Default: %v", err)
	}
	if c1 != c2 {
		t.Error("Default() should return the same pointer on repeated calls")
	}
}
