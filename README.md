# kompoze

Smart Docker Compose to Kubernetes converter with production-grade defaults.

[![CI](https://github.com/compositor/kompoze/actions/workflows/ci.yml/badge.svg)](https://github.com/compositor/kompoze/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/compositor/kompoze)](https://goreportcard.com/report/github.com/compositor/kompoze)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Why kompoze?

Unlike existing tools that produce bare-minimum manifests, kompoze generates **production-ready** Kubernetes resources with best practices built in.

| Feature | Kompose | kompoze |
|---------|---------|---------|
| Resource limits | - | Auto-detected defaults |
| Health probes | - | Smart defaults by port type |
| Security context | - | Restrictive by default |
| Helm chart output | - | Full chart with values.yaml |
| Kustomize output | - | base + overlays (dev/staging/prod) |
| Interactive wizard | - | TUI wizard with smart suggestions |
| Built-in validation | - | Validates generated manifests |
| NetworkPolicy | - | Auto-generated from depends_on |
| HPA | - | Auto for non-database services |
| PDB | - | Auto for replicas > 1 |

## Quick Start

```bash
# Install
go install github.com/compositor/kompoze@latest

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

### Go install

```bash
go install github.com/compositor/kompoze@latest
```

### From source

```bash
git clone https://github.com/compositor/kompoze.git
cd kompoze
make build
./bin/kompoze --help
```

### Binary releases

Download from [GitHub Releases](https://github.com/compositor/kompoze/releases).

## Usage

### Basic Conversion

```bash
kompoze convert docker-compose.yml -o k8s/
```

Generates for each service:
- **Deployment** with resource limits, health probes, security context
- **Service** (ClusterIP) for services with ports
- **ConfigMap** for non-sensitive environment variables
- **Secret** references for sensitive variables (PASSWORD, TOKEN, etc.)
- **PVC** for named volumes
- **ServiceAccount** with automount disabled
- **Ingress** for HTTP services
- **HPA** for non-database services
- **PDB** for services with replicas > 1
- **NetworkPolicy** based on depends_on relationships

### Helm Chart Output

```bash
kompoze convert docker-compose.yml --helm -o my-chart/
```

Generates a complete Helm chart:
```
my-chart/
  Chart.yaml
  values.yaml          # All configurable parameters
  templates/
    _helpers.tpl
    NOTES.txt
    <service>-deployment.yaml
    <service>-service.yaml
    ...
```

### Kustomize Output

```bash
kompoze convert docker-compose.yml --kustomize -o kustomize/
```

Generates:
```
kustomize/
  base/                # Common resources
  overlays/
    dev/               # Minimal resources, 1 replica
    staging/           # Medium resources, 2 replicas
    prod/              # Full resources, HPA, PDB, Ingress
```

### Interactive Wizard

```bash
kompoze convert docker-compose.yml --wizard
```

The wizard analyzes your compose file and provides smart suggestions:
- Detects service types (web server, database, cache, app server)
- Suggests StatefulSet for databases
- Suggests Ingress for web servers
- Suggests HPA for application servers
- Auto-detects sensitive environment variables

### Validation

```bash
# Validate with warnings
kompoze convert docker-compose.yml --validate

# Strict mode (fail on warnings)
kompoze convert docker-compose.yml --validate --strict
```

### All Flags

```
kompoze convert [docker-compose.yml] [flags]

Flags:
  -f, --file stringArray     Compose files (can be specified multiple times)
  -o, --output string        Output directory (default "./k8s")
  -n, --namespace string     Kubernetes namespace (default "default")
      --app-name string      Application name
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
      --dry-run              Print manifests to stdout
```

## Smart Defaults

kompoze generates production-grade manifests with these defaults:

**Resource Limits:**
- Requests: 100m CPU, 128Mi memory
- Limits: 500m CPU, 256Mi memory
- Uses compose `deploy.resources` when specified

**Health Probes:**
- HTTP ports (80, 8080, 3000, etc.) get `httpGet` probes
- Other ports get `tcpSocket` probes
- Compose `healthcheck` maps to liveness + readiness probes

**Security Context:**
- Pod: `runAsNonRoot: true`, `fsGroup: 1000`
- Container: `allowPrivilegeEscalation: false`, `readOnlyRootFilesystem: true`
- Capabilities: drop ALL, add only what's in compose `cap_add`

## Supported Docker Compose Features

- docker-compose v3.8+ (files without `version` also accepted)
- Multi-file merge (`-f base.yml -f override.yml`)
- Environment variable expansion: `${VAR}`, `${VAR:-default}`, `${VAR-default}`
- Services: image, command, entrypoint, ports, volumes, environment, env_file
- Networking: depends_on (simple + extended), networks, expose
- Health: healthcheck (test, interval, timeout, retries, start_period)
- Deploy: replicas, resources (limits/reservations), restart_policy, placement
- Security: privileged, read_only, cap_add, cap_drop, user, security_opt
- Volumes: named volumes, tmpfs (bind mounts produce warnings)
- Labels: both list (`"key=value"`) and map format
- Build: parsed but ignored (K8s requires pre-built images)

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
- `wordpress` → Deployment + Service + Ingress + HPA + ConfigMap + Secret
- `mysql` → StatefulSet (auto-detected) + headless Service + Secret + PDB
- PVCs for both volumes, NetworkPolicies from `depends_on`

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
- `gateway` (nginx) → web-server → Deployment + Ingress + HPA
- `users-api` (node) → app-server → Deployment + HPA
- `users-db` (postgres) → database → StatefulSet + PDB + 10Gi PVC

### Multiple Compose Files

```bash
# Merge base + override (standard Docker Compose behavior)
kompoze convert -f docker-compose.yml -f docker-compose.prod.yml -o k8s/
```

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
                      │ ConvertResult
         ┌────────────┼────────────┐
         │            │            │
  ┌──────▼──┐  ┌──────▼──┐  ┌─────▼─────┐
  │  Output  │  │  Helm   │  │ Kustomize │
  │  YAML    │  │  Chart  │  │ base +    │
  │  files   │  │ values  │  │ overlays  │
  └──────────┘  └─────────┘  └───────────┘
         │
  ┌──────▼──────┐
  │  Validator  │
  │ local checks│
  │ +kubeconform│
  └─────────────┘
```

**Resource Generation Pipeline:**

For each compose service, the converter:
1. Detects service type by image name (web-server, database, cache, app-server)
2. Generates workload (Deployment or StatefulSet for databases)
3. Adds smart defaults (probes, resource limits, security context)
4. Creates supporting resources (Service, ConfigMap, Secret, PVC, etc.)
5. Auto-generates Ingress/HPA/PDB based on service type

## Troubleshooting

### "unsupported compose version" error

kompoze requires docker-compose v3.8+. If your file uses an older version, update the `version` field:
```yaml
version: "3.8"  # minimum supported
```
Files without a `version` field are also accepted (Compose Spec v3+ implied).

### Sensitive variables not going to Secrets

kompoze auto-detects sensitive variables by key name patterns: `PASSWORD`, `SECRET`, `TOKEN`, `KEY`, `CREDENTIALS`. If your variable doesn't match, rename the key or use the `--wizard` mode to manually configure Secret generation.

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
- `build` configurations are ignored (K8s needs pre-built images)
- `network_mode: host` is not converted
- Bind mounts map to `hostPath` (not recommended for production)
- `links` are deprecated in compose v3+ and not supported
- `extends` is not supported (merge files with `-f` instead)

## Development

```bash
make build         # Build binary
make test          # Run tests
make lint          # Run linter
make coverage      # Generate coverage report
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Write tests for your changes
4. Run `make test && make lint`
5. Submit a Pull Request

## License

[MIT](LICENSE)
