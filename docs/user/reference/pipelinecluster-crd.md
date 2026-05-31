# PipelineCluster CRD Reference

Complete reference for the PipelineCluster Custom Resource Definition.

## Overview

The PipelineCluster CRD defines a cluster of Redpanda Connect instances for distributed stream processing.

## Spec

### Metadata

- **name** (string, required): Unique name for the cluster
- **namespace** (string): Kubernetes namespace (defaults to default)

### Status

- **phase** (string): Current lifecycle phase
- **instances**: Number of running cluster instances
- **conditions**: Array of cluster conditions
- **observedGeneration**: Generation of the most recently observed spec

## Fields

### spec.replicas

Number of cluster pod instances.

**Type:** integer  
**Default:** 1

### spec.image

Container image for cluster instances.

**Type:** string  
**Default:** Latest operator-managed image
