# Deploy your first Pipeline

> **Audience:** Pipeline authors
> **Prerequisites:** [Install via Helm](install.md)

This page deploys a minimal pipeline that reads from a generator and writes to stdout. It is a smoke test to confirm the operator is working and your RBAC is correct.

## Create a namespace (if needed)

```bash
kubectl create namespace rpc-operator-poc
```

## Write the pipeline manifest

Save this as `hello-pipeline.yaml`:

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
        mapping: 'root = "hello from rpc-operator"'
        interval: 5s
        count: 3
    output:
      stdout: {}
```

This pipeline uses the `generate` input to produce three messages at 5-second intervals, then exits.

## Apply the manifest

```bash
kubectl apply -f hello-pipeline.yaml
```

## Watch it run

```bash
kubectl -n rpc-operator-poc get pipelines.rpc.operator.io -w
# NAME             PHASE     POD                  AGE
# hello-pipeline   Pending   hello-pipeline-pod   2s
# hello-pipeline   Running   hello-pipeline-pod   4s
# hello-pipeline   Stopped   hello-pipeline-pod   20s
```

The pipeline prints its three messages and exits with code 0. The operator sets `status.phase` to `Stopped` (not `Failed`) because the process exited cleanly.

```bash
kubectl -n rpc-operator-poc logs <pod-name>
# hello from rpc-operator
# hello from rpc-operator
# hello from rpc-operator
```

## Clean up

```bash
kubectl delete -f hello-pipeline.yaml
```

!!! tip
    To keep the `Pipeline` CR but stop the pod, set `spec.stopped: true` instead of deleting. See [Stop and re-run](../authoring/stop-rerun.md).

## Next steps

- [Verify and next steps](verify.md) — confirm your setup is production-ready
- [Pipeline anatomy](../authoring/anatomy.md) — understand `spec.rawYAML` and write real pipelines
- [Secrets via secretKeyRef](../authoring/secrets.md) — inject credentials from Kubernetes Secrets
