/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package controller

import (
	"encoding/json"
	"fmt"

	"sigs.k8s.io/yaml"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
)

// renderPipelineYAML produces a Redpanda Connect config from a PipelineSpec.
// The rendered document also enables the HTTP server on :4195 so liveness and
// readiness probes have something to talk to.
func renderPipelineYAML(spec *rpcv1alpha1.PipelineSpec) (string, error) {
	inputBlock, err := componentBlock(&spec.Input)
	if err != nil {
		return "", fmt.Errorf("input: %w", err)
	}
	outputBlock, err := componentBlock(&spec.Output)
	if err != nil {
		return "", fmt.Errorf("output: %w", err)
	}
	procBlocks := make([]map[string]any, 0, len(spec.Processors))
	for i := range spec.Processors {
		b, err := componentBlock(&spec.Processors[i])
		if err != nil {
			return "", fmt.Errorf("processors[%d]: %w", i, err)
		}
		procBlocks = append(procBlocks, b)
	}

	doc := map[string]any{
		"input":    inputBlock,
		"pipeline": map[string]any{"processors": procBlocks},
		"output":   outputBlock,
		"http": map[string]any{
			"enabled": true,
			"address": "0.0.0.0:4195",
		},
	}

	out, err := yaml.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func componentBlock(c *rpcv1alpha1.ComponentSpec) (map[string]any, error) {
	var cfg any = map[string]any{}
	if len(c.Config.Raw) > 0 && string(c.Config.Raw) != "null" {
		if err := json.Unmarshal(c.Config.Raw, &cfg); err != nil {
			return nil, fmt.Errorf("config not valid JSON: %w", err)
		}
	}
	return map[string]any{c.Type: cfg}, nil
}
