package validator

import (
	"fmt"
	"testing"

	"github.com/compositor/kompoze/internal/converter"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestValidateManifests_ValidResult(t *testing.T) {
	replicas := int32(1)
	result := &converter.ConvertResult{
		Deployments: []appsv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "web"},
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "web"},
					},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "web",
									Image: "nginx:1.25",
									Resources: corev1.ResourceRequirements{
										Requests: corev1.ResourceList{
											corev1.ResourceCPU: resource.MustParse("100m"),
										},
										Limits: corev1.ResourceList{
											corev1.ResourceCPU: resource.MustParse("500m"),
										},
									},
									LivenessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{Path: "/", Port: intstr.FromInt32(80)},
										},
									},
									ReadinessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{Path: "/", Port: intstr.FromInt32(80)},
										},
									},
									SecurityContext: &corev1.SecurityContext{},
								},
							},
						},
					},
				},
			},
		},
	}

	errors := ValidateManifests(result)
	if HasErrors(errors) {
		t.Errorf("expected no errors, got: %v", errors)
	}
}

func TestValidateManifests_LatestTag(t *testing.T) {
	replicas := int32(1)
	result := &converter.ConvertResult{
		Deployments: []appsv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "web"},
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "web"},
					},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "web", Image: "nginx:latest"},
							},
						},
					},
				},
			},
		},
	}

	errors := ValidateManifests(result)
	if !HasWarnings(errors) {
		t.Error("expected warning for latest tag")
	}
}

func TestValidateManifests_NoTag(t *testing.T) {
	replicas := int32(1)
	result := &converter.ConvertResult{
		Deployments: []appsv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "web"},
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "web"},
					},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "web", Image: "nginx"},
							},
						},
					},
				},
			},
		},
	}

	errors := ValidateManifests(result)
	if !HasWarnings(errors) {
		t.Error("expected warning for image without tag")
	}
}

func TestValidateManifests_NoContainers(t *testing.T) {
	replicas := int32(1)
	result := &converter.ConvertResult{
		Deployments: []appsv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "empty"},
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{},
					},
				},
			},
		},
	}

	errors := ValidateManifests(result)
	if !HasErrors(errors) {
		t.Error("expected error for deployment with no containers")
	}
}

func TestValidateManifests_IngressNoHost(t *testing.T) {
	result := &converter.ConvertResult{
		Ingresses: []networkingv1.Ingress{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "web"},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: ""},
					},
				},
			},
		},
	}

	errors := ValidateManifests(result)
	if !HasWarnings(errors) {
		t.Error("expected warning for ingress with no host")
	}
}

func TestValidateWithKubeconform_NotFound(t *testing.T) {
	replicas := int32(1)
	result := &converter.ConvertResult{
		Deployments: []appsv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "web"},
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "web"},
					},
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
	}

	// Use a lookPath that always fails
	notFoundLookPath := func(file string) (string, error) {
		return "", fmt.Errorf("not found")
	}

	errors := validateWithKubeconformInternal(result, notFoundLookPath)
	if len(errors) != 1 {
		t.Fatalf("expected 1 info error, got %d", len(errors))
	}
	if errors[0].Severity != "info" {
		t.Errorf("expected severity 'info', got '%s'", errors[0].Severity)
	}
	if errors[0].Message != "kubeconform not found, skipping schema validation" {
		t.Errorf("unexpected message: %s", errors[0].Message)
	}
}

func TestValidateWithKubeconform_EmptyResult(t *testing.T) {
	result := &converter.ConvertResult{}

	// Use a lookPath that succeeds (simulating kubeconform being available)
	// But since there are no manifests, it should return nil
	foundLookPath := func(file string) (string, error) {
		return "/usr/bin/kubeconform", nil
	}

	// This will try to run kubeconform which may not exist, but with 0 manifests
	// it should return nil before running the command.
	// Note: we can't fully test the "found" path without kubeconform installed,
	// so we test the empty manifests short-circuit.
	errors := validateWithKubeconformInternal(result, foundLookPath)
	if errors != nil {
		t.Errorf("expected nil for empty result, got %v", errors)
	}
}

func TestParseKubeconformOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLen  int
		wantSev  string
		wantKind string
	}{
		{
			name:    "valid resource",
			input:   `{"filename":"stdin","kind":"Deployment","name":"web","version":"apps/v1","status":"statusValid","msg":""}`,
			wantLen: 0,
		},
		{
			name:     "invalid resource",
			input:    `{"filename":"stdin","kind":"Deployment","name":"web","version":"apps/v1","status":"statusInvalid","msg":"spec.replicas: Invalid type"}`,
			wantLen:  1,
			wantSev:  "error",
			wantKind: "Deployment/web",
		},
		{
			name:     "error resource",
			input:    `{"filename":"stdin","kind":"Service","name":"api","version":"v1","status":"statusError","msg":"could not find schema"}`,
			wantLen:  1,
			wantSev:  "error",
			wantKind: "Service/api",
		},
		{
			name:    "mixed output",
			input:   "{\"filename\":\"stdin\",\"kind\":\"Deployment\",\"name\":\"web\",\"version\":\"apps/v1\",\"status\":\"statusValid\",\"msg\":\"\"}\n{\"filename\":\"stdin\",\"kind\":\"Service\",\"name\":\"api\",\"version\":\"v1\",\"status\":\"statusInvalid\",\"msg\":\"bad port\"}",
			wantLen: 1,
			wantSev: "error",
		},
		{
			name:    "empty output",
			input:   "",
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := parseKubeconformOutput([]byte(tt.input))
			if len(errors) != tt.wantLen {
				t.Errorf("expected %d errors, got %d: %v", tt.wantLen, len(errors), errors)
			}
			if tt.wantLen > 0 && len(errors) > 0 {
				if errors[0].Severity != tt.wantSev {
					t.Errorf("expected severity '%s', got '%s'", tt.wantSev, errors[0].Severity)
				}
				if tt.wantKind != "" && errors[0].Resource != tt.wantKind {
					t.Errorf("expected resource '%s', got '%s'", tt.wantKind, errors[0].Resource)
				}
			}
		})
	}
}

func TestFilterBySeverity(t *testing.T) {
	errors := []ValidationError{
		{Severity: "error", Message: "err1"},
		{Severity: "warning", Message: "warn1"},
		{Severity: "warning", Message: "warn2"},
		{Severity: "info", Message: "info1"},
	}

	errs := FilterBySeverity(errors, "error")
	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d", len(errs))
	}

	warns := FilterBySeverity(errors, "warning")
	if len(warns) != 2 {
		t.Errorf("expected 2 warnings, got %d", len(warns))
	}
}
