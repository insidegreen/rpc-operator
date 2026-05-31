# Migrating between clusters

> **Audience:** Pipeline authors
> **Prerequisites:** [Running pipelines on a cluster](cluster-ref.md)

You may need to move a stream from one `PipelineCluster` to another — for example, when decommissioning a cluster or redistributing load.

!!! warning
    Migrating a stream involves a brief interruption. The stream is removed from the source cluster and re-deployed on the target. In-flight messages at the time of removal may be reprocessed (at-least-once delivery).

## Steps

1. **Ensure the target cluster is Ready**

   ```bash
   kubectl -n rpc-operator-poc get pipelineclusters.rpc.operator.io
   # NAME          DESIRED   READY   PHASE     AGE
   # etl-small     2         2       Ready     1d
   # etl-large     4         4       Ready     2h
   ```

2. **Stop the pipeline**

   ```bash
   kubectl -n rpc-operator-poc patch pipelines.rpc.operator.io my-stream \
     --type=merge -p '{"spec":{"stopped":true}}'
   ```

   Wait for `status.phase` to be `Stopped`.

3. **Update `spec.clusterRef`**

   ```bash
   kubectl -n rpc-operator-poc patch pipelines.rpc.operator.io my-stream \
     --type=merge -p '{"spec":{"clusterRef":"etl-large"}}'
   ```

4. **Resume the pipeline**

   ```bash
   kubectl -n rpc-operator-poc patch pipelines.rpc.operator.io my-stream \
     --type=merge -p '{"spec":{"stopped":false}}'
   ```

The operator deploys the stream on the new cluster. Verify:

```bash
kubectl -n rpc-operator-poc get pipelines.rpc.operator.io my-stream -o jsonpath='{.status}'
# {"phase":"Running","assignedCluster":"etl-large","assignedInstance":"etl-large-2","streamID":"my-stream"}
```

## Migrating when the source cluster is unreachable

If the source cluster is down (all instances crashed, cluster deleted), the stop step may fail with a stream-delete error. In this case you can force the migration:

```bash
# Skip the stop step — go directly to updating clusterRef
kubectl -n rpc-operator-poc patch pipelines.rpc.operator.io my-stream \
  --type=merge -p '{"spec":{"clusterRef":"etl-large","stopped":false}}'
```

The operator will attempt to clean up the stream from the old cluster (best-effort) and deploy it on the new cluster. If the old cluster is permanently gone, the cleanup attempt will fail silently.
