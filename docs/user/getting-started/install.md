# Install the Operator

This page covers installing the RPC Operator on your Kubernetes cluster.

## Prerequisites

- Kubernetes 1.11+
- Helm 3.0+ (for Helm-based installation)
- kubectl configured for your cluster

## Using Helm

```bash
helm install rpc-operator ./charts/rpc-operator \
  -n rpc-operator-system --create-namespace
```

Refer to the Helm chart `values.yaml` in `charts/rpc-operator/` for configuration options and examples.
