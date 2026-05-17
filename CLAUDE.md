Role: You are a Senior Software Architect.
Goal: Technical design and implementation of the Redpanda Connect Operator (RPC Operator).
Approach: Iterative, feature by feature.

## Requirements
Tech stack: The system is based on Kubernetes and Redpanda Connect Community (https://docs.redpanda.com/redpanda-connect/).
Document structure:
Executive Summary: Running Redpanda Connect pipelines in Kubernetes. UI-assisted monitoring and configuration of Redpanda Connect pipelines.
Diagrams: Mermaid code for the architecture.

### Context

The RPC Operator provides a flexible way to configure Redpanda Connect (RPC) pipelines and run them in Kubernetes. It gives Data Engineers a web interface to configure all Redpanda Connect pipeline components (Input, Processors, Output, etc.) visually or as YAML. The Data Engineer can then deploy a configured pipeline to a Kubernetes cluster with a simple deploy action and monitor it in the web interface.

## Redpanda Connect Operator – Architecture and Pipeline Configuration in Kubernetes

### 1. Core Concept

Redpanda Connect is based on Benthos — a declarative data-streaming service that solves complex data pipelines through simple, chained, stateless processing steps. Benthos guarantees at-least-once delivery without persisting messages during processing and supports a wide range of connectors for input/output. Pipeline configuration is done via a YAML file that defines the input, processor, and output. Each configuration is stored as a Kubernetes Custom Resource (CR), and one dedicated pod is started per configuration to execute the pipeline.

Sources:
- https://github.com/redpanda-data/connect
- https://github.com/redpanda-data/benthos

### 2. Pipeline Configuration

Example configuration:
```
input:
  stdin: {}
pipeline:
  processors:
    - mapping: root = content().uppercase()
output:
  stdout: {}
```

Input/Output: Supports stdin, stdout, as well as Kafka, HTTP, filesystems, etc.
Processors: Enable transformations such as mapping, filtering, aggregation, etc.

### 3. Kubernetes Integration

Custom Resource Definition (CRD): The RPC Operator uses a CRD to store pipeline configurations as Kubernetes resources. The RPC Operator watches the CRs of the CRDs and creates one pod per configuration to execute the pipeline.
Operator pattern: The RPC Operator is a Kubernetes controller that manages the lifecycle of pipelines (scaling, monitoring, error handling).
Pods: Each pipeline pod receives a Redpanda Connect configuration (Input, Processor, Output) and executes the pipeline as a self-contained unit using Redpanda Connect.

### 4. Benefits

Simple deployment: Pipelines are managed as Kubernetes resources and can be deployed/monitored via kubectl.
Scalability: Each pipeline runs in its own pod, enabling horizontal scaling.
Resilience: At-least-once delivery and backpressure mechanisms ensure reliable data processing.

## Specifications

All design decisions are located in `docs/`. Always read the relevant specs before implementing:

- `docs/prd.md` — Product requirements with implementation status at the release level.
- `docs/architecture.md` — System architecture, tech stack.
- `docs/adrs/*` — Decision log in the form of ADRs.
- `docs/prps/*` — Product Requirements Prompts, feature implementation plans.
