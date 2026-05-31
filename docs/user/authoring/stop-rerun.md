# Stop and re-run

> **Audience:** Pipeline authors
> **Prerequisites:** [Deploying and redeploying](deploy.md)

`spec.stopped` lets you pause a pipeline without deleting its Kubernetes resources. The operator deletes the pod (and keeps it absent) while the `Pipeline` CR, ConfigMap, and PodMonitor remain in place.

## Stop a pipeline

Patch the running pipeline:

```bash
kubectl -n rpc-operator-poc patch pipelines.rpc.operator.io my-pipeline \
  --type=merge -p '{"spec":{"stopped":true}}'
```

Or edit the manifest and re-apply:

```yaml
spec:
  stopped: true
  rawYAML: |
    ...
```

The operator deletes the pod. `status.phase` becomes `Stopped`.

```bash
kubectl -n rpc-operator-poc get pipelines.rpc.operator.io
# NAME          PHASE     POD   AGE
# my-pipeline   Stopped         5m
```

## Resume a pipeline

Set `spec.stopped` back to `false`:

```bash
kubectl -n rpc-operator-poc patch pipelines.rpc.operator.io my-pipeline \
  --type=merge -p '{"spec":{"stopped":false}}'
```

The operator creates a new pod. The pipeline resumes processing from the source's committed offset (e.g. Kafka consumer group offset).

## Why use this instead of delete?

| `spec.stopped: true` | `kubectl delete` |
|---|---|
| CR, ConfigMap, PodMonitor kept | Everything removed |
| Easy to resume | Must re-apply manifest |
| Visible in `kubectl get pipelines.rpc.operator.io` | Gone from cluster |
| Retains `status` history | No history |

Use `spec.stopped` when you want to pause processing temporarily — for maintenance, cost saving, or debugging — while keeping the pipeline's definition in the cluster.

## Finished pipelines

Pipelines that run to completion (finite inputs like `generate` with `count > 0`) exit with code 0. The operator detects a clean exit and sets `status.phase` to `Stopped` (not `Failed`). The pod is not restarted. This is expected behavior.
