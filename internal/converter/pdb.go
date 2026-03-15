package converter

import (
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// generatePDB creates a PodDisruptionBudget for services with more than 1 replica.
// Returns nil if replicas <= 1.
func generatePDB(name string, replicas int32, opts ConvertOptions) *policyv1.PodDisruptionBudget {
	if replicas <= 1 {
		return nil
	}

	labels := standardLabels(name, opts.AppName)
	selector := selectorLabels(name)

	var minAvailable intstr.IntOrString
	if replicas > 2 {
		minAvailable = intstr.FromString("50%")
	} else {
		minAvailable = intstr.FromInt32(1)
	}

	pdb := &policyv1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy/v1",
			Kind:       "PodDisruptionBudget",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: opts.Namespace,
			Labels:    labels,
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: &minAvailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: selector,
			},
		},
	}

	return pdb
}
