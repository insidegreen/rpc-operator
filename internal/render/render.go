/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package render

import (
	"fmt"

	"sigs.k8s.io/yaml"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
)

// RenderPipelineYAML produces a Redpanda Connect config from a PipelineSpec.
// Only rawYAML is supported; the rendered document enables the HTTP server on
// :4195 for liveness/readiness probes and Prometheus scraping.
func RenderPipelineYAML(spec *rpcv1alpha1.PipelineSpec) (string, error) {
	if spec.RawYAML == "" {
		return "", fmt.Errorf("rawYAML is required")
	}
	return injectHTTPConfig(spec.RawYAML)
}

// RenderStreamConfig produces the config body posted to a cluster instance's
// streams API (PUT /streams/{id}). It is the rendered pipeline minus the http
// server block, because the cluster pod already runs its own HTTP server on that port.
func RenderStreamConfig(spec *rpcv1alpha1.PipelineSpec) (string, error) {
	out, err := RenderPipelineYAML(spec)
	if err != nil {
		return "", err
	}
	return stripHTTPBlock(out)
}

// RenderPipelineYAMLForDisplay produces the user-facing YAML shown in the UI:
// the rendered config minus the operator-injected http server block. It is the
// same output as RenderStreamConfig (delegates to it) — the controller must keep
// using RenderPipelineYAML so the pod retains its liveness/readiness probes.
func RenderPipelineYAMLForDisplay(spec *rpcv1alpha1.PipelineSpec) (string, error) {
	return RenderStreamConfig(spec)
}

// stripHTTPBlock removes the top-level "http" key from a YAML document.
func stripHTTPBlock(yamlText string) (string, error) {
	var raw any
	if err := yaml.Unmarshal([]byte(yamlText), &raw); err != nil {
		return "", fmt.Errorf("invalid YAML: %w", err)
	}
	doc, ok := raw.(map[string]any)
	if !ok || doc == nil {
		return yamlText, nil
	}
	delete(doc, "http")
	out, err := yaml.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// injectHTTPConfig parses rawYAML, adds the http server block if absent, and
// re-serializes. This ensures liveness/readiness probes and Prometheus scraping work.
func injectHTTPConfig(rawYAML string) (string, error) {
	var raw any
	if err := yaml.Unmarshal([]byte(rawYAML), &raw); err != nil {
		return "", fmt.Errorf("invalid YAML: %w", err)
	}
	doc, ok := raw.(map[string]any)
	if !ok || doc == nil {
		return "", fmt.Errorf("YAML must be a mapping")
	}
	if _, exists := doc["http"]; !exists {
		doc["http"] = map[string]any{
			"enabled": true,
			"address": "0.0.0.0:4195",
		}
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

