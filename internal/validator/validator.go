// Package validator checks generated Kubernetes manifests for correctness and best practices.
package validator

import (
	"fmt"
	"strings"

	"github.com/compositor/kompoze/internal/converter"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ValidationError represents a single validation issue.
type ValidationError struct {
	Resource string // e.g. "deployment/api"
	Field    string // e.g. "spec.containers[0].resources"
	Severity string // "error" | "warning" | "info"
	Message  string
}

// ValidateManifests checks generated K8s resources for common issues.
func ValidateManifests(result *converter.ConvertResult) []ValidationError {
	var errors []ValidationError

	errors = append(errors, validateDeployments(result)...)
	errors = append(errors, validateStatefulSets(result)...)
	errors = append(errors, validateServices(result)...)
	errors = append(errors, validateIngresses(result)...)
	errors = append(errors, validatePVCs(result)...)

	return errors
}

// validateWorkloadContainers validates common container-level issues for any workload.
func validateWorkloadContainers(resource string, kind string, containers []corev1.Container, selector *metav1.LabelSelector) []ValidationError {
	var errors []ValidationError

	if len(containers) == 0 {
		errors = append(errors, ValidationError{
			Resource: resource,
			Field:    "spec.template.spec.containers",
			Severity: "error",
			Message:  fmt.Sprintf("%s has no containers", kind),
		})
		return errors
	}

	container := containers[0]

	// Check image tag
	if container.Image != "" {
		if strings.HasSuffix(container.Image, ":latest") || !strings.Contains(container.Image, ":") {
			errors = append(errors, ValidationError{
				Resource: resource,
				Field:    "spec.containers[0].image",
				Severity: "warning",
				Message:  fmt.Sprintf("Image '%s' uses 'latest' tag or has no tag; pin to a specific version", container.Image),
			})
		}
	}

	// Check resource limits
	if container.Resources.Limits == nil && container.Resources.Requests == nil {
		errors = append(errors, ValidationError{
			Resource: resource,
			Field:    "spec.containers[0].resources",
			Severity: "warning",
			Message:  "No resource limits or requests specified",
		})
	}

	// Check probes
	if container.LivenessProbe == nil {
		errors = append(errors, ValidationError{
			Resource: resource,
			Field:    "spec.containers[0].livenessProbe",
			Severity: "warning",
			Message:  "No liveness probe configured",
		})
	}
	if container.ReadinessProbe == nil {
		errors = append(errors, ValidationError{
			Resource: resource,
			Field:    "spec.containers[0].readinessProbe",
			Severity: "warning",
			Message:  "No readiness probe configured",
		})
	}

	// Check security context
	if container.SecurityContext == nil {
		errors = append(errors, ValidationError{
			Resource: resource,
			Field:    "spec.containers[0].securityContext",
			Severity: "info",
			Message:  "No container security context set",
		})
	}

	// Check labels
	if selector == nil || len(selector.MatchLabels) == 0 {
		errors = append(errors, ValidationError{
			Resource: resource,
			Field:    "spec.selector.matchLabels",
			Severity: "error",
			Message:  fmt.Sprintf("%s has no selector labels", kind),
		})
	}

	return errors
}

func validateDeployments(result *converter.ConvertResult) []ValidationError {
	var errors []ValidationError

	for _, d := range result.Deployments {
		resource := fmt.Sprintf("deployment/%s", d.Name)

		if d.Name == "" {
			errors = append(errors, ValidationError{
				Resource: resource,
				Field:    "metadata.name",
				Severity: "error",
				Message:  "Deployment name is empty",
			})
		}

		errors = append(errors, validateWorkloadContainers(resource, "Deployment",
			d.Spec.Template.Spec.Containers, d.Spec.Selector)...)
	}

	return errors
}

func validateStatefulSets(result *converter.ConvertResult) []ValidationError {
	var errors []ValidationError

	for _, ss := range result.StatefulSets {
		resource := fmt.Sprintf("statefulset/%s", ss.Name)

		if ss.Name == "" {
			errors = append(errors, ValidationError{
				Resource: resource,
				Field:    "metadata.name",
				Severity: "error",
				Message:  "StatefulSet name is empty",
			})
		}

		if ss.Spec.ServiceName == "" {
			errors = append(errors, ValidationError{
				Resource: resource,
				Field:    "spec.serviceName",
				Severity: "error",
				Message:  "StatefulSet has no serviceName (headless Service required)",
			})
		}

		errors = append(errors, validateWorkloadContainers(resource, "StatefulSet",
			ss.Spec.Template.Spec.Containers, ss.Spec.Selector)...)
	}

	return errors
}

func validateServices(result *converter.ConvertResult) []ValidationError {
	var errors []ValidationError

	// Build a set of deployment names for cross-reference
	deploymentNames := make(map[string]bool)
	for _, d := range result.Deployments {
		deploymentNames[d.Name] = true
	}

	for _, s := range result.Services {
		resource := fmt.Sprintf("service/%s", s.Name)

		if len(s.Spec.Ports) == 0 {
			errors = append(errors, ValidationError{
				Resource: resource,
				Field:    "spec.ports",
				Severity: "error",
				Message:  "Service has no ports defined",
			})
		}

		// Check port/targetPort consistency
		for _, port := range s.Spec.Ports {
			if port.TargetPort.IntValue() == 0 && port.TargetPort.String() == "" {
				errors = append(errors, ValidationError{
					Resource: resource,
					Field:    fmt.Sprintf("spec.ports[%s].targetPort", port.Name),
					Severity: "warning",
					Message:  "Service port has no targetPort specified",
				})
			}
		}
	}

	return errors
}

func validateIngresses(result *converter.ConvertResult) []ValidationError {
	var errors []ValidationError

	for _, ing := range result.Ingresses {
		resource := fmt.Sprintf("ingress/%s", ing.Name)

		for _, rule := range ing.Spec.Rules {
			if rule.Host == "" {
				errors = append(errors, ValidationError{
					Resource: resource,
					Field:    "spec.rules[].host",
					Severity: "warning",
					Message:  "Ingress rule has no host specified",
				})
			}
		}
	}

	return errors
}

func validatePVCs(result *converter.ConvertResult) []ValidationError {
	var errors []ValidationError

	for _, pvc := range result.PVCs {
		resource := fmt.Sprintf("pvc/%s", pvc.Name)

		if len(pvc.Spec.AccessModes) == 0 {
			errors = append(errors, ValidationError{
				Resource: resource,
				Field:    "spec.accessModes",
				Severity: "error",
				Message:  "PVC has no access modes specified",
			})
		}
	}

	return errors
}

// HasErrors returns true if any validation error has severity "error".
func HasErrors(errors []ValidationError) bool {
	for _, e := range errors {
		if e.Severity == "error" {
			return true
		}
	}
	return false
}

// HasWarnings returns true if any validation error has severity "warning".
func HasWarnings(errors []ValidationError) bool {
	for _, e := range errors {
		if e.Severity == "warning" {
			return true
		}
	}
	return false
}

// FilterBySeverity returns only errors matching the given severity.
func FilterBySeverity(errors []ValidationError, severity string) []ValidationError {
	var filtered []ValidationError
	for _, e := range errors {
		if e.Severity == severity {
			filtered = append(filtered, e)
		}
	}
	return filtered
}
