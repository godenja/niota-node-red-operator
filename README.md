# niota Node-RED Operator

A Kubernetes operator that manages Node-RED instances as a first-class custom resource (`NodeRedInstance`). Each CR results in a fully configured Node-RED deployment with persistent storage, OAuth2 authentication, and a Traefik IngressRoute.

## Prerequisites

- Kubernetes 1.26+
- Helm 3.10+
- Traefik v2/v3 installed in the cluster (IngressRoute CRDs must be present)
- A GitHub Container Registry pull secret if using a private image

## Installing the chart

The Helm chart is published to GitHub Pages via the [chart-releaser](https://github.com/helm/chart-releaser-action) action.

### 1. Add the Helm repository

```bash
helm repo add niota https://godenja.github.io/niota-node-red-operator
helm repo update
```

### 2. Install the operator

```bash
helm install niota-node-red-operator niota/niota-node-red-operator \
  --namespace niota-system \
  --create-namespace
```

This installs the operator with the CRD in the `niota-system` namespace. The operator watches `NodeRedInstance` resources across all namespaces.

### Customising the installation

| Value | Default | Description |
|---|---|---|
| `installCRDs` | `true` | Install the `NodeRedInstance` CRD |
| `operator.replicaCount` | `1` | Number of operator replicas |
| `operator.image.repository` | `ghcr.io/godenja/niota-node-red-operator` | Operator image repository |
| `operator.image.tag` | *(chart appVersion)* | Operator image tag |
| `operator.leaderElect` | `true` | Enable leader election |
| `operator.resources` | see values.yaml | CPU/memory requests and limits |

Pass overrides with `--set` or a custom `values.yaml`:

```bash
helm install niota-node-red-operator niota/niota-node-red-operator \
  --namespace niota-system \
  --create-namespace \
  --set operator.replicaCount=2
```

### Upgrading

```bash
helm upgrade niota-node-red-operator niota/niota-node-red-operator \
  --namespace niota-system \
  --reuse-values
```

### Uninstalling

```bash
helm uninstall niota-node-red-operator --namespace niota-system
```

> **Note:** Uninstalling the chart does not delete existing `NodeRedInstance` resources or their managed Kubernetes objects (Deployments, PVCs, Secrets, etc.). Remove CRs manually before uninstalling if you want a clean teardown.

---

## Creating a NodeRedInstance

Once the operator is running, create a `NodeRedInstance` CR to spin up a Node-RED instance.

### Minimal example

```yaml
apiVersion: niota.io/v1alpha1
kind: NodeRedInstance
metadata:
  name: my-node-red
  namespace: default
spec:
  id: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4"   # 32-char hex ID
  domain: "node-red.example.com"
  tlsSecretName: "node-red-tls"
  oauth:
    clientId: "my-oauth-client-id"
    clientSecret: "my-oauth-client-secret"
    authUrl: "https://auth.example.com/oauth2/authorize"
    tokenUrl: "https://auth.example.com/oauth2/token"
  image:
    repository: "nodered/node-red"
    tag: "latest"
```

Apply it:

```bash
kubectl apply -f my-node-red.yaml
```

### Full example with optional fields

```yaml
apiVersion: niota.io/v1alpha1
kind: NodeRedInstance
metadata:
  name: my-node-red
  namespace: default
spec:
  # Unique 32-character hex identifier. Used as URL path prefix and as
  # the base name for all managed resources.
  id: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4"

  # Public hostname for the instance.
  domain: "node-red.example.com"

  # Name of the Kubernetes TLS Secret used by the Traefik IngressRoute.
  tlsSecretName: "node-red-tls"

  # Ingress class annotation for multi-Traefik setups (default: traefik).
  ingressClass: "traefik"

  oauth:
    clientId: "my-oauth-client-id"
    clientSecret: "my-oauth-client-secret"
    authUrl: "https://auth.example.com/oauth2/authorize"
    tokenUrl: "https://auth.example.com/oauth2/token"

  image:
    repository: "nodered/node-red"
    tag: "3.1.9"
    # Optional: name of an imagePullSecret in the same namespace.
    pullSecretName: "my-registry-secret"

  # PVC capacity for Node-RED data (default: 1Gi).
  storageSize: "5Gi"

  # StorageClass for the PVC. Omit to use the cluster default.
  storageClass: "fast-ssd"

  # Encryption key for Node-RED's credentials file. When omitted the
  # operator auto-generates a random 32-byte key on first reconciliation.
  # Provide this when migrating an existing instance to preserve credentials.
  credentialSecret: "my-32-byte-hex-credential-secret"
```

### Checking the status

```bash
# List all instances with their readiness
kubectl get noderedinstances -A

# Describe a specific instance
kubectl describe noderedinstance my-node-red -n default
```

The `Ready` column reflects the `.status.ready` field. Detailed conditions are available under `.status.conditions`.

---

## Field reference

| Field | Required | Description |
|---|---|---|
| `spec.id` | Yes | 32-character lowercase hex string, unique per instance |
| `spec.domain` | Yes | Public hostname (used in the IngressRoute) |
| `spec.tlsSecretName` | Yes | Kubernetes TLS Secret name for HTTPS |
| `spec.ingressClass` | No | Traefik ingress class annotation (default: `traefik`) |
| `spec.oauth.clientId` | Yes | OAuth2 client ID |
| `spec.oauth.clientSecret` | Yes | OAuth2 client secret |
| `spec.oauth.authUrl` | Yes | OAuth2 authorization endpoint URL |
| `spec.oauth.tokenUrl` | Yes | OAuth2 token endpoint URL |
| `spec.image.repository` | Yes | Container image repository |
| `spec.image.tag` | No | Image tag (default: `latest`) |
| `spec.image.pullSecretName` | No | Name of an imagePullSecret in the same namespace |
| `spec.storageSize` | No | PVC size (default: `1Gi`) |
| `spec.storageClass` | No | PVC storage class (cluster default when omitted) |
| `spec.credentialSecret` | No | Node-RED credentials encryption key (auto-generated when omitted) |
