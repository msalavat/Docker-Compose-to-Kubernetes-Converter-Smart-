package helm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/compositor/kompoze/internal/converter"
	"github.com/compositor/kompoze/internal/parser"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func testCompose() *parser.ComposeFile {
	return &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {
				Image: "nginx:1.25",
				Ports: []parser.PortConfig{
					{ContainerPort: 80, Protocol: "tcp"},
				},
				Environment: map[string]string{
					"LOG_LEVEL": "info",
				},
			},
		},
	}
}

func testResult() *converter.ConvertResult {
	replicas := int32(1)
	return &converter.ConvertResult{
		Deployments: []appsv1.Deployment{
			{
				TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
				ObjectMeta: metav1.ObjectMeta{Name: "web"},
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "web",
									Image: "nginx:1.25",
									Ports: []corev1.ContainerPort{{ContainerPort: 80}},
								},
							},
						},
					},
				},
			},
		},
		Services: []corev1.Service{
			{
				TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
				ObjectMeta: metav1.ObjectMeta{Name: "web"},
			},
		},
	}
}

func TestGenerate_CreatesChartStructure(t *testing.T) {
	dir := t.TempDir()
	compose := testCompose()
	result := testResult()

	err := Generate(compose, result, GenerateOptions{
		OutputDir: dir,
		AppName:   "myapp",
		Namespace: "default",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify Chart.yaml
	chartYAML, err := os.ReadFile(filepath.Join(dir, "Chart.yaml"))
	if err != nil {
		t.Fatalf("Chart.yaml not found: %v", err)
	}
	if !strings.Contains(string(chartYAML), "name: myapp") {
		t.Error("Chart.yaml missing app name")
	}
	if !strings.Contains(string(chartYAML), "apiVersion: v2") {
		t.Error("Chart.yaml missing apiVersion")
	}

	// Verify values.yaml
	valuesYAML, err := os.ReadFile(filepath.Join(dir, "values.yaml"))
	if err != nil {
		t.Fatalf("values.yaml not found: %v", err)
	}
	values := string(valuesYAML)
	if !strings.Contains(values, "web:") {
		t.Error("values.yaml missing web service section")
	}
	if !strings.Contains(values, "repository: nginx") {
		t.Error("values.yaml missing image repository")
	}
	if !strings.Contains(values, "tag:") {
		t.Error("values.yaml missing image tag")
	}

	// Verify _helpers.tpl
	_, err = os.ReadFile(filepath.Join(dir, "templates", "_helpers.tpl"))
	if err != nil {
		t.Fatalf("_helpers.tpl not found: %v", err)
	}

	// Verify NOTES.txt
	_, err = os.ReadFile(filepath.Join(dir, "templates", "NOTES.txt"))
	if err != nil {
		t.Fatalf("NOTES.txt not found: %v", err)
	}
}

func TestGenerate_ServiceTemplates(t *testing.T) {
	dir := t.TempDir()
	compose := testCompose()
	result := testResult()

	err := Generate(compose, result, GenerateOptions{
		OutputDir: dir,
		AppName:   "myapp",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	templatesDir := filepath.Join(dir, "templates")

	// Check deployment template
	depTpl, err := os.ReadFile(filepath.Join(templatesDir, "web-deployment.yaml"))
	if err != nil {
		t.Fatalf("web-deployment.yaml template not found: %v", err)
	}
	depContent := string(depTpl)
	if !strings.Contains(depContent, "{{") {
		t.Error("deployment template should contain Helm templating")
	}
	if !strings.Contains(depContent, ".Values.web") {
		t.Error("deployment template should reference .Values.web")
	}

	// Check service template
	svcTpl, err := os.ReadFile(filepath.Join(templatesDir, "web-service.yaml"))
	if err != nil {
		t.Fatalf("web-service.yaml template not found: %v", err)
	}
	if !strings.Contains(string(svcTpl), "{{") {
		t.Error("service template should contain Helm templating")
	}
}

func TestGenerate_DefaultAppName(t *testing.T) {
	dir := t.TempDir()
	compose := testCompose()
	result := testResult()

	err := Generate(compose, result, GenerateOptions{
		OutputDir: dir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	chartYAML, err := os.ReadFile(filepath.Join(dir, "Chart.yaml"))
	if err != nil {
		t.Fatalf("Chart.yaml not found: %v", err)
	}
	if !strings.Contains(string(chartYAML), "name: kompoze-app") {
		t.Error("expected default app name 'kompoze-app'")
	}
}

func TestBuildServiceInfos(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"api": {
				Image: "myapp/api:2.0",
				Ports: []parser.PortConfig{
					{HostPort: 8080, ContainerPort: 3000, Protocol: "tcp"},
				},
				Environment: map[string]string{
					"LOG_LEVEL":   "info",
					"DB_PASSWORD": "secret",
				},
			},
		},
	}
	result := &converter.ConvertResult{}
	infos := buildServiceInfos(compose, result)

	if len(infos) != 1 {
		t.Fatalf("expected 1 service info, got %d", len(infos))
	}

	info := infos[0]
	if info.Name != "api" {
		t.Errorf("expected name=api, got %s", info.Name)
	}
	if info.Image != "myapp/api" {
		t.Errorf("expected image=myapp/api, got %s", info.Image)
	}
	if info.ImageTag != "2.0" {
		t.Errorf("expected tag=2.0, got %s", info.ImageTag)
	}
	if len(info.Ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(info.Ports))
	}
	if info.Ports[0].ContainerPort != 3000 {
		t.Errorf("expected container port 3000, got %d", info.Ports[0].ContainerPort)
	}
}

func TestGenerateChartYAML(t *testing.T) {
	content := generateChartYAML("test-app")
	if !strings.Contains(content, "name: test-app") {
		t.Error("missing app name in Chart.yaml")
	}
	if !strings.Contains(content, "apiVersion: v2") {
		t.Error("missing apiVersion")
	}
	if !strings.Contains(content, "kompoze") {
		t.Error("missing kompoze reference in description")
	}
}

func TestGenerateHelpersTpl(t *testing.T) {
	content := generateHelpersTpl("myapp")
	if !strings.Contains(content, "myapp") {
		t.Error("helpers should reference app name")
	}
	if !strings.Contains(content, "define") {
		t.Error("helpers should contain template definitions")
	}
}
