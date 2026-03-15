package kustomize

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/compositor/kompoze/internal/converter"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func testResult() *converter.ConvertResult {
	replicas := int32(1)
	return &converter.ConvertResult{
		Deployments: []appsv1.Deployment{
			{
				TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
				ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"},
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "web", Image: "nginx:1.25"},
							},
						},
					},
				},
			},
		},
		Services: []corev1.Service{
			{
				TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
				ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"},
			},
		},
	}
}

func TestGenerate_CreatesDirectoryStructure(t *testing.T) {
	dir := t.TempDir()
	result := testResult()

	err := Generate(result, GenerateOptions{
		OutputDir: dir,
		AppName:   "myapp",
		Namespace: "default",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check base directory
	baseDir := filepath.Join(dir, "base")
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		t.Fatal("base directory not created")
	}

	// Check overlay directories
	for _, env := range []string{"dev", "staging", "prod"} {
		overlayDir := filepath.Join(dir, "overlays", env)
		if _, err := os.Stat(overlayDir); os.IsNotExist(err) {
			t.Fatalf("overlay directory %s not created", env)
		}
	}
}

func TestGenerate_BaseKustomization(t *testing.T) {
	dir := t.TempDir()
	result := testResult()

	err := Generate(result, GenerateOptions{OutputDir: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "base", "kustomization.yaml"))
	if err != nil {
		t.Fatalf("base kustomization.yaml not found: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "kind: Kustomization") {
		t.Error("missing Kustomization kind")
	}
	if !strings.Contains(content, "web-deployment.yaml") {
		t.Error("missing deployment resource")
	}
	if !strings.Contains(content, "web-service.yaml") {
		t.Error("missing service resource")
	}
	if !strings.Contains(content, "managed-by: kompoze") {
		t.Error("missing managed-by label")
	}
}

func TestGenerate_BaseResources(t *testing.T) {
	dir := t.TempDir()
	result := testResult()

	err := Generate(result, GenerateOptions{OutputDir: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	baseDir := filepath.Join(dir, "base")

	// Deployment file exists
	depData, err := os.ReadFile(filepath.Join(baseDir, "web-deployment.yaml"))
	if err != nil {
		t.Fatalf("deployment file not found: %v", err)
	}
	depContent := string(depData)
	if !strings.Contains(depContent, "nginx:1.25") {
		t.Error("deployment missing image")
	}
	// Namespace should be stripped in base
	if strings.Contains(depContent, "namespace: default") {
		t.Error("base resources should not have namespace set")
	}

	// Service file exists
	_, err = os.ReadFile(filepath.Join(baseDir, "web-service.yaml"))
	if err != nil {
		t.Fatalf("service file not found: %v", err)
	}
}

func TestGenerate_DevOverlay(t *testing.T) {
	dir := t.TempDir()
	result := testResult()

	err := Generate(result, GenerateOptions{OutputDir: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	devDir := filepath.Join(dir, "overlays", "dev")
	data, err := os.ReadFile(filepath.Join(devDir, "kustomization.yaml"))
	if err != nil {
		t.Fatalf("dev kustomization.yaml not found: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "namespace: dev") {
		t.Error("dev overlay missing namespace")
	}
	if !strings.Contains(content, "../../base") {
		t.Error("dev overlay missing base reference")
	}

	// Check replicas patch exists
	replicasPatch, err := os.ReadFile(filepath.Join(devDir, "web-replicas-patch.yaml"))
	if err != nil {
		t.Fatalf("replicas patch not found: %v", err)
	}
	if !strings.Contains(string(replicasPatch), "replicas: 1") {
		t.Error("dev overlay should have replicas=1")
	}
}

func TestGenerate_ProdOverlay(t *testing.T) {
	dir := t.TempDir()
	result := testResult()

	err := Generate(result, GenerateOptions{OutputDir: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prodDir := filepath.Join(dir, "overlays", "prod")
	data, err := os.ReadFile(filepath.Join(prodDir, "kustomization.yaml"))
	if err != nil {
		t.Fatalf("prod kustomization.yaml not found: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "namespace: production") {
		t.Error("prod overlay missing namespace")
	}

	// Check replicas patch for prod (should be 3 for single-replica)
	replicasPatch, err := os.ReadFile(filepath.Join(prodDir, "web-replicas-patch.yaml"))
	if err != nil {
		t.Fatalf("replicas patch not found: %v", err)
	}
	if !strings.Contains(string(replicasPatch), "replicas: 3") {
		t.Error("prod overlay should have replicas=3 for default services")
	}
}

func TestGenerateReplicasPatch(t *testing.T) {
	patch := generateReplicasPatch("api", 5)
	if !strings.Contains(patch, "name: api") {
		t.Error("missing name")
	}
	if !strings.Contains(patch, "replicas: 5") {
		t.Error("missing replicas")
	}
	if !strings.Contains(patch, "kind: Deployment") {
		t.Error("missing kind")
	}
}

func TestGenerateResourcesPatch(t *testing.T) {
	patch := generateResourcesPatch("web", "100m", "128Mi", "500m", "256Mi")
	if !strings.Contains(patch, "name: web") {
		t.Error("missing name")
	}
	if !strings.Contains(patch, "cpu: 100m") {
		t.Error("missing CPU request")
	}
	if !strings.Contains(patch, "memory: 256Mi") {
		t.Error("missing memory limit")
	}
}
