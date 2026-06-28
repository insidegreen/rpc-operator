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

func TestRenderPipelineYAMLForDisplay_RawYAMLStripsHTTP(t *testing.T) {
	spec := &rpcv1alpha1.PipelineSpec{
		RawYAML: "input:\n  stdin: {}\noutput:\n  stdout: {}\n",
	}

	got, err := render.RenderPipelineYAMLForDisplay(spec)
	if err != nil {
		t.Fatalf("RenderPipelineYAMLForDisplay: %v", err)
	}

	if strings.Contains(got, "http:") {
		t.Errorf("display YAML must not contain http block\n--- output ---\n%s", got)
	}
	if !strings.Contains(got, "stdin:") || !strings.Contains(got, "stdout:") {
		t.Errorf("display YAML missing user content\n--- output ---\n%s", got)
	}
}
