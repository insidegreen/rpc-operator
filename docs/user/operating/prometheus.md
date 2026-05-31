# Prometheus integration

> **Audience:** Platform admins
> **Prerequisites:** [Install via Helm](../getting-started/install.md)

The RPC Operator automatically creates a `PodMonitor` resource for every pipeline pod. Prometheus scrapes throughput and error-rate metrics from each pod's `/metrics` endpoint.

## Prerequisites

- Prometheus Operator installed (or kube-prometheus-stack)
- The `PodMonitor` CRD available in the cluster (`kubectl get crd podmonitors.monitoring.coreos.com`)
- Prometheus configured to scrape `PodMonitor` resources in pipeline namespaces

## Connect the operator to Prometheus

Set `operator.prometheusUrl` to your Prometheus base URL. This enables the throughput graph in the embedded UI:

```bash
helm upgrade rpc-operator ./charts/rpc-operator \
  -n rpc-operator-system \
  --set operator.prometheusUrl=http://prometheus-operated.cattle-monitoring-system.svc:9090
```

!!! note
    `prometheusUrl` is used only by the operator's API to proxy metric queries to the UI. The `PodMonitor` is created automatically regardless of this setting — Prometheus scrapes pipeline pods even without this value.

## What is scraped

Each pipeline pod exposes Redpanda Connect's built-in Prometheus metrics on port 4195 at `/metrics`. The metrics include:

| Metric | Description |
|---|---|
| `input_received` | Messages received by the input |
| `output_sent` | Messages successfully sent by the output |
| `output_error` | Output errors |
| `processor_error` | Processor errors |

The operator API exposes four query names via its `/metrics?query=<name>` endpoint:

| Query name | PromQL | Description |
|---|---|---|
| `throughput` | `rate(output_sent{pod="<pod>"}[1m])` | Output messages per second |
| `error_rate` | `rate(output_error{pod="<pod>"}[1m])` | Output errors per second |
| `input_rate` | `rate(input_received{pod="<pod>"}[1m])` | Input messages per second |
| `processor_error_rate` | `rate(processor_error{pod="<pod>"}[1m])` | Processor errors per second |

## PodMonitor per pipeline

The operator creates one `PodMonitor` per `Pipeline` CR in the same namespace, with the same name as the pipeline. It targets the pipeline pod by its `rpc.operator.io/pipeline` label and scrapes port 4195.

You can inspect a PodMonitor:

```bash
kubectl -n <pipeline-namespace> get podmonitors
kubectl -n <pipeline-namespace> describe podmonitor <pipeline-name>
```

## Verify metrics are flowing

```bash
# Port-forward to the pipeline pod's metrics endpoint:
kubectl -n <pipeline-namespace> port-forward pod/<pipeline-pod> 4195:4195
curl http://localhost:4195/metrics | grep output_sent
```
