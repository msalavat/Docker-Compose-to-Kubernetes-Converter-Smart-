package converter

import (
	"testing"

	"github.com/compositor/kompoze/internal/parser"
)

func intPtr(i int) *int { return &i }

func TestConvert_MinimalService(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {Image: "nginx:1.25"},
		},
	}
	result, err := Convert(compose, DefaultOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Deployments) != 1 {
		t.Fatalf("expected 1 deployment, got %d", len(result.Deployments))
	}
	d := result.Deployments[0]
	if d.Name != "web" {
		t.Errorf("expected name=web, got %s", d.Name)
	}
	if d.Spec.Template.Spec.Containers[0].Image != "nginx:1.25" {
		t.Errorf("unexpected image: %s", d.Spec.Template.Spec.Containers[0].Image)
	}
	if d.Labels["app.kubernetes.io/managed-by"] != "kompoze" {
		t.Errorf("missing managed-by label")
	}
}

func TestConvert_WithPorts(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {
				Image: "nginx",
				Ports: []parser.PortConfig{
					{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
				},
			},
		},
	}
	result, err := Convert(compose, DefaultOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Deployments) != 1 {
		t.Fatalf("expected 1 deployment, got %d", len(result.Deployments))
	}
	if len(result.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(result.Services))
	}
	svc := result.Services[0]
	if len(svc.Spec.Ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(svc.Spec.Ports))
	}
}

func TestConvert_WithEnvironment(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"api": {
				Image: "myapp",
				Environment: map[string]string{
					"LOG_LEVEL": "info",
					"DB_URL":    "postgres://localhost",
				},
			},
		},
	}
	result, err := Convert(compose, DefaultOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ConfigMaps) != 1 {
		t.Fatalf("expected 1 configmap, got %d", len(result.ConfigMaps))
	}
	cm := result.ConfigMaps[0]
	if cm.Data["LOG_LEVEL"] != "info" {
		t.Errorf("expected LOG_LEVEL=info, got %s", cm.Data["LOG_LEVEL"])
	}
}

func TestConvert_SensitiveEnvVars(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"api": {
				Image: "myapp",
				Environment: map[string]string{
					"DB_PASSWORD": "secret123",
					"API_KEY":     "abc123",
					"LOG_LEVEL":   "info",
				},
			},
		},
	}
	result, err := Convert(compose, DefaultOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ConfigMaps) != 1 {
		t.Fatalf("expected 1 configmap, got %d", len(result.ConfigMaps))
	}
	cm := result.ConfigMaps[0]
	if _, ok := cm.Data["DB_PASSWORD"]; ok {
		t.Error("DB_PASSWORD should not be in ConfigMap")
	}
	if _, ok := cm.Data["API_KEY"]; ok {
		t.Error("API_KEY should not be in ConfigMap")
	}
	if cm.Data["LOG_LEVEL"] != "info" {
		t.Errorf("expected LOG_LEVEL=info in ConfigMap")
	}
}

func TestConvert_WithHealthcheck(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {
				Image: "nginx",
				Ports: []parser.PortConfig{{ContainerPort: 80, Protocol: "tcp"}},
				Healthcheck: &parser.HealthcheckConfig{
					Test:     parser.ShellCommand{"CMD", "curl", "-f", "http://localhost/"},
					Interval: "30s",
					Timeout:  "10s",
					Retries:  3,
				},
			},
		},
	}
	result, err := Convert(compose, DefaultOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	container := result.Deployments[0].Spec.Template.Spec.Containers[0]
	if container.LivenessProbe == nil {
		t.Fatal("expected liveness probe")
	}
	if container.ReadinessProbe == nil {
		t.Fatal("expected readiness probe")
	}
	if container.LivenessProbe.Exec == nil {
		t.Fatal("expected exec probe from healthcheck")
	}
}

func TestConvert_SmartProbesHTTP(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {
				Image: "nginx",
				Ports: []parser.PortConfig{{ContainerPort: 80, Protocol: "tcp"}},
			},
		},
	}
	result, err := Convert(compose, DefaultOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	container := result.Deployments[0].Spec.Template.Spec.Containers[0]
	if container.LivenessProbe == nil || container.LivenessProbe.HTTPGet == nil {
		t.Fatal("expected HTTP liveness probe for port 80")
	}
}

func TestConvert_SmartProbesTCP(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"db": {
				Image: "postgres",
				Ports: []parser.PortConfig{{ContainerPort: 5432, Protocol: "tcp"}},
			},
		},
	}
	result, err := Convert(compose, DefaultOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	container := result.Deployments[0].Spec.Template.Spec.Containers[0]
	if container.LivenessProbe == nil || container.LivenessProbe.TCPSocket == nil {
		t.Fatal("expected TCP liveness probe for non-HTTP port")
	}
}

