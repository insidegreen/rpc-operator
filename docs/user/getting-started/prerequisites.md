# Prerequisites

> **Audience:** Both
> **Prerequisites:** none

Before you install the RPC Operator or deploy your first pipeline, make sure the following are in place.

## For platform administrators (installing the operator)

| Requirement | Version | Notes |
|---|---|---|
| Kubernetes cluster | ≥ 1.25 | Local clusters (kind, k3d) work for development |
| Helm | ≥ 3.10 | `helm version` |
| kubectl | ≥ 1.25 | Configured with a kubeconfig pointing at your cluster |
| Cluster-admin access | — | Required for CRD installation and namespace creation |

If you want to use **Prometheus metrics**, you also need a Prometheus instance with the `PodMonitor` CRD available (shipped with the Prometheus Operator or kube-prometheus-stack).

If you want **OIDC SSO**, your Kubernetes apiserver must be configured with `--oidc-issuer-url` and `--oidc-client-id` flags matching your identity provider before you install the operator. See [OIDC SSO](../operating/oidc.md) for details.

## For pipeline authors (deploying pipelines)

| Requirement | Version | Notes |
|---|---|---|
| kubectl | ≥ 1.25 | Configured with a kubeconfig for the target cluster |
| Access to the target namespace | — | `get`, `create`, `update`, `delete` on `pipelines.rpc.operator.io` |
| Redpanda Connect knowledge | — | You write pipeline YAML; see [Redpanda Connect docs](https://docs.redpanda.com/redpanda-connect/) |

!!! note
    The operator must already be installed in your cluster. Ask your platform admin if you are unsure.
