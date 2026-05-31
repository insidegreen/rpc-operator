# Operator CLI Flags

> **Audience:** Platform admins
> **Prerequisites:** none

The operator is a single binary that serves both the controller and the API. All flags have defaults suitable for Helm-based deployments; the Helm chart sets the production values. Flags are most relevant in development mode (`go run ./cmd/main.go [flags]`).

## Commonly used flags

| Flag | Default | Description |
|---|---|---|
| `--api-bind-address` | `:8082` | Address for the REST API and embedded UI. Set to `""` to disable the API (controller only). |
| `--health-probe-bind-address` | `:8081` | Address for liveness (`/healthz`) and readiness (`/readyz`) probes. |
| `--auth-enabled` | `true` | Master auth switch. `false` disables login entirely (v0.7 compatibility mode). Never combine with a public ingress. |
| `--anonymous-read-enabled` | `false` | Allow unauthenticated GETs. Requires `--auth-enabled=true`. |
| `--anonymous-logs-enabled` | `false` | Also allow unauthenticated log WebSocket. Requires `--anonymous-read-enabled`. |
| `--prometheus-url` | `""` | Prometheus base URL for the throughput graph. Empty disables the graph; everything else continues to work. |
| `--watch-namespaces` | `""` | Comma-separated namespace allowlist. Empty = cluster-wide. |
| `--leader-elect` | `false` | Enable leader election for multi-replica deployments. |

## OIDC flags

| Flag | Default | Description |
|---|---|---|
| `--oidc-issuer` | `""` | OIDC issuer URL. Non-empty enables OIDC; requires `--auth-enabled`. |
| `--oidc-client-id` | `""` | OIDC public client ID. |
| `--oidc-scopes` | `openid,email,offline_access` | Comma-separated scopes to request. |
| `--oidc-redirect-url` | `""` | OAuth 2.0 redirect URI (must be registered at the IdP). |
| `--oidc-ui-redirect-url` | `""` | Where the browser lands after the callback. Empty = `/`. |

## Metrics and TLS flags

| Flag | Default | Description |
|---|---|---|
| `--metrics-bind-address` | `0` | Address for operator self-metrics. `0` = disabled. `:8443` = HTTPS with authn/authz. |
| `--metrics-secure` | `true` | Serve metrics over HTTPS when enabled. |
| `--metrics-cert-path` | `""` | Directory containing the metrics TLS certificate. |
| `--metrics-cert-name` | `tls.crt` | Metrics TLS certificate filename. |
| `--metrics-cert-key` | `tls.key` | Metrics TLS key filename. |
| `--webhook-cert-path` | `""` | Directory containing the webhook TLS certificate. |
| `--webhook-cert-name` | `tls.crt` | Webhook TLS certificate filename. |
| `--webhook-cert-key` | `tls.key` | Webhook TLS key filename. |
| `--enable-http2` | `false` | Enable HTTP/2 for the metrics and webhook servers. |

## Other flags

| Flag | Default | Description |
|---|---|---|
| `--nats-image` | `""` | NATS container image for PipelineProject (F50). Empty = operator default (`nats:2.10-alpine`). |
| `--visual-editor-enabled` | `false` | Enable the visual pipeline editor API paths. |
| `--zap-log-level` | `debug` | Log level (`info`, `debug`, `error`). |
| `--zap-encoder` | `console` | Log encoder: `console` (human-readable) or `json`. |

## Typical dev startup

```bash
go run ./cmd/main.go \
  --auth-enabled=false \
  --watch-namespaces=rpc-operator-poc \
  --prometheus-url=http://prometheus-operated.cattle-monitoring-system.svc:9090
```

Full flag list: `go run ./cmd/main.go --help`