func TestConvert_WithResources(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"api": {
				Image: "myapp",
				Deploy: &parser.DeployConfig{
					Resources: &parser.Resources{
						Limits:       &parser.ResourceSpec{CPUs: "1.0", Memory: "512M"},
						Reservations: &parser.ResourceSpec{CPUs: "0.25", Memory: "128M"},
					},
				},
			},
		},
	}
	result, err := Convert(compose, DefaultOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	container := result.Deployments[0].Spec.Template.Spec.Containers[0]
	limCPU := container.Resources.Limits.Cpu()
	if limCPU.String() != "1" {
		t.Errorf("expected CPU limit=1, got %s", limCPU.String())
	}
}

func TestConvert_DefaultResources(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {Image: "nginx"},
		},
	}
	result, err := Convert(compose, DefaultOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	container := result.Deployments[0].Spec.Template.Spec.Containers[0]
	if container.Resources.Requests.Cpu().String() != "100m" {
		t.Errorf("expected default CPU request=100m, got %s", container.Resources.Requests.Cpu().String())
	}
}

func TestConvert_WithDependsOn(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"api": {
				Image: "myapp",
				DependsOn: parser.DependsOn{
					"db": {Condition: "service_healthy"},
				},
			},
			"db": {Image: "postgres"},
		},
	}
	result, err := Convert(compose, DefaultOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find the api deployment
	var apiDep *int
	for i, d := range result.Deployments {
		if d.Name == "api" {
			apiDep = &i
			break
		}
	}
	if apiDep == nil {
		t.Fatal("api deployment not found")
	}

	initContainers := result.Deployments[*apiDep].Spec.Template.Spec.InitContainers
	if len(initContainers) != 1 {
		t.Fatalf("expected 1 init container, got %d", len(initContainers))
	}
	if initContainers[0].Name != "wait-for-db" {
		t.Errorf("expected init container name=wait-for-db, got %s", initContainers[0].Name)
	}
}

func TestConvert_SecurityContext(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {
				Image:   "nginx",
				CapAdd:  []string{"SYS_NICE"},
				CapDrop: []string{"ALL"},
			},
		},
	}
	result, err := Convert(compose, DefaultOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	container := result.Deployments[0].Spec.Template.Spec.Containers[0]
	if container.SecurityContext == nil {
		t.Fatal("expected security context")
	}
	if *container.SecurityContext.AllowPrivilegeEscalation {
		t.Error("expected allowPrivilegeEscalation=false")
	}
	if len(container.SecurityContext.Capabilities.Add) != 1 {
		t.Errorf("expected 1 cap_add, got %d", len(container.SecurityContext.Capabilities.Add))
	}
}

func TestConvert_NoProbesWhenDisabled(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {
				Image: "nginx",
				Ports: []parser.PortConfig{{ContainerPort: 80, Protocol: "tcp"}},
			},
		},
	}
	opts := DefaultOptions()
	opts.AddProbes = false
	result, err := Convert(compose, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	container := result.Deployments[0].Spec.Template.Spec.Containers[0]
	if container.LivenessProbe != nil {
		t.Error("expected no liveness probe when disabled")
	}
}

func TestConvert_Replicas(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"api": {
				Image:  "myapp",
				Deploy: &parser.DeployConfig{Replicas: intPtr(3)},
			},
		},
	}
	result, err := Convert(compose, DefaultOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *result.Deployments[0].Spec.Replicas != 3 {
		t.Errorf("expected 3 replicas, got %d", *result.Deployments[0].Spec.Replicas)
	}
}

func TestConvert_Namespace(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {Image: "nginx"},
		},
	}
	opts := DefaultOptions()
	opts.Namespace = "production"
	result, err := Convert(compose, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Deployments[0].Namespace != "production" {
		t.Errorf("expected namespace=production, got %s", result.Deployments[0].Namespace)
	}
}

func TestConvert_NilCompose(t *testing.T) {
	_, err := Convert(nil, DefaultOptions())
	if err == nil {
		t.Fatal("expected error for nil compose")
	}
}

func TestConvert_EmptyServices(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{},
	}
	_, err := Convert(compose, DefaultOptions())
	if err == nil {
		t.Fatal("expected error for empty services")
	}
}

func TestConvert_FullComposeFile(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"api": {
				Image: "myapp/api:1.0",
				Ports: []parser.PortConfig{{HostPort: 8080, ContainerPort: 8080, Protocol: "tcp"}},
				Environment: map[string]string{
					"LOG_LEVEL":   "info",
					"DB_PASSWORD": "secret",
				},
				DependsOn: parser.DependsOn{"db": {Condition: "service_healthy"}},
				Deploy:    &parser.DeployConfig{Replicas: intPtr(2)},
			},
			"db": {
				Image: "postgres:15",
				Ports: []parser.PortConfig{{ContainerPort: 5432, Protocol: "tcp"}},
				Environment: map[string]string{
					"POSTGRES_DB":       "mydb",
					"POSTGRES_PASSWORD": "secret",
				},
			},
		},
	}
	result, err := Convert(compose, DefaultOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Deployments) != 2 {
		t.Errorf("expected 2 deployments, got %d", len(result.Deployments))
	}
	if len(result.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(result.Services))
	}
	if len(result.ConfigMaps) != 2 {
		t.Errorf("expected 2 configmaps, got %d", len(result.ConfigMaps))
	}
}
