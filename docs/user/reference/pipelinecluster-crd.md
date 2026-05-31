# PipelineCluster CRD

> **API Version:** `rpc.operator.io/v1alpha1`
> **Kind:** `PipelineCluster`
> **Scope:** Namespaced

A `PipelineCluster` CR provisions a StatefulSet of Redpanda Connect instances running in streams mode. Multiple `Pipeline` CRs can reference the cluster via `spec.clusterRef` to run as lightweight streams instead of dedicated pods.

## Minimal Example

```yaml
apiVersion: rpc.operator.io/v1alpha1
kind: PipelineCluster
metadata:
  name: etl-small
  namespace: rpc-operator-poc
spec:
  replicas: 2
```

## Spec

### replicas (integer, optional)

Number of Redpanda Connect instances (StatefulSet pods) in the cluster.

- **Constraint:** minimum 1
- **Mutability:** mutable; the StatefulSet is updated in-place (scale up/down)
- **Default:** `1`

Increasing replicas adds capacity for more streams. The operator uses round-robin assignment when deploying new streams.

### image (string, optional)

Container image for each Redpanda Connect instance.

- **Constraint:** must be a valid container image reference
- **Mutability:** mutable; triggers a StatefulSet rollout (brief interruption on each instance)
- **Default:** `docker.redpanda.com/redpandadata/connect:4`

### jsonLogging (boolean, optional)

Force structured JSON logs on every instance. Required for the operator API to filter logs by stream ID.

- **Constraint:** none
- **Mutability:** mutable; triggers a StatefulSet rollout
- **Default:** `true`

!!! warning
    Setting `jsonLogging: false` disables per-stream log filtering in the operator API. The `/logs` WebSocket endpoint will return all logs from the instance, unfiltered.

### resources (object, optional)

CPU and memory requests and limits applied to each instance container. Uses standard Kubernetes `ResourceRequirements` format.

- **Constraint:** standard Kubernetes resource quantity syntax
- **Mutability:** mutable; triggers a StatefulSet rollout
- **Default:** `{}` (no requests or limits)

Example:

```yaml
spec:
  resources:
    requests:
      cpu: "500m"
      memory: "512Mi"
    limits:
      cpu: "2"
      memory: "2Gi"
```

## Status

### phase (string)

High-level lifecycle stage of the cluster.

| Value | Description |
|---|---|
| `Pending` | StatefulSet is being created or instances are not yet ready |
| `Ready` | All desired replicas are Ready |
| `Degraded` | One or more replicas are not Ready |

### readyReplicas (integer)

Number of StatefulSet replicas currently in Ready state. Compare with `spec.replicas` to detect degraded state.

### observedGeneration (integer)

The `metadata.generation` this status reflects.

### conditions (array)

Standard Kubernetes `Condition` array for reconciliation events.

## Common Patterns

=== "Small shared cluster"
    ```yaml
    spec:
      replicas: 2
      resources:
        requests:
          cpu: "250m"
          memory: "256Mi"
        limits:
          cpu: "1"
          memory: "1Gi"
    ```

=== "Custom image"
    ```yaml
    spec:
      replicas: 3
      image: docker.redpanda.com/redpandadata/connect:4.36.0
    ```
