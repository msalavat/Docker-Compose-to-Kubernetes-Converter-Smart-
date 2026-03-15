package converter

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// generateServiceAccount creates a ServiceAccount with automount disabled.
func generateServiceAccount(name string, opts ConvertOptions) corev1.ServiceAccount {
	labels := standardLabels(name, opts.AppName)
	automount := false

	return corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: opts.Namespace,
			Labels:    labels,
		},
		AutomountServiceAccountToken: &automount,
	}
}
