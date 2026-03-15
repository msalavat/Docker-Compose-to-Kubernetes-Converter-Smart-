package converter

import (
	"fmt"
	"sort"

	"github.com/compositor/kompoze/internal/parser"
	"github.com/compositor/kompoze/internal/wizard"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
)

// ConvertOptions holds configuration for the conversion process.
type ConvertOptions struct {
	OutputDir        string
	Namespace        string
	AppName          string
	AddProbes        bool
	AddResources     bool
	AddSecurity      bool
	SingleFile       bool
	AddNetworkPolicy bool
	// WizardOverrides holds per-service overrides from the interactive wizard.
	// If nil, smart defaults from service type detection are used.
	WizardOverrides map[string]wizard.ServiceWizardConfig
}

// DefaultOptions returns ConvertOptions with production-grade defaults.
func DefaultOptions() ConvertOptions {
	return ConvertOptions{
		OutputDir:        "./k8s",
		Namespace:        "default",
		AddProbes:        true,
		AddResources:     true,
		AddSecurity:      true,
		AddNetworkPolicy: true,
	}
}

// ConvertResult holds all generated Kubernetes resources.
type ConvertResult struct {
	Deployments     []appsv1.Deployment
	StatefulSets    []appsv1.StatefulSet
	Services        []corev1.Service
	ConfigMaps      []corev1.ConfigMap
	Secrets         []corev1.Secret
	PVCs            []corev1.PersistentVolumeClaim
	Ingresses       []networkingv1.Ingress
	HPAs            []autoscalingv2.HorizontalPodAutoscaler
	PDBs            []policyv1.PodDisruptionBudget
	ServiceAccounts []corev1.ServiceAccount
	NetworkPolicies []networkingv1.NetworkPolicy
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

		// Detect service type early for smart decisions
		svcType := wizard.DetectServiceType(svc.Image)
		useStatefulSet := wizard.ShouldSuggestStatefulSet(svcType)

		// Check for wizard overrides
		var wizCfg *wizard.ServiceWizardConfig
		if opts.WizardOverrides != nil {
			if wc, ok := opts.WizardOverrides[name]; ok {
				wizCfg = &wc
				// Wizard can override the workload kind
				useStatefulSet = (wc.Kind == "StatefulSet")
			}
		}

		// Determine replicas: wizard override > compose deploy > default
		var replicas int32 = 1
		if wizCfg != nil && wizCfg.Replicas > 0 {
			replicas = wizCfg.Replicas
		} else if svc.Deploy != nil && svc.Deploy.Replicas != nil {
			replicas = int32(*svc.Deploy.Replicas)
		}

		if useStatefulSet {
			// StatefulSet for databases — includes VolumeClaimTemplates
			ss := generateStatefulSet(name, &svc, opts)
			// Apply wizard replicas
			ss.Spec.Replicas = &replicas
			result.StatefulSets = append(result.StatefulSets, ss)

			// Headless Service required for StatefulSet
			if len(svc.Ports) > 0 {
				k8sSvc := generateService(name, &svc, opts)
				k8sSvc.Spec.ClusterIP = "None" // headless
				result.Services = append(result.Services, k8sSvc)
			}
		} else {
			// Deployment for non-database services
			volResult := generateVolumes(name, &svc, compose, opts)
			result.PVCs = append(result.PVCs, volResult.PVCs...)
			warnings = append(warnings, volResult.Warnings...)

			deployment := generateDeployment(name, &svc, opts)
			// Apply wizard replicas
			deployment.Spec.Replicas = &replicas
			if len(volResult.PodVolumes) > 0 {
				deployment.Spec.Template.Spec.Volumes = volResult.PodVolumes
				deployment.Spec.Template.Spec.Containers[0].VolumeMounts = volResult.VolumeMounts
			}
			result.Deployments = append(result.Deployments, deployment)

			if len(svc.Ports) > 0 {
				k8sSvc := generateService(name, &svc, opts)
				result.Services = append(result.Services, k8sSvc)
			}
		}

		// Generate ConfigMap and Secret (if environment defined)
		if len(svc.Environment) > 0 {
			configMap, secretKeys := generateConfigMap(name, &svc, opts)
			if len(configMap.Data) > 0 {
				result.ConfigMaps = append(result.ConfigMaps, configMap)
			}
			if len(secretKeys) > 0 {
				secret := generateSecret(name, &svc, secretKeys, opts)
				result.Secrets = append(result.Secrets, secret)
			}
		}

		// Always generate ServiceAccount
		sa := generateServiceAccount(name, opts)
		result.ServiceAccounts = append(result.ServiceAccounts, sa)

		// Generate Ingress: wizard override or auto-detect HTTP ports
		addIngress := false
		ingressHost := ""
		addTLS := true
		if wizCfg != nil {
			addIngress = wizCfg.AddIngress
			ingressHost = wizCfg.IngressHost
			addTLS = wizCfg.AddTLS
		}
		if addIngress && ingressHost != "" {
			// Wizard-specified Ingress with custom host
			ingress := generateIngressWithHost(name, &svc, opts, ingressHost, addTLS)
			if ingress != nil {
				result.Ingresses = append(result.Ingresses, *ingress)
			}
		} else if wizCfg == nil {
			// No wizard: auto-detect from HTTP ports
			if ingress := generateIngress(name, &svc, opts); ingress != nil {
				result.Ingresses = append(result.Ingresses, *ingress)
			}
		}

		// Generate HPA: wizard override or auto-detect
		addHPA := false
		if wizCfg != nil {
			addHPA = wizCfg.AddHPA
		} else {
			addHPA = (svcType != wizard.ServiceTypeDatabase && svcType != wizard.ServiceTypeCache)
		}
		if addHPA {
			hpa := generateHPA(name, opts)
			if wizCfg != nil {
				// Apply wizard HPA settings
				if wizCfg.HPAMin > 0 {
					hpa.Spec.MinReplicas = &wizCfg.HPAMin
				}
				if wizCfg.HPAMax > 0 {
					hpa.Spec.MaxReplicas = wizCfg.HPAMax
				}
				if wizCfg.HPATargetCPU > 0 && len(hpa.Spec.Metrics) > 0 && hpa.Spec.Metrics[0].Resource != nil {
					hpa.Spec.Metrics[0].Resource.Target.AverageUtilization = &wizCfg.HPATargetCPU
				}
			}
			result.HPAs = append(result.HPAs, *hpa)
		}

		// Generate PDB: wizard override or auto (replicas > 1)
		addPDB := false
		if wizCfg != nil {
			addPDB = wizCfg.AddPDB
		} else {
			addPDB = (replicas > 1)
		}
		if addPDB {
			if pdb := generatePDB(name, replicas, opts); pdb != nil {
				result.PDBs = append(result.PDBs, *pdb)
			}
		}

		// Generate NetworkPolicy if enabled
		if opts.AddNetworkPolicy {
			np := generateNetworkPolicy(name, &svc, opts)
			result.NetworkPolicies = append(result.NetworkPolicies, *np)
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
