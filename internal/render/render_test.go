/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package render_test

import (
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
	"github.com/insidegreen/rpc-operator-claude/internal/render"
)

func TestRenderPipelineYAML_Generate(t *testing.T) {
	spec := &rpcv1alpha1.PipelineSpec{
		Input: rpcv1alpha1.ComponentSpec{
			Type: "generate",
			Config: runtime.RawExtension{Raw: []byte(
				`{"mapping":"root = \"hello\"","interval":"1s","count":5}`,
			)},
		},
		Processors: []rpcv1alpha1.ComponentSpec{{
			Type: "mapping",
			Config: runtime.RawExtension{Raw: []byte(
				`{"mapping":"root = content().uppercase()"}`,
			)},
		}},
		Output: rpcv1alpha1.ComponentSpec{
			Type:   "stdout",
			Config: runtime.RawExtension{Raw: []byte(`{}`)},
		},
	}

	got, err := render.RenderPipelineYAML(spec)
	if err != nil {
		t.Fatalf("RenderPipelineYAML: %v", err)
	}

	mustContain := []string{
		"input:",
		"generate:",
		"interval: 1s",
		"count: 5",
		"processors:",
		"mapping:",
		"output:",
		"stdout:",
		"http:",
		"address: 0.0.0.0:4195",
	}
	for _, want := range mustContain {
		if !strings.Contains(got, want) {
			t.Errorf("rendered YAML missing %q\n--- output ---\n%s", want, got)
		}
	}
}

func TestRenderPipelineYAML_EmptyConfig(t *testing.T) {
	spec := &rpcv1alpha1.PipelineSpec{
		Input:  rpcv1alpha1.ComponentSpec{Type: "stdin"},
		Output: rpcv1alpha1.ComponentSpec{Type: "stdout"},
	}

	got, err := render.RenderPipelineYAML(spec)
	if err != nil {
		t.Fatalf("RenderPipelineYAML: %v", err)
	}
	for _, want := range []string{"stdin: {}", "stdout: {}"} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered YAML missing %q\n--- output ---\n%s", want, got)
		}
	}
}

func TestRenderPipelineYAML_NullConfig(t *testing.T) {
	spec := &rpcv1alpha1.PipelineSpec{
		Input:  rpcv1alpha1.ComponentSpec{Type: "stdin", Config: runtime.RawExtension{Raw: []byte("null")}},
		Output: rpcv1alpha1.ComponentSpec{Type: "stdout"},
	}
	got, err := render.RenderPipelineYAML(spec)
	if err != nil {
		t.Fatalf("RenderPipelineYAML: %v", err)
	}
	if !strings.Contains(got, "stdin: {}") {
		t.Errorf("null Config.Raw should render as empty object\n%s", got)
	}
}

func TestRenderPipelineYAML_InvalidJSON(t *testing.T) {
	spec := &rpcv1alpha1.PipelineSpec{
		Input: rpcv1alpha1.ComponentSpec{
			Type:   "generate",
			Config: runtime.RawExtension{Raw: []byte(`{not valid json`)},
		},
		Output: rpcv1alpha1.ComponentSpec{Type: "stdout"},
	}
	_, err := render.RenderPipelineYAML(spec)
	if err == nil {
		t.Fatal("expected error for invalid JSON config, got nil")
	}
	if !strings.Contains(err.Error(), "config not valid JSON") {
		t.Errorf("expected error to mention 'config not valid JSON', got: %v", err)
	}
}
