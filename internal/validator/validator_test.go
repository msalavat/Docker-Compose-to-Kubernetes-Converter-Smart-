package validator

import (
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
