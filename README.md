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

- docker-compose v3.8+
- Services: image, command, entrypoint, ports, volumes, environment, env_file
- Networking: depends_on, networks, expose
- Health: healthcheck (test, interval, timeout, retries)
- Deploy: replicas, resources (limits/reservations), restart_policy
- Security: privileged, read_only, cap_add, cap_drop, user
- Volumes: named volumes, tmpfs (bind mounts produce warnings)

## Development

```bash
make build         # Build binary
make test          # Run tests
make lint          # Run linter
make coverage      # Generate coverage report
```

## License

[MIT](LICENSE)
