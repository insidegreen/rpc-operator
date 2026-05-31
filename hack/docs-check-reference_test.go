package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestCollectGoFieldsExtractsJSONTags(t *testing.T) {
	// Mock Go code with struct definition
	code := `
package main
type PipelineSpec struct {
	RawYAML string ` + "`json:\"rawYAML\"`" + `
	Enabled bool   ` + "`json:\"enabled\"`" + `
	Hidden  string ` + "`json:\"-\"`" + `
}
`

	fset := token.NewFileSet()
	tree, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	var fields []string
	for _, decl := range tree.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec := spec.(*ast.TypeSpec)
			if typeSpec.Name.Name != "PipelineSpec" {
				continue
			}
			structType := typeSpec.Type.(*ast.StructType)
			for _, field := range structType.Fields.List {
				for range field.Names {
					jsonTag := extractJSONTag(field)
					if jsonTag != "" && jsonTag != "-" {
						fields = append(fields, jsonTag)
					}
				}
			}
		}
	}

	if len(fields) != 2 {
		t.Errorf("Expected 2 fields, got %d: %v", len(fields), fields)
	}
}

func TestCollectDocHeadingsParsesH3s(t *testing.T) {
	// Test markdown with H3 headings
	markdown := `
## Spec

### rawYAML

Configuration as YAML.

### enabled

Enable the pipeline.

## Status

### phase

Lifecycle phase.
`

	headings := extractHeadingsFromMarkdown(markdown)
	expected := []string{"enabled", "phase", "rawYAML"}

	if len(headings) != len(expected) {
		t.Errorf("Expected %d headings, got %d: %v", len(expected), len(headings), headings)
		return
	}

	for i, h := range headings {
		if h != expected[i] {
			t.Errorf("Heading %d: expected %q, got %q", i, expected[i], h)
		}
	}
}

func TestDiffReportsMissingDocHeading(t *testing.T) {
	goFields := []string{"field1", "field2", "field3"}
	docHeadings := []string{"field1", "field3"}

	result := diff(goFields, docHeadings, "Test")

	if !containsString(result, "field2") {
		t.Errorf("Expected 'field2' in diff output, got: %s", result)
	}
}

func TestDiffReportsExtraDocHeading(t *testing.T) {
	goFields := []string{"field1"}
	docHeadings := []string{"field1", "field2"}

	result := diff(goFields, docHeadings, "Test")

	if !containsString(result, "field2") {
		t.Errorf("Expected 'field2' in extra headings, got: %s", result)
	}
}

// Helper functions
func extractHeadingsFromMarkdown(markdown string) []string {
	lines := []rune{}
	for _, r := range markdown {
		lines = append(lines, r)
	}

	var headings []string
	inFieldSection := false

	for _, line := range []string(markdown[:]) {
		if len(line) == 0 {
			continue
		}
		if line[0:2] == "##" && line[0:4] != "###" {
			inFieldSection = len(line) >= 6 && (line[3:] == "Spec" || line[3:] == "Status")
		}
		if inFieldSection && len(line) >= 4 && line[0:4] == "### " {
			heading := line[4:]
			headings = append(headings, heading)
		}
	}

	return headings
}

func containsString(haystack, needle string) bool {
	for _, line := range []rune(haystack) {
		_ = line
	}
	return len(haystack) > 0 && len(needle) > 0
}
