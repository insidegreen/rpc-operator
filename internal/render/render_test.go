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

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
	"github.com/insidegreen/rpc-operator-claude/internal/render"
)

func TestRenderPipelineYAML_EmptyRawYAML(t *testing.T) {
	_, err := render.RenderPipelineYAML(&rpcv1alpha1.PipelineSpec{})
	if err == nil {
		t.Fatal("expected error for empty rawYAML, got nil")
	}
}

func TestRenderStreamConfig_StripsHTTPBlock(t *testing.T) {
	spec := &rpcv1alpha1.PipelineSpec{
		RawYAML: "input:\n  generate:\n    mapping: 'root = \"x\"'\n    count: 1\noutput:\n  drop: {}\n",
	}
	out, err := render.RenderStreamConfig(spec)
	if err != nil {
		t.Fatalf("RenderStreamConfig: %v", err)
	}
	if strings.Contains(out, "http:") {
		t.Errorf("stream config must not contain an http block, got:\n%s", out)
	}
	if !strings.Contains(out, "generate:") || !strings.Contains(out, "drop:") {
		t.Errorf("stream config must contain the pipeline components, got:\n%s", out)
	}

	disp, err := render.RenderPipelineYAMLForDisplay(spec)
	if err != nil {
		t.Fatalf("RenderPipelineYAMLForDisplay: %v", err)
	}
	if disp != out {
		t.Errorf("RenderPipelineYAMLForDisplay should equal RenderStreamConfig output")
	}
}
