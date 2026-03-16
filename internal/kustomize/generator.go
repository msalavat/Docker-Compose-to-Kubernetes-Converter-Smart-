// Package kustomize generates Kustomize base and overlay structures from converted manifests.
package kustomize

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/compositor/kompoze/internal/converter"
	"k8s.io/apimachinery/pkg/runtime"
	sigsyaml "sigs.k8s.io/yaml"
)

// GenerateOptions holds configuration for Kustomize generation.
type GenerateOptions struct {
	OutputDir string
	AppName   string
	Namespace string
}

// Generate creates a Kustomize structure with base + overlays (dev/staging/prod).
func Generate(result *converter.ConvertResult, opts GenerateOptions) error {
	baseDir := filepath.Join(opts.OutputDir, "base")
	overlays := map[string]string{
		"dev":     filepath.Join(opts.OutputDir, "overlays", "dev"),
		"staging": filepath.Join(opts.OutputDir, "overlays", "staging"),
		"prod":    filepath.Join(opts.OutputDir, "overlays", "prod"),
	}

	// Create directories
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("creating base directory: %w", err)
	}
	for _, dir := range overlays {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating overlay directory: %w", err)
		}
	}

	// Write base resources and kustomization.yaml
	resources, err := writeBaseResources(baseDir, result)
	if err != nil {
		return fmt.Errorf("writing base resources: %w", err)
	}

	if err := writeBaseKustomization(baseDir, resources); err != nil {
		return fmt.Errorf("writing base kustomization: %w", err)
	}

	// Write overlays
	if err := writeDevOverlay(overlays["dev"], result); err != nil {
		return fmt.Errorf("writing dev overlay: %w", err)
	}
	if err := writeStagingOverlay(overlays["staging"], result); err != nil {
		return fmt.Errorf("writing staging overlay: %w", err)
	}
	if err := writeProdOverlay(overlays["prod"], result); err != nil {
		return fmt.Errorf("writing prod overlay: %w", err)
	}

	return nil
}

func writeBaseResources(baseDir string, result *converter.ConvertResult) ([]string, error) {
	var resources []string

	for i := range result.Deployments {
		d := &result.Deployments[i]
		// Clear namespace from base (overlays set it)
		d.Namespace = ""
		filename := d.Name + "-deployment.yaml"
		if err := writeK8sResource(baseDir, filename, d); err != nil {
			return nil, err
		}
		resources = append(resources, filename)
	}

	for i := range result.StatefulSets {
		ss := &result.StatefulSets[i]
		ss.Namespace = ""
		filename := ss.Name + "-statefulset.yaml"
		if err := writeK8sResource(baseDir, filename, ss); err != nil {
			return nil, err
		}
		resources = append(resources, filename)
	}

	for i := range result.Services {
		s := &result.Services[i]
		s.Namespace = ""
		filename := s.Name + "-service.yaml"
		if err := writeK8sResource(baseDir, filename, s); err != nil {
			return nil, err
		}
		resources = append(resources, filename)
	}

	for i := range result.ConfigMaps {
		cm := &result.ConfigMaps[i]
		cm.Namespace = ""
		filename := cm.Name + ".yaml"
		if err := writeK8sResource(baseDir, filename, cm); err != nil {
			return nil, err
		}
		resources = append(resources, filename)
	}

	for i := range result.Secrets {
		sec := &result.Secrets[i]
		sec.Namespace = ""
		filename := sec.Name + ".yaml"
		if err := writeK8sResource(baseDir, filename, sec); err != nil {
			return nil, err
		}
		resources = append(resources, filename)
	}

	for i := range result.PVCs {
		pvc := &result.PVCs[i]
		pvc.Namespace = ""
		filename := pvc.Name + ".yaml"
		if err := writeK8sResource(baseDir, filename, pvc); err != nil {
			return nil, err
		}
		resources = append(resources, filename)
	}

	for i := range result.ServiceAccounts {
		sa := &result.ServiceAccounts[i]
		sa.Namespace = ""
		filename := sa.Name + "-serviceaccount.yaml"
		if err := writeK8sResource(baseDir, filename, sa); err != nil {
			return nil, err
		}
		resources = append(resources, filename)
	}

	return resources, nil
}

