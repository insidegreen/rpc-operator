# RPC Operator

> **Primary remote:** [forgejo.thecloudroute.com/tom/rpc-operator](https://forgejo.thecloudroute.com/tom/rpc-operator) — development, issues, and PRs live here.
> [github.com/insidegreen/rpc-operator](https://github.com/insidegreen/rpc-operator) is a read-only mirror (internal docs excluded). Do not commit directly on GitHub.

Kubernetes operator for [Redpanda Connect](https://docs.redpanda.com/redpanda-connect/) pipelines.
Data engineers configure pipelines visually or as YAML through an embedded web UI
and deploy them to a Kubernetes cluster with a single click.

## Online user documentation

https://insidegreen.github.io/rpc-operator/

## Local Development

### Prerequisites

| Tool | Version |
|---|---|
| Go | ≥ 1.24 |
| Node.js | ≥ 20 (for UI build) |
| kubectl | ≥ 1.11 |
| Kubernetes cluster | ≥ 1.11 (kubeconfig configured) |

### Quickstart

```bash
# 1. Install the CRDs into the cluster (once)
make install

# 2. Build the UI
make ui-build

# 3. Start the operator — uses the current kubeconfig context
go run ./cmd/main.go
```

The operator connects to the active kubeconfig context.
The web UI is then available at **http://localhost:8082**.

```bash
# Inspect / switch context
kubectl config current-context
kubectl config use-context <context-name>
```

### Operator CLI Flags

The operator is started with `go run ./cmd/main.go [flags]`. All flags have
sensible defaults — the ones below are the most commonly relevant in a dev setup.

| Flag | Default | Purpose |
|---|---|---|
| `--api-bind-address` | `:8082` | Address for the REST API + embedded UI. Empty (`""`) disables the API server (operator loop only). |
| `--health-probe-bind-address` | `:8081` | Liveness/readiness probes (`/healthz`, `/readyz`). |
| `--auth-enabled` | `true` | F43 master switch. `--auth-enabled=false` reproduces v0.7 behavior (no login, all requests via operator SA). Typically set to `false` for hot-reload dev. |
| `--prometheus-url` | _empty_ | Prometheus base URL for the throughput graph (F15). Empty disables only the graph; everything else still works. Example: `--prometheus-url=http://prometheus-operated.cattle-monitoring-system.svc:9090` |
| `--watch-namespaces` | _empty_ | F21 allowlist as a comma-separated list. Empty = cluster-wide (sees all pipelines). Example: `--watch-namespaces=rpc-operator-poc,default` |
| `--leader-elect` | `false` | For multi-replica setups. Leave off in single-pod dev. |
| `--metrics-bind-address` | `0` | Operator self-metrics (controller-runtime). `0` = off (default; the per-pipeline PodMonitor from F36 is unaffected). `:8443` enables HTTPS with authn/authz. |
| `--zap-log-level` | `debug` | Log level (`info`, `debug`, `error`); `opts.Development=true` in code sets the default to `debug`. |
| `--zap-encoder` | `console` | `console` (human-readable) or `json` (for structured logs). |

Typical dev startup with all relevant flags:

```bash
go run ./cmd/main.go \
  --auth-enabled=false \
  --watch-namespaces=rpc-operator-poc \
  --prometheus-url=http://prometheus-operated.cattle-monitoring-system.svc:9090
```

> **Production flags:** `--metrics-secure`, `--metrics-cert-*`, `--webhook-cert-*`, `--enable-http2` are intended for in-cluster Helm deployments and are normally not needed in dev. Full list: `go run ./cmd/main.go --help`.

### UI Development with Hot Reload

For frontend changes a separate Vite dev server is enough — no operator restart
required. Vite automatically proxies `/api` requests to `:8082`.

```bash
# Terminal 1 — operator
go run ./cmd/main.go

# Terminal 2 — Vite dev server (hot reload)
make ui-dev
# → http://localhost:5173
```

### Running Tests

```bash
# All Go tests
make test

# A single package
go test ./internal/render/...

# Linter
make lint

# TypeScript type check
cd ui && npx tsc --noEmit
```

### Test a Pipeline Manually

```bash
# Show deployed pipelines
kubectl get pipeline.rpc.operator.io -n rpc-operator-poc

# Pipeline pods
kubectl get pods -n rpc-operator-poc

# Pipeline logs
kubectl logs -n rpc-operator-poc <pod-name>

# Delete a pipeline
kubectl delete pipeline.rpc.operator.io <name> -n rpc-operator-poc
```

### Useful Make Targets

```bash
make help          # All targets with descriptions
make ui-build      # Build the React UI (→ internal/api/static/)
make ui-dev        # Start the Vite dev server
make build         # Build the Go binary including the UI (→ bin/manager)
make test          # Run Go tests
make lint          # golangci-lint
make install       # Install CRDs into the cluster
make uninstall     # Remove CRDs from the cluster
```

---

## User Documentation

Installation, configuration, authentication modes (token / OIDC), namespace
allowlist, Prometheus integration, pipeline authoring, PipelineCluster,
operations, and CRD reference live in the user documentation:

**https://insidegreen.github.io/rpc-operator/**

---

## Container Image

The operator image (manager + UI in a single binary) is built and pushed to the
Forgejo registry automatically by Forgejo Actions on every release tag and
every push to `main`.

**Image:** `forgejo.thecloudroute.com/tom/rpc-operator`

**Tags:**

| Tag             | Meaning                                       |
|-----------------|-----------------------------------------------|
| `vX.Y.Z`        | Release build from a Git tag                  |
| `latest`        | Most recent release tag                       |
| `main`          | Latest commit on the `main` branch            |
| `main-<sha7>`   | Specific main commit (for bisect / pinning)   |

**Architectures:** `linux/amd64` (arm64 will follow once an arm64 Forgejo runner is available).

**Pull:**

```bash
docker pull forgejo.thecloudroute.com/tom/rpc-operator:latest
```

**Build locally** (maintainers, with a Docker daemon):

```bash
make docker-build IMG=forgejo.thecloudroute.com/tom/rpc-operator:dev
```

> **CI build:** Forgejo Actions builds the image with
> [Kaniko](https://github.com/GoogleContainerTools/kaniko) without a Docker
> daemon (`.forgejo/workflows/image.yml`). Multi-arch is not enabled because
> Kaniko is a single-arch builder and currently only an amd64 runner is
> available.

---

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
