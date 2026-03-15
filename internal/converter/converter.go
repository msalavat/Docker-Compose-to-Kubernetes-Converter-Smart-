package converter

import (
	"fmt"
	"sort"

	"github.com/compositor/kompoze/internal/parser"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// ConvertOptions holds configuration for the conversion process.
type ConvertOptions struct {
	OutputDir    string
	Namespace    string
	AppName      string
	AddProbes    bool
	AddResources bool
	AddSecurity  bool
	SingleFile   bool
}

// DefaultOptions returns ConvertOptions with production-grade defaults.
func DefaultOptions() ConvertOptions {
	return ConvertOptions{
		OutputDir:    "./k8s",
		Namespace:    "default",
		AddProbes:    true,
		AddResources: true,
		AddSecurity:  true,
	}
}

// ConvertResult holds all generated Kubernetes resources.
type ConvertResult struct {
	Deployments []appsv1.Deployment
	Services    []corev1.Service
	ConfigMaps  []corev1.ConfigMap
	PVCs        []corev1.PersistentVolumeClaim
}

// Convert transforms a parsed ComposeFile into Kubernetes resources.
func Convert(compose *parser.ComposeFile, opts ConvertOptions) (*ConvertResult, error) {
	if compose == nil {
		return nil, fmt.Errorf("compose file is nil")
	}
	if len(compose.Services) == 0 {
		return nil, fmt.Errorf("no services to convert")
	}

	result := &ConvertResult{}

	// Sort service names for deterministic output
	names := make([]string, 0, len(compose.Services))
	for name := range compose.Services {
		names = append(names, name)
	}
	sort.Strings(names)

	var warnings []string

	for _, name := range names {
		svc := compose.Services[name]

		// Generate volumes (PVCs, pod volumes, mounts)
		volResult := generateVolumes(name, &svc, compose, opts)
		result.PVCs = append(result.PVCs, volResult.PVCs...)
		warnings = append(warnings, volResult.Warnings...)

		// Generate Deployment (with volume mounts)
		deployment := generateDeployment(name, &svc, opts)
		if len(volResult.PodVolumes) > 0 {
			deployment.Spec.Template.Spec.Volumes = volResult.PodVolumes
			deployment.Spec.Template.Spec.Containers[0].VolumeMounts = volResult.VolumeMounts
		}
		result.Deployments = append(result.Deployments, deployment)

		// Generate Service (if ports defined)
		if len(svc.Ports) > 0 {
			k8sSvc := generateService(name, &svc, opts)
			result.Services = append(result.Services, k8sSvc)
		}

		// Generate ConfigMap (if environment defined with non-secret vars)
		if len(svc.Environment) > 0 {
			configMap, secretKeys := generateConfigMap(name, &svc, opts)
			if len(configMap.Data) > 0 {
				result.ConfigMaps = append(result.ConfigMaps, configMap)
			}
			_ = secretKeys // will be used for Secret generation later
		}
	}
	_ = warnings // will be reported by CLI

	return result, nil
}

// standardLabels returns the standard set of labels for a resource.
func standardLabels(serviceName string, appName string) map[string]string {
	labels := map[string]string{
		"app.kubernetes.io/name":       serviceName,
		"app.kubernetes.io/managed-by": "kompoze",
	}
	if appName != "" {
		labels["app.kubernetes.io/part-of"] = appName
	}
	return labels
}

// selectorLabels returns labels used for pod selection.
func selectorLabels(serviceName string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name": serviceName,
	}
}
