# Pipeline CRD Reference

Complete reference for the Pipeline Custom Resource Definition.

## Overview

The Pipeline CRD defines a Redpanda Connect pipeline and its lifecycle on Kubernetes.

## Spec

### Metadata

- **name** (string, required): Unique name for the pipeline
- **namespace** (string): Kubernetes namespace (defaults to default)

### Status

- **phase** (string): Current lifecycle phase (Pending, Running, Failed, Succeeded)
- **conditions**: Array of pipeline conditions
- **observedGeneration**: Generation of the most recently observed spec

## Fields

### spec.rawYAML

Raw Redpanda Connect pipeline configuration as YAML string.

**Type:** string  
**Required:** yes

The rawYAML field contains the complete pipeline definition.