func writeBaseKustomization(baseDir string, resources []string) error {
	var sb strings.Builder
	sb.WriteString("apiVersion: kustomize.config.k8s.io/v1beta1\n")
	sb.WriteString("kind: Kustomization\n\n")
	sb.WriteString("commonLabels:\n")
	sb.WriteString("  app.kubernetes.io/managed-by: kompoze\n\n")
	sb.WriteString("resources:\n")
	for _, r := range resources {
		sb.WriteString(fmt.Sprintf("  - %s\n", r))
	}

	return os.WriteFile(filepath.Join(baseDir, "kustomization.yaml"), []byte(sb.String()), 0644)
}

func writeDevOverlay(dir string, result *converter.ConvertResult) error {
	var sb strings.Builder
	sb.WriteString("apiVersion: kustomize.config.k8s.io/v1beta1\n")
	sb.WriteString("kind: Kustomization\n\n")
	sb.WriteString("namespace: dev\n\n")
	sb.WriteString("resources:\n")
	sb.WriteString("  - ../../base\n\n")

	// Patches: replicas=1, minimal resources
	var patches []string
	for _, d := range result.Deployments {
		patch := generateReplicasPatch(d.Name, 1)
		patchFile := d.Name + "-replicas-patch.yaml"
		if err := os.WriteFile(filepath.Join(dir, patchFile), []byte(patch), 0644); err != nil {
			return err
		}
		patches = append(patches, patchFile)

		resPatch := generateResourcesPatch(d.Name, "50m", "64Mi", "200m", "128Mi")
		resPatchFile := d.Name + "-resources-patch.yaml"
		if err := os.WriteFile(filepath.Join(dir, resPatchFile), []byte(resPatch), 0644); err != nil {
			return err
		}
		patches = append(patches, resPatchFile)
	}

	if len(patches) > 0 {
		sb.WriteString("patches:\n")
		for _, p := range patches {
			sb.WriteString(fmt.Sprintf("  - path: %s\n", p))
		}
	}

	return os.WriteFile(filepath.Join(dir, "kustomization.yaml"), []byte(sb.String()), 0644)
}

func writeStagingOverlay(dir string, result *converter.ConvertResult) error {
	var sb strings.Builder
	sb.WriteString("apiVersion: kustomize.config.k8s.io/v1beta1\n")
	sb.WriteString("kind: Kustomization\n\n")
	sb.WriteString("namespace: staging\n\n")
	sb.WriteString("resources:\n")
	sb.WriteString("  - ../../base\n\n")

	var patches []string
	for _, d := range result.Deployments {
		patch := generateReplicasPatch(d.Name, 2)
		patchFile := d.Name + "-replicas-patch.yaml"
		if err := os.WriteFile(filepath.Join(dir, patchFile), []byte(patch), 0644); err != nil {
			return err
		}
		patches = append(patches, patchFile)

		resPatch := generateResourcesPatch(d.Name, "100m", "128Mi", "500m", "256Mi")
		resPatchFile := d.Name + "-resources-patch.yaml"
		if err := os.WriteFile(filepath.Join(dir, resPatchFile), []byte(resPatch), 0644); err != nil {
			return err
		}
		patches = append(patches, resPatchFile)
	}

	// Add Ingress resources for services that have them
	for _, ing := range result.Ingresses {
		ingCopy := ing.DeepCopy()
		ingCopy.Namespace = ""
		// Update host for staging
		if len(ingCopy.Spec.Rules) > 0 {
			host := "staging-" + ingCopy.Spec.Rules[0].Host
			ingCopy.Spec.Rules[0].Host = host
			if len(ingCopy.Spec.TLS) > 0 {
				ingCopy.Spec.TLS[0].Hosts = []string{host}
			}
		}
		filename := ing.Name + "-ingress.yaml"
		if err := writeK8sResource(dir, filename, ingCopy); err != nil {
			return err
		}
		patches = append(patches, filename)
	}

	if len(patches) > 0 {
		sb.WriteString("patches:\n")
		for _, p := range patches {
			sb.WriteString(fmt.Sprintf("  - path: %s\n", p))
		}
	}

	return os.WriteFile(filepath.Join(dir, "kustomization.yaml"), []byte(sb.String()), 0644)
}

