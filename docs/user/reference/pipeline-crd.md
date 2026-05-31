# Pipeline CRD

> **API Version:** `rpc.operator.io/v1alpha1`
> **Kind:** `Pipeline`
> **Scope:** Namespaced

A `Pipeline` CR represents one Redpanda Connect data pipeline. The operator manages the full lifecycle: it creates the pod, ConfigMap, and PodMonitor when the CR is created, updates them when the spec changes, and removes them when the CR is deleted.

## Minimal Example

```yaml
apiVersion: rpc.operator.io/v1alpha1
kind: Pipeline
metadata:
  name: hello-pipeline
  namespace: rpc-operator-poc
spec:
  rawYAML: |
    input:
      generate:
        mapping: 'root = "hello"'
        interval: 5s
        count: 3
    output:
      stdout: {}
```

## Spec

### rawYAML (string, optional)

A complete Redpanda Connect configuration in YAML format. The operator mounts this into a ConfigMap that the pipeline pod reads at startup.

- **Constraint:** must be valid Redpanda Connect YAML; the operator does not validate the content — errors surface as pod startup failures
- **Mutability:** mutable; updating triggers a pod restart
- **Default:** `""`

When `rawYAML` is set, the structured fields (`input`, `processors`, `output`) are ignored.

The operator automatically injects an HTTP server block if one is absent:
```yaml
http:
  enabled: true
  address: "0.0.0.0:4195"
```

### image (string, optional)

Container image for the Redpanda Connect process.

- **Constraint:** must be a valid container image reference
- **Mutability:** mutable; updating triggers a pod restart
- **Default:** `docker.redpanda.com/redpandadata/connect:4`

### replicas (integer, optional)

Number of replicas. Currently fixed at 1; multi-replica is not yet supported.

- **Constraint:** minimum 1, maximum 1
- **Mutability:** immutable (effectively)
- **Default:** `1`

### secretRefs (array, optional)

List of Kubernetes Secret key references to inject as environment variables into the pipeline pod.

- **Constraint:** each entry requires `envVar`, `secretName`, and `key`; the Secret must exist in the same namespace
- **Mutability:** mutable; changes trigger a pod restart
- **Default:** `[]`

Each `secretRef` entry:

| Field | Type | Required | Description |
|---|---|---|---|
| `envVar` | string | yes | Env var name in the pod; must match `[A-Za-z_][A-Za-z0-9_]*` |
| `secretName` | string | yes | Name of the Kubernetes Secret |
| `key` | string | yes | Key within the Secret's `data` map |

### stopped (boolean, optional)

When `true`, the operator deletes the pipeline pod and keeps it absent. The Pipeline CR, ConfigMap, and PodMonitor remain. Set back to `false` to resume.

- **Constraint:** none
- **Mutability:** mutable; toggling triggers pod creation/deletion
- **Default:** `false`

### clusterRef (string, optional)

Name of a `PipelineCluster` in the same namespace. When set, the pipeline is deployed as a stream on the cluster instead of in its own pod.

- **Constraint:** the referenced PipelineCluster must exist in the same namespace and be in `Ready` phase
- **Mutability:** mutable (with a brief interruption — stream is removed and re-deployed)
- **Default:** `""` (pod mode)

### input (object, optional)

*Visual-editor field (not covered)* — populated by the visual editor; not stable as a hand-edited API. Use `spec.rawYAML` instead.

### processors (array, optional)

*Visual-editor field (not covered)* — populated by the visual editor; not stable as a hand-edited API. Use `spec.rawYAML` instead.

### output (object, optional)

*Visual-editor field (not covered)* — populated by the visual editor; not stable as a hand-edited API. Use `spec.rawYAML` instead.

## Status

### phase (string)

High-level lifecycle stage of the pipeline's pod.

| Value | Description |
|---|---|
| `Pending` | Pod is being created or is waiting for scheduling |
| `Running` | Pod is running and the Redpanda Connect process is active |
| `Failed` | Pod exited with a non-zero code |
| `Stopped` | Pod is absent because `spec.stopped=true` or the pipeline finished cleanly (exit 0) |

### podName (string)

Name of the pipeline pod (pod mode only). Empty in stream mode.

### assignedCluster (string)

Name of the `PipelineCluster` hosting the stream (stream mode only). Empty in pod mode.

### assignedInstance (string)

Name of the cluster pod (e.g. `etl-small-1`) that hosts the stream (stream mode only). Empty in pod mode.

### streamID (string)

The deployed stream ID, equal to the Pipeline's name (stream mode only). Empty in pod mode.

### observedGeneration (integer)

The `metadata.generation` this status reflects. Used to detect stale status.

### conditions (array)

Standard Kubernetes `Condition` array. The operator sets conditions for reconciliation events.

## Common Patterns

=== "Kafka consumer"
    ```yaml
    spec:
      rawYAML: |
        input:
          kafka_franz:
            seed_brokers: ["kafka.default.svc:9092"]
            topics: ["events"]
            consumer_group: my-consumer
        output:
          stdout: {}
    ```

=== "HTTP poller"
    ```yaml
    spec:
      rawYAML: |
        input:
          http_client:
            url: https://api.example.com/events
            verb: GET
            rate_limit: ""
        output:
          stdout: {}
    ```

=== "With secrets"
    ```yaml
    spec:
      secretRefs:
        - envVar: API_KEY
          secretName: api-credentials
          key: api-key
      rawYAML: |
        input:
          http_client:
            url: https://api.example.com/events
            headers:
              Authorization: "Bearer ${API_KEY}"
        output:
          stdout: {}
    ```
