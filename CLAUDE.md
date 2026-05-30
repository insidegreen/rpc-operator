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
- `docs/archive/*` — Contains spec and plans that are archived and not relevant anymore. Do not read it!

## E2E Test Environment (ds9s3)

End-to-end tests run against a real cluster — unit/envtest **cannot** reach the cluster-mode HTTP
paths (nil clientset, no pod DNS, no Prometheus). Do **not** use kind. This section captures the
fixed facts so you don't have to re-investigate.

**Cluster & access**
- kubeconfig context: `ds9s3-ds9k3sm1` (remote Rancher cluster). Verify with `kubectl config current-context`.
- App namespace: `rpc-operator-poc`. Operator runs in `rpc-operator-system`.
- **CRD short-name collision:** `kubectl get pipelines` resolves to numaflow's CRD, **not** ours.
  Always use the fully-qualified name: `kubectl get pipelines.rpc.operator.io`.

**Operator / API deployment**
- Deployment `rpc-operator-system/rpc-operator`; image tag follows `main-<short-sha>`
  (e.g. `forgejo.thecloudroute.com/tom/rpc-operator:main-<sha>`). Confirm the tag matches the commit
  under test before trusting results.
- Single binary serves controller **and** API. API service: `rpc-operator-system/rpc-operator:8082`.
  Reach it with `kubectl -n rpc-operator-system port-forward svc/rpc-operator <local>:8082`
  (pick a free local port — 8082 may be held by a stale forward).
- In-cluster flags differ from `run.sh` (local dev): `--anonymous-read-enabled=false`,
  `--anonymous-logs-enabled=false` (so a Bearer token is required), and
  `--prometheus-url=http://prometheus-operated.cattle-monitoring-system.svc:9090`
  (Rancher monitoring; this Prometheus must have scraped the cluster's PodMonitor for metrics to return data).

**Auth for API calls**
- Mint a token: `kubectl -n rpc-operator-poc create token tom --duration=1h`
  (SA `tom` from `role.sh` has pipelines + pods/log read in `rpc-operator-poc`).
- HTTP endpoints: pass `-H "Authorization: Bearer $TOK"`.
- **WebSocket** endpoints (e.g. `/logs`) take the token as a **query param** `?token=$TOK`
  (browsers can't set the Authorization header on a WS upgrade), **not** a header.

**Tooling notes**
- WS client: `websocat` is installed. It needs an **open stdin** or it dies with `os error 22` —
  use `tail -f /dev/null | websocat "ws://localhost:<port>/...?token=$TOK"`. There is no `timeout`
  on macOS; background the capture and stop it with `pkill -f websocat`.

**Cluster-mode (clusterRef / streams) fixtures**
- A ready `PipelineCluster` named `pipelinecluster-sample` (2 instances:
  `pipelinecluster-sample-0/-1`, headless svc on `:4195`) is usually present.
- `config/samples/rpc_v1alpha1_pipeline_clusterref.yaml` deploys a clusterRef Pipeline as a stream.
  Placement lands in `.status` (`assignedCluster`, `assignedInstance`, `streamID == pipeline name`).
- Endpoints to exercise (cluster mode = `status.assignedInstance != ""`):
  - Logs (WS): `…/api/v1/namespaces/rpc-operator-poc/pipelines/<name>/logs?token=$TOK`
    — strict-filtered to JSON log lines whose `stream` field == pipeline name.
  - Metrics: `…/pipelines/<name>/metrics?query=<throughput|error_rate|input_rate|processor_error_rate>`
    — builds `rate(<metric>{pod="<instance>",stream="<name>"}[1m])`.
- **Stream-isolation check:** place ≥2 streams on the *same* instance pod; per-stream metrics/logs
  must differ (pod-aggregate would be identical).
- **Caveat:** a `stdout`-output pipeline writes raw untagged text to the pod log; the strict `stream`
  filter correctly drops it, so `/logs` shows only structured system lines. Use a `log` processor
  (emits JSON with the `stream` field) to see per-message data.
