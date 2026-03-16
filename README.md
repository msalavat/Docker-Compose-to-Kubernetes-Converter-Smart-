# kompoze

Smart Docker Compose to Kubernetes converter with production-grade defaults.

[![CI](https://github.com/msalavat/Docker-Compose-to-Kubernetes-Converter-Smart-/actions/workflows/ci.yml/badge.svg)](https://github.com/msalavat/Docker-Compose-to-Kubernetes-Converter-Smart-/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/compositor/kompoze)](https://goreportcard.com/report/github.com/compositor/kompoze)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Why kompoze?

Unlike existing tools that produce bare-minimum manifests, kompoze generates **production-ready** Kubernetes resources with best practices built in.

| Feature | Kompose | kompoze |
|---------|---------|---------|
| Resource limits | - | Auto-detected defaults |
| Health probes | - | Smart defaults by port type |
| Security context | - | Restrictive by default |
| StatefulSet for databases | - | Auto-detected by image name |
| Helm chart output | - | Full chart with values.yaml |
| Kustomize output | - | base + overlays (dev/staging/prod) |
| Interactive wizard | - | TUI wizard with smart suggestions |
| Built-in validation | - | Local checks + kubeconform |
| NetworkPolicy | - | Auto-generated |
| HPA | - | Auto for web/app servers |
| PDB | - | Auto for multi-replica services |
| ServiceAccount | - | Per-service with automount disabled |
| Multi-file merge | - | `-f base.yml -f override.yml` |

## Quick Start

```bash
# Basic conversion
kompoze convert docker-compose.yml -o k8s/

# Helm chart output
kompoze convert docker-compose.yml --helm -o helm-chart/

# Kustomize with overlays
kompoze convert docker-compose.yml --kustomize -o kustomize/

# Interactive wizard
kompoze convert docker-compose.yml --wizard

# Preview without writing files
kompoze convert docker-compose.yml --dry-run
```

## Installation

### From source

```bash
git clone https://github.com/msalavat/Docker-Compose-to-Kubernetes-Converter-Smart-.git
cd Docker-Compose-to-Kubernetes-Converter-Smart-
make build
./bin/kompoze --help
```

### Binary releases

Download pre-built binaries (Linux, macOS, Windows / amd64, arm64) from [GitHub Releases](https://github.com/msalavat/Docker-Compose-to-Kubernetes-Converter-Smart-/releases).

## Usage

### Basic Conversion

```bash
kompoze convert docker-compose.yml -o k8s/
```

Generates up to 11 resource types per service:
- **Deployment** with resource limits, health probes, security context
- **StatefulSet** for databases (auto-detected by image: postgres, mysql, mongo, etc.)
- **Service** (ClusterIP for Deployments, headless for StatefulSets)
- **ConfigMap** for non-sensitive environment variables
- **Secret** references for sensitive variables (PASSWORD, SECRET, TOKEN, KEY, CREDENTIALS, PRIVATE_KEY, API_KEY)
- **PersistentVolumeClaim** for named volumes (10Gi default for databases, 1Gi for others)
- **ServiceAccount** with automount disabled
- **Ingress** for HTTP services (web servers)
- **HorizontalPodAutoscaler** for web/app servers (min=2, max=10, CPU target=70%)
- **PodDisruptionBudget** for multi-replica and database services
- **NetworkPolicy** with DNS egress and managed-by ingress rules

### Multi-file Compose

```bash
# Merge base + override (standard Docker Compose behavior)
kompoze convert -f docker-compose.yml -f docker-compose.prod.yml -o k8s/
```

Override files merge sequentially: scalar fields override, environment maps merge (override wins), ports and volumes append.

### Helm Chart Output

```bash
kompoze convert docker-compose.yml --helm -o my-chart/
```

Generates a complete Helm v3 chart:
```
my-chart/
  Chart.yaml
  values.yaml              # All configurable parameters per service
  templates/
    _helpers.tpl            # Common template functions
    NOTES.txt               # Installation notes
    <service>-deployment.yaml
    <service>-service.yaml
    <service>-configmap.yaml
    <service>-secret.yaml
    <service>-pvc.yaml
    <service>-ingress.yaml
    <service>-hpa.yaml
    <service>-pdb.yaml
    <service>-networkpolicy.yaml
    <service>-serviceaccount.yaml
```

Each service in `values.yaml` has toggles: `enabled`, `ingress.enabled`, `hpa.enabled`, `pdb.enabled`, `persistence.enabled` with full customization of image, replicas, resources, env, and secrets.

### Kustomize Output

```bash
kompoze convert docker-compose.yml --kustomize -o kustomize/
```

Generates:
```
kustomize/
  base/                    # All resources + kustomization.yaml
  overlays/
    dev/                   # Minimal resources, 1 replica
    staging/               # Medium resources, 2 replicas
    prod/                  # Full resources, HPA, PDB, Ingress
```

### Interactive Wizard

```bash
kompoze convert docker-compose.yml --wizard
```

The TUI wizard (Bubble Tea + Lipgloss) guides through 4 phases:
1. **Namespace** - custom namespace input
2. **Output format** - manifests / Helm / Kustomize
3. **Per-service config** - replicas, Ingress, TLS (cert-manager), HPA, PDB
4. **Summary** - review and confirm

Smart suggestions based on detected service type:
| Service Type | Images | Suggestions |
|-------------|--------|-------------|
| web-server | nginx, httpd, apache, traefik, caddy, haproxy, envoy | Ingress, HPA, TLS, replicas=2 |
| database | postgres, mysql, mariadb, mongo, cockroach, cassandra, elasticsearch, opensearch, couchdb, neo4j, influxdb | StatefulSet, PDB, PVC 10Gi |
| cache | redis, memcached, valkey | PDB |
| app-server | node, python, golang, java, ruby, php, dotnet, flask, django, express, spring, laravel, rails | HPA |
| generic | (default) | Basic defaults |

### Validation

```bash
# Validate with warnings
kompoze convert docker-compose.yml --validate

# Strict mode (fail on warnings)
kompoze convert docker-compose.yml --validate --strict
```

Built-in checks (3 severity levels: error, warning, info):
- Image tag validation (warns on `latest` or untagged)
- Resource limits presence
- Health probe presence
- Security context verification
- Deployment/StatefulSet structure (selectors, containers, serviceName)
- Service port consistency
- Ingress rules validation
- PVC access modes

Optional [kubeconform](https://github.com/yannh/kubeconform) integration validates against Kubernetes OpenAPI schemas when available on PATH.

### All Flags

```
kompoze convert [docker-compose.yml] [flags]

Flags:
  -f, --file stringArray     Compose files (can be specified multiple times)
  -o, --output string        Output directory (default "./k8s")
  -n, --namespace string     Kubernetes namespace (default "default")
      --app-name string      Application name (default: from compose file name)
      --helm                 Generate Helm chart
      --kustomize            Generate Kustomize structure
      --wizard               Interactive wizard mode
      --validate             Validate generated manifests
      --strict               Fail on validation warnings
      --no-probes            Skip default health probes
      --no-resources         Skip default resource limits
      --no-security          Skip default security context
      --no-network-policy    Skip NetworkPolicy generation
      --single-file          Output all manifests in single file
  -q, --quiet                Suppress non-error output
  -v, --verbose              Verbose output
      --dry-run              Print manifests to stdout, don't write files
```

## Smart Defaults

kompoze generates production-grade manifests with these defaults (all toggleable via flags):

**Resource Limits** (`--no-resources` to disable):
- Requests: 100m CPU, 128Mi memory
- Limits: 500m CPU, 256Mi memory
- Overridden by compose `deploy.resources` when specified

**Health Probes** (`--no-probes` to disable):
- HTTP ports (80, 443, 8080, 3000, 5000, 8000, 8443) get `httpGet` probes on `/`
- Other ports get `tcpSocket` probes
- Liveness: initialDelay=10s, period=15s, timeout=5s, failureThreshold=3
- Readiness: initialDelay=5s, period=15s, timeout=5s, failureThreshold=3
- Compose `healthcheck` maps to both liveness and readiness probes

**Security Context** (`--no-security` to disable):
- Pod: `runAsNonRoot: true`, `fsGroup: 1000`
- Container: `allowPrivilegeEscalation: false`, `readOnlyRootFilesystem: true`
- Capabilities: drop ALL, add only what's in compose `cap_add`
- Exception: `privileged: true` services get `runAsNonRoot: false`

**Network Policy** (`--no-network-policy` to disable):
- Egress: allows DNS (UDP/TCP port 53)
- Ingress: allows from pods with `app.kubernetes.io/managed-by: kompoze`

**Labels** (app.kubernetes.io/* convention):
- `app.kubernetes.io/name: <service-name>`
- `app.kubernetes.io/managed-by: kompoze`
- `app.kubernetes.io/part-of: <app-name>` (when `--app-name` provided)

## Supported Docker Compose Features

- Docker Compose v3.8+ (files without `version` also accepted)
- Multi-file merge (`-f base.yml -f override.yml`)
- **Services**: image, container_name, command, entrypoint, ports, volumes, environment, env_file, expose
- **Ports**: short (`8080:80`, `8080:80/udp`) and long syntax (`target`, `published`, `protocol`)
- **Volumes**: short (`name:/path`, `./path:/container:ro`) and long syntax (`type`, `source`, `target`, `read_only`)
- **Environment**: both list (`KEY=value`) and map (`KEY: value`) formats
- **Networking**: depends_on (simple list + extended with `condition`), networks (list + map with aliases)
- **Health**: healthcheck (test, interval, timeout, retries, start_period, disable)
- **Deploy**: replicas, resources (limits/reservations), restart_policy, placement, labels
- **Security**: privileged, read_only, cap_add, cap_drop, user, security_opt
- **Other**: working_dir, stdin_open, tty, sysctls, extra_hosts, dns, dns_search, logging, labels (list + map), restart
- **Top-level**: volumes, networks, secrets, configs
- **Build**: parsed but ignored with warning (K8s requires pre-built images)

## Real-World Examples

### WordPress + MySQL

```yaml
# docker-compose.yml
version: "3.8"
services:
  wordpress:
    image: wordpress:6.4
    ports:
      - "8080:80"
    environment:
      WORDPRESS_DB_HOST: mysql:3306
      WORDPRESS_DB_PASSWORD: wp_secret
    volumes:
      - wp-content:/var/www/html/wp-content
    depends_on:
      mysql:
        condition: service_healthy

  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root_secret
    volumes:
      - mysql-data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]

volumes:
  wp-content:
  mysql-data:
```

```bash
kompoze convert docker-compose.yml -o k8s/ --validate
```

**What kompoze generates:**
- `wordpress` -> Deployment + Service + Ingress + HPA + ConfigMap + Secret + ServiceAccount + NetworkPolicy
- `mysql` -> StatefulSet (auto-detected) + headless Service + Secret + PDB + ServiceAccount + NetworkPolicy
- PVCs for both volumes (1Gi for wp-content, 10Gi for mysql-data)

### Microservices Architecture

```yaml
services:
  gateway:
    image: nginx:1.25
    ports: ["80:80", "443:443"]
    depends_on: [users-api, orders-api]
    deploy:
      replicas: 2

  users-api:
    image: node:20-alpine
    ports: ["3001:3000"]
    environment:
      DB_HOST: users-db
      DB_PASSWORD: users-secret
    depends_on: [users-db]

  users-db:
    image: postgres:16
    volumes: [users-db-data:/var/lib/postgresql/data]

volumes:
  users-db-data:
```

```bash
# Generate Helm chart
kompoze convert docker-compose.yml --helm -o charts/myapp/

# Or Kustomize with environment overlays
kompoze convert docker-compose.yml --kustomize -o deploy/
```

**Smart detection:**
- `gateway` (nginx) -> web-server -> Deployment + Ingress + HPA
- `users-api` (node) -> app-server -> Deployment + HPA
- `users-db` (postgres) -> database -> StatefulSet + PDB + 10Gi PVC

## Architecture

```
                    ┌─────────────────────────────────────────────────────┐
                    │                      CLI (cobra)                     │
                    │  convert --helm --wizard --validate -o output/       │
                    └──────────┬──────────────────────────────┬────────────┘
                               │                              │
                        ┌──────▼──────┐               ┌───────▼──────┐
                        │   Parser    │               │   Wizard     │
                        │ compose.yml │               │  (Bubble Tea)│
                        │  v3.8+      │               │  TUI prompts │
                        └──────┬──────┘               └───────┬──────┘
                               │ ComposeFile                  │ WizardConfig
                               └──────────┬──────────────────┘
                                          │
                                   ┌──────▼──────┐
                                   │  Converter  │
                                   │  K8s types  │
                                   └──────┬──────┘
                                          │ ConvertResult (11 resource types)
                         ┌────────────────┼────────────────┐
                         │                │                 │
                  ┌──────▼──────┐  ┌──────▼──────┐  ┌──────▼──────┐
                  │   Output    │  │    Helm     │  │  Kustomize  │
                  │  YAML files │  │  Chart +    │  │  base +     │
                  │  or stdout  │  │  values.yaml│  │  3 overlays │
                  └──────┬──────┘  └─────────────┘  └─────────────┘
                         │
                  ┌──────▼──────┐
                  │  Validator  │
                  │ local checks│
                  │ +kubeconform│
                  └─────────────┘
```

**Resource Generation Pipeline:**

For each compose service, the converter:
1. Detects service type by image name (web-server, database, cache, app-server, generic)
2. Generates workload (Deployment or StatefulSet for databases)
3. Adds smart defaults (probes, resource limits, security context)
4. Creates supporting resources (Service, ConfigMap, Secret, PVC, ServiceAccount)
5. Auto-generates Ingress/HPA/PDB/NetworkPolicy based on service type and config

## Troubleshooting

### "unsupported compose version" error

kompoze requires docker-compose v3.8+. If your file uses an older version, update the `version` field:
```yaml
version: "3.8"  # minimum supported
```
Files without a `version` field are also accepted (Compose Spec v3+ implied).

### Sensitive variables not going to Secrets

kompoze auto-detects sensitive variables by key name patterns: `PASSWORD`, `SECRET`, `TOKEN`, `KEY`, `CREDENTIALS`, `PRIVATE_KEY`, `API_KEY`. If your variable doesn't match, rename the key or use the `--wizard` mode to manually configure Secret generation.

### StatefulSet not generated for my database

Service type is detected by image name. The image must contain one of: `postgres`, `mysql`, `mariadb`, `mongo`, `cockroach`, `cassandra`, `elasticsearch`, `opensearch`, `couchdb`, `neo4j`, `influxdb`. Custom database images need `--wizard` mode to manually select StatefulSet.

### Bind mounts produce warnings

Bind mounts (e.g., `./data:/app/data`) are not directly supported in Kubernetes. kompoze converts them to `hostPath` volumes with a warning. Use named volumes instead for portability.

### kubeconform validation not running

Install kubeconform separately:
```bash
go install github.com/yannh/kubeconform/cmd/kubeconform@latest
# or
brew install kubeconform
```
Then use `--validate` flag. If kubeconform is not on PATH, only local checks run.

### HPA generated for services that shouldn't scale

Use `--wizard` mode to disable HPA per-service, or pass `--no-probes --no-resources` to skip all smart defaults.

## Known Limitations

- Only docker-compose v3.8+ is supported (v2.x is not supported)
- `build` configurations are parsed but ignored (K8s needs pre-built images)
- `network_mode: host` is not converted
- Bind mounts map to `hostPath` (not recommended for production)
- `links` are deprecated in compose v3+ and not supported
- `extends` is not supported (merge files with `-f` instead)

## Development

```bash
make build              # Build binary
make test               # Run unit tests (with -race)
make test-integration   # Run integration tests
make lint               # Run golangci-lint
make coverage           # Generate coverage report (coverage.html)
make install            # Install to $GOPATH/bin
make clean              # Remove bin/, dist/, coverage.*
make release            # GoReleaser snapshot release
```

**Test coverage:** 141 tests across 9 packages (parser: 36, wizard: 31, converter: 19, cmd: 18, integration: 9, kustomize: 7, validator: 9, helm: 6, output: 6).

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Write tests for your changes
4. Run `make test && make lint`
5. Submit a Pull Request

## License

[MIT](LICENSE)
