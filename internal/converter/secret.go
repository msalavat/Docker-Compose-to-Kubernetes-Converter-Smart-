package converter

import (
	"github.com/compositor/kompoze/internal/parser"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// generateSecret creates a Kubernetes Secret from sensitive environment variables.
func generateSecret(name string, svc *parser.ServiceConfig, secretKeys []string, opts ConvertOptions) corev1.Secret {
	labels := standardLabels(name, opts.AppName)

	data := make(map[string]string)
	for _, k := range secretKeys {
		if v, ok := svc.Environment[k]; ok {
			data[k] = v
		}
	}

	return corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-secret",
			Namespace: opts.Namespace,
			Labels:    labels,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: data,
	}
}
