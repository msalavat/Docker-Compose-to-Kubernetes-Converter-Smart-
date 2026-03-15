package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseComposeBytes_MinimalCompose(t *testing.T) {
	input := []byte(`
version: "3.8"
services:
  web:
    image: nginx:1.25
`)
	compose, err := ParseComposeBytes(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(compose.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(compose.Services))
	}
	svc := compose.Services["web"]
	if svc.Image != "nginx:1.25" {
		t.Errorf("expected image nginx:1.25, got %s", svc.Image)
	}
}

func TestParseComposeBytes_NoVersion(t *testing.T) {
	input := []byte(`
services:
  web:
    image: nginx
`)
	compose, err := ParseComposeBytes(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(compose.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(compose.Services))
	}
}

func TestParseComposeBytes_UnsupportedVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{"v2", `version: "2"`},
		{"v2.1", `version: "2.1"`},
		{"v3.0", `version: "3.0"`},
		{"v3.7", `version: "3.7"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(tt.version + "\nservices:\n  web:\n    image: nginx\n")
			_, err := ParseComposeBytes(input)
			if err == nil {
				t.Fatal("expected error for unsupported version")
			}
		})
	}
}

func TestParseComposeBytes_SupportedVersions(t *testing.T) {
	tests := []string{"3.8", "3.9", "3"}
	for _, v := range tests {
		t.Run("v"+v, func(t *testing.T) {
			input := []byte(`version: "` + v + `"` + "\nservices:\n  web:\n    image: nginx\n")
			_, err := ParseComposeBytes(input)
			if err != nil {
				t.Fatalf("unexpected error for version %s: %v", v, err)
			}
		})
	}
}

func TestParseComposeBytes_EmptyFile(t *testing.T) {
	_, err := ParseComposeBytes([]byte{})
	if err == nil {
		t.Fatal("expected error for empty file")
	}
}

func TestParseComposeBytes_InvalidYAML(t *testing.T) {
	_, err := ParseComposeBytes([]byte(":::invalid yaml:::"))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestParseComposeBytes_NoServices(t *testing.T) {
	input := []byte(`version: "3.8"`)
	_, err := ParseComposeBytes(input)
	if err == nil {
		t.Fatal("expected error for no services")
	}
}

func TestParsePorts(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []PortConfig
	}{
		{
			name:  "simple mapping",
			input: `ports: ["8080:80"]`,
			expected: []PortConfig{
				{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
			},
		},
		{
			name:  "container only",
			input: `ports: ["80"]`,
			expected: []PortConfig{
				{ContainerPort: 80, Protocol: "tcp"},
			},
		},
		{
			name:  "with protocol",
			input: `ports: ["8080:80/udp"]`,
			expected: []PortConfig{
				{HostPort: 8080, ContainerPort: 80, Protocol: "udp"},
			},
		},
		{
			name:  "multiple ports",
			input: `ports: ["80:80", "443:443"]`,
			expected: []PortConfig{
				{HostPort: 80, ContainerPort: 80, Protocol: "tcp"},
				{HostPort: 443, ContainerPort: 443, Protocol: "tcp"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(`version: "3.8"
services:
  web:
    image: nginx
    ` + tt.input)
			compose, err := ParseComposeBytes(input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			svc := compose.Services["web"]
			if len(svc.Ports) != len(tt.expected) {
				t.Fatalf("expected %d ports, got %d", len(tt.expected), len(svc.Ports))
			}
			for i, exp := range tt.expected {
				got := svc.Ports[i]
				if got.HostPort != exp.HostPort || got.ContainerPort != exp.ContainerPort || got.Protocol != exp.Protocol {
					t.Errorf("port[%d]: expected %+v, got %+v", i, exp, got)
				}
			}
		})
	}
}

func TestParsePortsLongSyntax(t *testing.T) {
	input := []byte(`version: "3.8"
services:
  web:
    image: nginx
    ports:
      - target: 80
        published: 8080
        protocol: tcp
`)
	compose, err := ParseComposeBytes(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	svc := compose.Services["web"]
	if len(svc.Ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(svc.Ports))
	}
	if svc.Ports[0].HostPort != 8080 || svc.Ports[0].ContainerPort != 80 {
		t.Errorf("unexpected port: %+v", svc.Ports[0])
	}
}

func TestParseVolumes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected VolumeMount
	}{
		{
			name:     "named volume",
			input:    `volumes: ["db-data:/var/lib/mysql"]`,
			expected: VolumeMount{Type: "volume", Source: "db-data", Target: "/var/lib/mysql"},
		},
		{
			name:     "bind mount",
			input:    `volumes: ["./config:/app/config"]`,
			expected: VolumeMount{Type: "bind", Source: "./config", Target: "/app/config"},
		},
		{
			name:     "read-only",
			input:    `volumes: ["./data:/app/data:ro"]`,
			expected: VolumeMount{Type: "bind", Source: "./data", Target: "/app/data", ReadOnly: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(`version: "3.8"
services:
  web:
    image: nginx
    ` + tt.input)
			compose, err := ParseComposeBytes(input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			svc := compose.Services["web"]
			if len(svc.Volumes) != 1 {
				t.Fatalf("expected 1 volume, got %d", len(svc.Volumes))
			}
			got := svc.Volumes[0]
			if got.Type != tt.expected.Type || got.Source != tt.expected.Source ||
				got.Target != tt.expected.Target || got.ReadOnly != tt.expected.ReadOnly {
				t.Errorf("expected %+v, got %+v", tt.expected, got)
			}
		})
	}
}

func TestParseVolumesLongSyntax(t *testing.T) {
	input := []byte(`version: "3.8"
services:
  web:
    image: nginx
    volumes:
      - type: tmpfs
        target: /data
`)
	compose, err := ParseComposeBytes(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	svc := compose.Services["web"]
	if len(svc.Volumes) != 1 {
		t.Fatalf("expected 1 volume, got %d", len(svc.Volumes))
	}
	if svc.Volumes[0].Type != "tmpfs" || svc.Volumes[0].Target != "/data" {
		t.Errorf("unexpected volume: %+v", svc.Volumes[0])
	}
}

func TestParseEnvironment_ListFormat(t *testing.T) {
	input := []byte(`version: "3.8"
services:
  web:
    image: nginx
    environment:
      - FOO=bar
      - BAZ=qux
`)
	compose, err := ParseComposeBytes(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	env := compose.Services["web"].Environment
	if env["FOO"] != "bar" || env["BAZ"] != "qux" {
		t.Errorf("unexpected environment: %v", env)
	}
}

func TestParseEnvironment_MapFormat(t *testing.T) {
	input := []byte(`version: "3.8"
services:
  web:
    image: nginx
    environment:
      FOO: bar
      BAZ: qux
`)
	compose, err := ParseComposeBytes(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	env := compose.Services["web"].Environment
	if env["FOO"] != "bar" || env["BAZ"] != "qux" {
		t.Errorf("unexpected environment: %v", env)
	}
}

func TestParseDependsOn_List(t *testing.T) {
	input := []byte(`version: "3.8"
services:
  web:
    image: nginx
    depends_on:
      - db
      - cache
  db:
    image: postgres
  cache:
    image: redis
`)
	compose, err := ParseComposeBytes(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	deps := compose.Services["web"].DependsOn
	if len(deps) != 2 {
		t.Fatalf("expected 2 deps, got %d", len(deps))
	}
	if deps["db"].Condition != "service_started" {
		t.Errorf("expected service_started, got %s", deps["db"].Condition)
	}
}

func TestParseDependsOn_Extended(t *testing.T) {
	input := []byte(`version: "3.8"
services:
  web:
    image: nginx
    depends_on:
      db:
        condition: service_healthy
  db:
    image: postgres
`)
	compose, err := ParseComposeBytes(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	deps := compose.Services["web"].DependsOn
	if deps["db"].Condition != "service_healthy" {
		t.Errorf("expected service_healthy, got %s", deps["db"].Condition)
	}
}

func TestParseHealthcheck(t *testing.T) {
	input := []byte(`version: "3.8"
services:
  web:
    image: nginx
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost/"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
`)
	compose, err := ParseComposeBytes(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	hc := compose.Services["web"].Healthcheck
	if hc == nil {
		t.Fatal("expected healthcheck")
	}
	if len(hc.Test) != 4 || hc.Test[0] != "CMD" {
		t.Errorf("unexpected test: %v", hc.Test)
	}
	if hc.Interval != "30s" || hc.Timeout != "10s" || hc.Retries != 3 {
		t.Errorf("unexpected healthcheck config: interval=%s timeout=%s retries=%d", hc.Interval, hc.Timeout, hc.Retries)
	}
}

func TestParseHealthcheckStringTest(t *testing.T) {
	input := []byte(`version: "3.8"
services:
  db:
    image: postgres
    healthcheck:
      test: "pg_isready -U admin"
      interval: 10s
`)
	compose, err := ParseComposeBytes(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	hc := compose.Services["db"].Healthcheck
	if hc == nil {
		t.Fatal("expected healthcheck")
	}
	if len(hc.Test) != 1 || hc.Test[0] != "pg_isready -U admin" {
		t.Errorf("unexpected test: %v", hc.Test)
	}
}

func TestParseDeployResources(t *testing.T) {
	input := []byte(`version: "3.8"
services:
  api:
    image: myapp
    deploy:
      replicas: 3
      resources:
        limits:
          cpus: "1.0"
          memory: 512M
        reservations:
          cpus: "0.25"
          memory: 128M
`)
	compose, err := ParseComposeBytes(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	deploy := compose.Services["api"].Deploy
	if deploy == nil {
		t.Fatal("expected deploy config")
	}
	if *deploy.Replicas != 3 {
		t.Errorf("expected 3 replicas, got %d", *deploy.Replicas)
	}
	if deploy.Resources == nil || deploy.Resources.Limits == nil {
		t.Fatal("expected resource limits")
	}
	if deploy.Resources.Limits.CPUs != "1.0" || deploy.Resources.Limits.Memory != "512M" {
		t.Errorf("unexpected limits: %+v", deploy.Resources.Limits)
	}
	if deploy.Resources.Reservations.CPUs != "0.25" || deploy.Resources.Reservations.Memory != "128M" {
		t.Errorf("unexpected reservations: %+v", deploy.Resources.Reservations)
	}
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "string",
			input:    `command: "echo hello"`,
			expected: []string{"echo hello"},
		},
		{
			name:     "list",
			input:    `command: ["./server", "--port", "8080"]`,
			expected: []string{"./server", "--port", "8080"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(`version: "3.8"
services:
  web:
    image: nginx
    ` + tt.input)
			compose, err := ParseComposeBytes(input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			cmd := compose.Services["web"].Command
			if len(cmd) != len(tt.expected) {
				t.Fatalf("expected %d parts, got %d", len(tt.expected), len(cmd))
			}
			for i, exp := range tt.expected {
				if cmd[i] != exp {
					t.Errorf("command[%d]: expected %q, got %q", i, exp, cmd[i])
				}
			}
		})
	}
}

func TestParseLabels(t *testing.T) {
	// Map format
	input := []byte(`version: "3.8"
services:
  web:
    image: nginx
    labels:
      app.team: backend
      app.tier: api
`)
	compose, err := ParseComposeBytes(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	labels := compose.Services["web"].Labels
	if labels["app.team"] != "backend" || labels["app.tier"] != "api" {
		t.Errorf("unexpected labels: %v", labels)
	}
}

func TestParseNetworks(t *testing.T) {
	input := []byte(`version: "3.8"
services:
  web:
    image: nginx
    networks:
      - frontend
      - backend
networks:
  frontend:
  backend:
`)
	compose, err := ParseComposeBytes(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	nets := compose.Services["web"].Networks
	if len(nets) != 2 {
		t.Fatalf("expected 2 networks, got %d", len(nets))
	}
	if _, ok := nets["frontend"]; !ok {
		t.Error("expected frontend network")
	}
}

func TestParseSecurityOptions(t *testing.T) {
	input := []byte(`version: "3.8"
services:
  web:
    image: nginx
    privileged: false
    read_only: true
    cap_add:
      - SYS_NICE
    cap_drop:
      - ALL
    user: "1000:1000"
`)
	compose, err := ParseComposeBytes(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	svc := compose.Services["web"]
	if svc.Privileged {
		t.Error("expected privileged=false")
	}
	if !svc.ReadOnly {
		t.Error("expected read_only=true")
	}
	if len(svc.CapAdd) != 1 || svc.CapAdd[0] != "SYS_NICE" {
		t.Errorf("unexpected cap_add: %v", svc.CapAdd)
	}
	if len(svc.CapDrop) != 1 || svc.CapDrop[0] != "ALL" {
		t.Errorf("unexpected cap_drop: %v", svc.CapDrop)
	}
	if svc.User != "1000:1000" {
		t.Errorf("unexpected user: %s", svc.User)
	}
}

func TestParseTopLevelVolumes(t *testing.T) {
	input := []byte(`version: "3.8"
services:
  web:
    image: nginx
volumes:
  db-data:
    driver: local
  static-files:
`)
	compose, err := ParseComposeBytes(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(compose.Volumes) != 2 {
		t.Fatalf("expected 2 volumes, got %d", len(compose.Volumes))
	}
	if compose.Volumes["db-data"].Driver != "local" {
		t.Errorf("expected driver=local, got %s", compose.Volumes["db-data"].Driver)
	}
}

func TestParseEnvVarSubstitution(t *testing.T) {
	os.Setenv("TEST_KOMPOZE_VAR", "from_env")
	defer os.Unsetenv("TEST_KOMPOZE_VAR")

	input := []byte(`version: "3.8"
services:
  web:
    image: nginx
    environment:
      - EXISTING=${TEST_KOMPOZE_VAR}
      - WITH_DEFAULT=${NONEXISTENT_VAR:-fallback}
      - NO_DEFAULT=${NONEXISTENT_VAR2}
`)
	compose, err := ParseComposeBytes(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	env := compose.Services["web"].Environment
	if env["EXISTING"] != "from_env" {
		t.Errorf("expected from_env, got %s", env["EXISTING"])
	}
	if env["WITH_DEFAULT"] != "fallback" {
		t.Errorf("expected fallback, got %s", env["WITH_DEFAULT"])
	}
}

func TestParseBuild(t *testing.T) {
	// String form
	input := []byte(`version: "3.8"
services:
  web:
    build: ./app
    image: myapp
`)
	compose, err := ParseComposeBytes(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compose.Services["web"].Build == nil || compose.Services["web"].Build.Context != "./app" {
		t.Errorf("unexpected build: %+v", compose.Services["web"].Build)
	}
}

func TestParseComposeFile_SimpleFixture(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "simple-compose.yml")
	compose, err := ParseComposeFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(compose.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(compose.Services))
	}
	if compose.Services["web"].Image != "nginx:1.25" {
		t.Errorf("unexpected web image: %s", compose.Services["web"].Image)
	}
	if compose.Services["cache"].Image != "redis:7-alpine" {
		t.Errorf("unexpected cache image: %s", compose.Services["cache"].Image)
	}
}

func TestParseComposeFile_FullFixture(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "full-compose.yml")
	compose, err := ParseComposeFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(compose.Services) != 4 {
		t.Fatalf("expected 4 services, got %d", len(compose.Services))
	}

	api := compose.Services["api"]
	if api.Image != "myapp/api:1.2.0" {
		t.Errorf("unexpected api image: %s", api.Image)
	}
	if len(api.Command) != 3 {
		t.Errorf("expected 3 command parts, got %d: %v", len(api.Command), api.Command)
	}
	if len(api.Ports) != 1 || api.Ports[0].ContainerPort != 8080 {
		t.Errorf("unexpected api ports: %v", api.Ports)
	}
	if len(api.DependsOn) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(api.DependsOn))
	}
	if api.DependsOn["db"].Condition != "service_healthy" {
		t.Errorf("expected service_healthy for db, got %s", api.DependsOn["db"].Condition)
	}
	if api.Healthcheck == nil {
		t.Error("expected healthcheck for api")
	}
	if api.Deploy == nil || *api.Deploy.Replicas != 2 {
		t.Error("expected 2 replicas for api")
	}

	db := compose.Services["db"]
	if len(db.Volumes) != 1 || db.Volumes[0].Source != "db-data" {
		t.Errorf("unexpected db volumes: %v", db.Volumes)
	}
	if len(db.CapAdd) != 1 || db.CapAdd[0] != "SYS_NICE" {
		t.Errorf("unexpected db cap_add: %v", db.CapAdd)
	}

	cache := compose.Services["cache"]
	if len(cache.Volumes) != 1 || cache.Volumes[0].Type != "tmpfs" {
		t.Errorf("unexpected cache volumes: %v", cache.Volumes)
	}

	if len(compose.Volumes) != 2 {
		t.Errorf("expected 2 top-level volumes, got %d", len(compose.Volumes))
	}
	if len(compose.Networks) != 2 {
		t.Errorf("expected 2 networks, got %d", len(compose.Networks))
	}
}

func TestParseComposeFile_WordPressFixture(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "wordpress-compose.yml")
	compose, err := ParseComposeFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(compose.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(compose.Services))
	}
	wp := compose.Services["wordpress"]
	if wp.Image != "wordpress:6.4" {
		t.Errorf("unexpected wordpress image: %s", wp.Image)
	}
	mysql := compose.Services["mysql"]
	if mysql.Healthcheck == nil {
		t.Error("expected healthcheck for mysql")
	}
}

func TestParseComposeFile_NotFound(t *testing.T) {
	_, err := ParseComposeFile("/nonexistent/docker-compose.yml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}
