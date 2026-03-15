package converter

import (
	"sort"

	"github.com/compositor/kompoze/internal/parser"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// generateConfigMap creates a ConfigMap from non-sensitive environment variables.
// Returns the ConfigMap and a list of sensitive keys that should be in a Secret.
func generateConfigMap(name string, svc *parser.ServiceConfig, opts ConvertOptions) (corev1.ConfigMap, []string) {
	labels := standardLabels(name, opts.AppName)

	data := make(map[string]string)
	var secretKeys []string

	// Sort keys for deterministic output
	keys := make([]string, 0, len(svc.Environment))
	for k := range svc.Environment {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if isSensitiveKey(k) {
			secretKeys = append(secretKeys, k)
		} else {
			data[k] = svc.Environment[k]
		}
	}

	cm := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-config",
			Namespace: opts.Namespace,
			Labels:    labels,
		},
		Data: data,
	}

	return cm, secretKeys
}
