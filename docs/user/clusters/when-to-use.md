# When to use a PipelineCluster

> **Audience:** Both
> **Prerequisites:** [Pipeline anatomy](../authoring/anatomy.md)

The RPC Operator supports two pipeline deployment models:

| Model | How it works | Best for |
|---|---|---|
| **Pod mode** (default) | Each `Pipeline` CR gets its own Kubernetes pod | Isolated, resource-intensive, or long-running pipelines |
| **Stream mode** | A `Pipeline` with `spec.clusterRef` runs as a stream on a shared `PipelineCluster` | Many short-lived or lightweight pipelines |

## Pod mode (default)

Every `Pipeline` CR without `spec.clusterRef` gets a dedicated pod. This is the simplest model:

- **Isolation:** each pipeline is fully isolated — a crash in one does not affect others
- **Resources:** each pod consumes its own CPU/memory allocation
- **Overhead:** Kubernetes scheduling latency applies to each pipeline; cold-start takes a few seconds

Use pod mode when:
- Pipelines run continuously and are resource-intensive (e.g. high-throughput Kafka consumers)
- You need strict isolation between pipelines
- You have tens of pipelines, not hundreds

## Stream mode (PipelineCluster)

A `PipelineCluster` is a StatefulSet of Redpanda Connect instances running in [streams mode](https://docs.redpanda.com/redpanda-connect/configuration/streams_mode/about/). Each Pipeline with `spec.clusterRef` runs as a lightweight stream on one of those instances.

- **Low overhead:** no new pod per pipeline; streams start in milliseconds
- **Shared resources:** all streams on an instance share its CPU/memory
- **Density:** one instance can host dozens of streams
- **Isolation caveat:** a crash in the Redpanda Connect process affects all streams on that instance

Use stream mode when:
- You have many (tens to hundreds) of short-lived or low-throughput pipelines
- Pipelines share similar resource profiles and can tolerate shared-process isolation
- You want to reduce Kubernetes pod count and scheduling overhead

## Decision guide

```
┌─────────────────────────────────────────┐
│ Do you have > 20 concurrent pipelines?  │
└──────────────┬──────────────────────────┘
               │ Yes                No
               ▼                    ▼
┌──────────────────────────┐    Pod mode is fine
│ Are they low-throughput? │
└──────────┬───────────────┘
           │ Yes                No
           ▼                    ▼
     Stream mode           Pod mode
```
