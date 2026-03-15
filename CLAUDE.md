# kompoze - Docker Compose to Kubernetes Converter (Smart)

## Project Overview
CLI-инструмент на Go, конвертирующий docker-compose.yml в production-ready Kubernetes манифесты.
Конкурент Kompose (CNCF, 9K stars) с фокусом на production-grade output.

## Tech Stack
- **Language**: Go 1.22+
- **Module**: github.com/compositor/kompoze
- **CLI**: github.com/spf13/cobra
- **TUI**: github.com/charmbracelet/bubbletea + lipgloss
- **YAML**: gopkg.in/yaml.v3
- **K8s types**: k8s.io/api, k8s.io/apimachinery, sigs.k8s.io/yaml
- **Validation**: kubeconform (optional external)
- **Release**: goreleaser

## Project Structure
```
cmd/                    # Cobra CLI commands (root, convert, version)
internal/
  parser/               # Docker Compose v3.8+ parser
  converter/            # K8s manifest generators (Deployment, Service, ConfigMap, PVC, Ingress, HPA, PDB, NetworkPolicy, ServiceAccount)
  wizard/               # Bubble Tea interactive wizard
  helm/                 # Helm chart generator
  kustomize/            # Kustomize base + overlays generator
  validator/            # Manifest validation
  output/               # YAML file writer
testdata/               # Test compose files and golden outputs
```

## Key Commands
```bash
make build              # Build binary
make test               # Run unit tests
make test-integration   # Run integration tests
make lint               # Run golangci-lint
make coverage           # Generate coverage report
```

## CLI Usage
```bash
kompoze convert docker-compose.yml -o k8s/
kompoze convert --wizard docker-compose.yml
kompoze convert --helm -o helm-chart/
kompoze convert --kustomize -o kustomize/
kompoze convert --dry-run docker-compose.yml
```

## Development Guidelines
- Use official K8s Go types (k8s.io/api/*), NEVER string concatenation for YAML
- Labels follow app.kubernetes.io/* convention
- All internal packages in internal/ (unexported)
- Table-driven tests, >=80% coverage per package
- Smart defaults: probes, resource limits, security context (all toggleable via flags)
- Sensitive env vars (PASSWORD, SECRET, TOKEN, KEY, CREDENTIALS) -> Secret refs, not ConfigMaps

## Architecture Decisions
- Own compose parser (not compose-go) for full control over normalization
- Dual syntax support for ports, volumes, environment (short + long)
- Service type detection by image name (nginx->web-server, postgres->database, etc.)
- Wizard optional, defaults work for non-interactive CI/CD usage

## Current Status
- Phase: Initial setup
- Next: Prompt 1 - Project initialization and scaffolding
