# rpc-operator Helm Chart

Kubernetes-Operator für Redpanda-Connect-Pipelines, plus eingebettete Web-UI.

> **DEV-SCOPE:** Die UI ist **ungesichert**. Nicht öffentlich exponieren.
> OIDC + RBAC kommen in v0.8.

## 5-Minuten-Quickstart

```bash
# 1. (einmalig) Namespace + Image-Pull-Secret für die Forgejo-Registry
kubectl create namespace rpc-operator-system
kubectl -n rpc-operator-system create secret docker-registry forgejo-pull \
  --docker-server=forgejo.thecloudroute.com \
  --docker-username=<user> --docker-password=<PAT>

# 2. Chart installieren (DEV: image.tag=main bis erstes Release-Tag existiert)
helm install rpc-operator ./charts/rpc-operator \
  -n rpc-operator-system \
  --set 'imagePullSecrets[0].name=forgejo-pull' \
  --set image.tag=main \
  --set examples.enabled=true

# 3. UI öffnen
kubectl -n rpc-operator-system port-forward svc/rpc-operator 8082:8082
open http://localhost:8082

# 4. Beispiel-Pipeline ansehen
kubectl -n rpc-operator-system get pipeline.rpc.operator.io
kubectl -n rpc-operator-system logs -l rpc.operator.io/pipeline=hello-world
```

## Konfiguration

| Wert | Default | Beschreibung |
|---|---|---|
| `image.repository` | `forgejo.thecloudroute.com/tom/rpc-operator` | Operator-Image |
| `image.tag` | `""` (= `Chart.appVersion`) | Tag — auf `"main"` setzen für Nightly DEV |
| `image.pullPolicy` | `IfNotPresent` | |
| `imagePullSecrets` | `[]` | Pull-Secrets für private Registry |
| `replicaCount` | `1` | Operator ist cluster-scoped — eine Instanz reicht |
| `operator.prometheusUrl` | `""` | UI-Throughput-Graph; leer = aus |
| `leaderElection.enabled` | `false` | Für `replicaCount > 1`; v0.7 default off |
| `metrics.enabled` | `false` | Operator-Self-Metrics auf `:8443` HTTPS mit Auth |
| `service.type` | `ClusterIP` | |
| `service.port` | `8082` | API + UI |
| `ingress.enabled` | `false` | UI via Ingress (siehe DEV-Warning) |
| `ingress.host` | `rpc-operator.example.com` | |
| `examples.enabled` | `false` | Installiert die hello-world Sample-Pipeline |
| `resources` | siehe `values.yaml` | Operator-Pod-Resourcen |

## CRDs

Helm installiert CRDs **nur beim ersten install**. Bei Schema-Änderungen:

```bash
kubectl apply -f charts/rpc-operator/crds/
```

## Deinstallation

```bash
helm uninstall rpc-operator -n rpc-operator-system
# CRDs werden NICHT mitgelöscht (Helm-Verhalten); falls gewünscht:
kubectl delete -f charts/rpc-operator/crds/
```

## Einschränkungen v0.7

- amd64-only Image (arm64 folgt mit eigenem Runner-Setup).
- Eine Operator-Instanz pro Cluster (cluster-scoped Reconciler).
- Keine Auth auf der UI — F20 OIDC kommt in v0.8.
- Pipeline-Pod-Image und -Resources sind aktuell hardcoded (F38).
- `leaderElection.enabled=true` schlägt fehl, weil das Chart die Lease-Role
  nicht ausliefert — als bekannte v0.7-Limitierung. Default bleibt `false`.
