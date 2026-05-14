package api

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
	"github.com/insidegreen/rpc-operator-claude/internal/api/catalog"
	"github.com/insidegreen/rpc-operator-claude/internal/render"
)

// ValidationError describes a single schema or render validation failure.
type ValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// ValidatePipeline schema-validates each component against the catalog and then
// performs a render dry-run. Returns nil if the pipeline is valid.
func ValidatePipeline(p *rpcv1alpha1.Pipeline, cat *catalog.Catalog) []ValidationError {
	var errs []ValidationError
	errs = append(errs, validateComponent("spec.input", &p.Spec.Input, "inputs", cat)...)
	for i := range p.Spec.Processors {
		path := fmt.Sprintf("spec.processors[%d]", i)
		errs = append(errs, validateComponent(path, &p.Spec.Processors[i], "processors", cat)...)
	}
	errs = append(errs, validateComponent("spec.output", &p.Spec.Output, "outputs", cat)...)

	if _, rerr := render.RenderPipelineYAML(&p.Spec); rerr != nil {
		errs = append(errs, ValidationError{Path: "spec", Message: "render failed: " + rerr.Error()})
	}
	return errs
}

func validateComponent(
	path string,
	c *rpcv1alpha1.ComponentSpec,
	category string,
	cat *catalog.Catalog,
) []ValidationError {
	if c.Type == "" {
		return []ValidationError{{Path: path + ".type", Message: "type is required"}}
	}
	comp, ok := cat.Get(category, c.Type)
	if !ok {
		return []ValidationError{{
			Path: path + ".type",
			Message: fmt.Sprintf(
				"unknown %s component %q (catalog covers v0.2 starter set only)",
				category, c.Type,
			),
		}}
	}

	raw := c.Config.Raw

	if len(comp.CompositeFields) > 0 {
		// Pattern B: config itself is an array (for_each, fallback) — configSchema is
		// empty and does not apply to arrays. Rely on the render dry-run below.
		if len(comp.CompositeFields) == 1 && comp.CompositeFields[0].Field == "" {
			return nil
		}
		// Pattern A: config is an object; strip composite sub-component fields before
		// validating scalar fields against configSchema (additionalProperties: false).
		raw = stripCompositeFields(raw, comp.CompositeFields)
	}

	return validateConfig(path+".config", raw, comp.ConfigSchema)
}

// stripCompositeFields removes composite sub-component fields from a JSON object so
// that configSchema (which only describes scalar fields) can validate the remainder.
func stripCompositeFields(raw []byte, fields []catalog.CompositeField) []byte {
	if len(raw) == 0 || string(raw) == "null" {
		return []byte("{}")
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return raw // let validateConfig report the parse error
	}
	for _, cf := range fields {
		if cf.Field != "" {
			delete(m, cf.Field)
		}
	}
	stripped, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	return stripped
}

func validateConfig(path string, raw []byte, schema json.RawMessage) []ValidationError {
	if len(raw) == 0 || string(raw) == "null" {
		raw = []byte("{}")
	}

	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schema))
	if err != nil {
		return []ValidationError{{Path: path, Message: "schema parse: " + err.Error()}}
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(path, schemaDoc); err != nil {
		return []ValidationError{{Path: path, Message: "schema compile: " + err.Error()}}
	}
	sch, err := compiler.Compile(path)
	if err != nil {
		return []ValidationError{{Path: path, Message: "schema compile: " + err.Error()}}
	}

	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return []ValidationError{{Path: path, Message: "config is not valid JSON: " + err.Error()}}
	}
	if err := sch.Validate(instance); err != nil {
		return []ValidationError{{Path: path, Message: err.Error()}}
	}
	return nil
}
