Role: You are a Senior Software Architect.
Goal: Technical design and implementation of the Redpanda Connect Operator (RPC Operator).
Approach: Iterative, feature by feature.

## Requirements
Tech stack: The system is based on Kubernetes and Redpanda Connect Community (https://docs.redpanda.com/redpanda-connect/).
Document structure:
Executive Summary: Running Redpanda Connect pipelines in Kubernetes. UI-assisted monitoring and configuration of Redpanda Connect pipelines.
Diagrams: Mermaid code for the architecture.

### Context

The RPC Operator provides a flexible way to configure Redpanda Connect (RPC) pipelines and run them in Kubernetes. It gives Data Engineers a web interface to configure all Redpanda Connect pipeline components (Input, Processors, Output, etc.) visually or as YAML. The Data Engineer can then deploy a configured pipeline to a Kubernetes cluster with a simple deploy action and monitor it in the web interface.

## How it works

Redpanda Connect (built on Benthos) runs declarative YAML pipelines — `input` → `pipeline.processors` → `output`, stateless and at-least-once. Example:

```yaml
input:
  stdin: {}
pipeline:
  processors:
    - mapping: root = content().uppercase()
output:
  stdout: {}
```

The operator stores each pipeline as a Kubernetes CR and reconciles it. Two execution models: **standalone** (one dedicated pod per Pipeline CR) and **cluster/streams mode** (Pipeline placed as a stream on a shared `PipelineCluster` instance — see the E2E section). Inputs/outputs cover stdin/stdout, Kafka, HTTP, filesystems, etc.

For deeper architecture, read `docs/architecture.md`.

## Commands

All via `make` (run `make help` for the full self-documented list):
- `make build` — manifests + codegen + fmt/vet + UI build + manager binary
- `make test` — unit + envtest (auto-downloads envtest assets); `make test-ci` assumes they're present
- `make lint` / `make lint-fix` — golangci-lint
- `make run` — controller + API locally against current kubeconfig (dev flags in `run.sh`)
- `make ui-dev` — Vite dev server, proxies `/api` → localhost:8082
- `make ui-build` / `make ui-test` — build React UI into `internal/api/static/` / run vitest
- `make docs-serve` / `make docs-build` — mkdocs (build is `--strict`)
- `make docs-check-reference` — CRD reference drift (Go fields vs markdown)

## Code Layout

- `cmd/main.go` — single entrypoint; one binary serves **both** the controller manager and the HTTP/UI API.
- `api/v1alpha1/` — CRD types (Pipeline, PipelineCluster, …) + deepcopy.
- `internal/controller/` — reconcilers (operator pattern).
- `internal/api/` — HTTP + websocket handlers (`handlers_*.go`), auth/OIDC, validation; serves the built UI from `internal/api/static/`.
- `internal/{streams,render,projectroute,nats}/` — streams-mode (cluster) client, config rendering, project-route logic, NATS.
- `ui/` — React + Vite + TypeScript; builds into `internal/api/static/`.
- `charts/` — Helm chart (defaults in `charts/rpc-operator/values.yaml`; root `values.yaml` is a gitignored per-dev override).

## Gotchas

- Go module path is `github.com/insidegreen/rpc-operator-claude` even though the repo is hosted on Forgejo (`forgejo.thecloudroute.com/tom/rpc-operator`) — no GitHub repo exists. Don't "fix" the import path.
- Language: `README.md` is English; PRD/ADRs/PRPs and commit messages are German.

## Specifications

All design decisions are located in `docs/`. Always read the relevant specs before implementing:

- `docs/prd.md` — Product requirements with implementation status at the release level.
- `docs/architecture.md` — System architecture, tech stack.
- `docs/adrs/*` — Decision log in the form of ADRs.
- `docs/prps/*` — Product Requirements Prompts, feature implementation plans.
- `docs/archive/*` — Contains spec and plans that are archived and not relevant anymore. Do not read it!

## E2E Test Environment (ds9s3)

End-to-end tests run against a real cluster — unit/envtest **cannot** reach the cluster-mode HTTP
paths (nil clientset, no pod DNS, no Prometheus). Do **not** use kind. This section captures the
fixed facts so you don't have to re-investigate.

**Cluster & access**
- kubeconfig context: `ds9s3-ds9k3sm1` (remote Rancher cluster). Verify with `kubectl config current-context`.
- App namespace: `rpc-operator-poc`. Operator runs in `rpc-operator-system`.
- **CRD short-name collision:** `kubectl get pipelines` resolves to numaflow's CRD, **not** ours.
  Always use the fully-qualified name: `kubectl get pipelines.rpc.operator.io`.

**Operator / API deployment**
- Deployment `rpc-operator-system/rpc-operator`; image tag follows `main-<short-sha>`
  (e.g. `forgejo.thecloudroute.com/tom/rpc-operator:main-<sha>`). Confirm the tag matches the commit
  under test before trusting results.
- Single binary serves controller **and** API. API service: `rpc-operator-system/rpc-operator:8082`.
  Reach it with `kubectl -n rpc-operator-system port-forward svc/rpc-operator <local>:8082`
  (pick a free local port — 8082 may be held by a stale forward).
- In-cluster flags differ from `run.sh` (local dev): `--anonymous-read-enabled=false`,
  `--anonymous-logs-enabled=false` (so a Bearer token is required), and
  `--prometheus-url=http://prometheus-operated.cattle-monitoring-system.svc:9090`
  (Rancher monitoring; this Prometheus must have scraped the cluster's PodMonitor for metrics to return data).

**Auth for API calls**
- Mint a token: `kubectl -n rpc-operator-poc create token tom --duration=1h`
  (SA `tom` from `role.sh` has pipelines + pods/log read in `rpc-operator-poc`).
- HTTP endpoints: pass `-H "Authorization: Bearer $TOK"`.
- **WebSocket** endpoints (e.g. `/logs`) take the token as a **query param** `?token=$TOK`
  (browsers can't set the Authorization header on a WS upgrade), **not** a header.

**Tooling notes**
- WS client: `websocat` is installed. It needs an **open stdin** or it dies with `os error 22` —
  use `tail -f /dev/null | websocat "ws://localhost:<port>/...?token=$TOK"`. There is no `timeout`
  on macOS; background the capture and stop it with `pkill -f websocat`.

**Cluster-mode (clusterRef / streams) fixtures**
- A ready `PipelineCluster` named `pipelinecluster-sample` (2 instances:
  `pipelinecluster-sample-0/-1`, headless svc on `:4195`) is usually present.
- `config/samples/rpc_v1alpha1_pipeline_clusterref.yaml` deploys a clusterRef Pipeline as a stream.
  Placement lands in `.status` (`assignedCluster`, `assignedInstance`, `streamID == pipeline name`).
- Endpoints to exercise (cluster mode = `status.assignedInstance != ""`):
  - Logs (WS): `…/api/v1/namespaces/rpc-operator-poc/pipelines/<name>/logs?token=$TOK`
    — strict-filtered to JSON log lines whose `stream` field == pipeline name.
  - Metrics: `…/pipelines/<name>/metrics?query=<throughput|error_rate|input_rate|processor_error_rate>`
    — builds `rate(<metric>{pod="<instance>",stream="<name>"}[1m])`.
- **Stream-isolation check:** place ≥2 streams on the *same* instance pod; per-stream metrics/logs
  must differ (pod-aggregate would be identical).
- **Caveat:** a `stdout`-output pipeline writes raw untagged text to the pod log; the strict `stream`
  filter correctly drops it, so `/logs` shows only structured system lines. Use a `log` processor
  (emits JSON with the `stream` field) to see per-message data.