func writeProdOverlay(dir string, result *converter.ConvertResult) error {
	var sb strings.Builder
	sb.WriteString("apiVersion: kustomize.config.k8s.io/v1beta1\n")
	sb.WriteString("kind: Kustomization\n\n")
	sb.WriteString("namespace: production\n\n")
	sb.WriteString("resources:\n")
	sb.WriteString("  - ../../base\n")

	var extraResources []string
	var patches []string

	for _, d := range result.Deployments {
		replicas := int32(3)
		if d.Spec.Replicas != nil && *d.Spec.Replicas > 1 {
			replicas = *d.Spec.Replicas
		}
		patch := generateReplicasPatch(d.Name, replicas)
		patchFile := d.Name + "-replicas-patch.yaml"
		if err := os.WriteFile(filepath.Join(dir, patchFile), []byte(patch), 0644); err != nil {
			return err
		}
		patches = append(patches, patchFile)

		resPatch := generateResourcesPatch(d.Name, "200m", "256Mi", "1000m", "512Mi")
		resPatchFile := d.Name + "-resources-patch.yaml"
		if err := os.WriteFile(filepath.Join(dir, resPatchFile), []byte(resPatch), 0644); err != nil {
			return err
		}
		patches = append(patches, resPatchFile)
	}

	// Add Ingress with prod hosts
	for _, ing := range result.Ingresses {
		ingCopy := ing.DeepCopy()
		ingCopy.Namespace = ""
		filename := ing.Name + "-ingress.yaml"
		if err := writeK8sResource(dir, filename, ingCopy); err != nil {
			return err
		}
		extraResources = append(extraResources, filename)
	}

	// Add HPAs
	for _, hpa := range result.HPAs {
		hpaCopy := hpa.DeepCopy()
		hpaCopy.Namespace = ""
		filename := hpa.Name + "-hpa.yaml"
		if err := writeK8sResource(dir, filename, hpaCopy); err != nil {
			return err
		}
		extraResources = append(extraResources, filename)
	}

	// Add PDBs
	for _, pdb := range result.PDBs {
		pdbCopy := pdb.DeepCopy()
		pdbCopy.Namespace = ""
		filename := pdb.Name + "-pdb.yaml"
		if err := writeK8sResource(dir, filename, pdbCopy); err != nil {
			return err
		}
		extraResources = append(extraResources, filename)
	}

	// Add NetworkPolicies
	for _, np := range result.NetworkPolicies {
		npCopy := np.DeepCopy()
		npCopy.Namespace = ""
		filename := np.Name + "-networkpolicy.yaml"
		if err := writeK8sResource(dir, filename, npCopy); err != nil {
			return err
		}
		extraResources = append(extraResources, filename)
	}

	for _, r := range extraResources {
		sb.WriteString(fmt.Sprintf("  - %s\n", r))
	}

	sb.WriteString("\n")

	if len(patches) > 0 {
		sb.WriteString("patches:\n")
		for _, p := range patches {
			sb.WriteString(fmt.Sprintf("  - path: %s\n", p))
		}
	}

	return os.WriteFile(filepath.Join(dir, "kustomization.yaml"), []byte(sb.String()), 0644)
}

func generateReplicasPatch(name string, replicas int32) string {
	return fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
spec:
  replicas: %d
`, name, replicas)
}

func generateResourcesPatch(name, cpuReq, memReq, cpuLim, memLim string) string {
	return fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
spec:
  template:
    spec:
      containers:
        - name: %s
          resources:
            requests:
              cpu: %s
              memory: %s
            limits:
              cpu: %s
              memory: %s
`, name, name, cpuReq, memReq, cpuLim, memLim)
}

func writeK8sResource(dir, filename string, obj runtime.Object) error {
	data, err := sigsyaml.Marshal(obj)
	if err != nil {
		return fmt.Errorf("marshaling %s: %w", filename, err)
	}
	return os.WriteFile(filepath.Join(dir, filename), []byte("# Generated by kompoze\n"+string(data)), 0644)
}
