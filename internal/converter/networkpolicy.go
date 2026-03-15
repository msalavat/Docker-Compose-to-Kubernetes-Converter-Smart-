package converter

import (
	"github.com/compositor/kompoze/internal/parser"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// generateNetworkPolicy creates a NetworkPolicy based on depends_on relationships.
// It allows ingress from services managed by kompoze on exposed ports and
// egress to dependencies. Default deny for all other traffic.
func generateNetworkPolicy(name string, svc *parser.ServiceConfig, opts ConvertOptions) *networkingv1.NetworkPolicy {
	labels := standardLabels(name, opts.AppName)
	selector := selectorLabels(name)

	// Build egress rules: allow traffic to each dependency
	var egressRules []networkingv1.NetworkPolicyEgressRule
	for depName := range svc.DependsOn {
		egressRules = append(egressRules, networkingv1.NetworkPolicyEgressRule{
			To: []networkingv1.NetworkPolicyPeer{
				{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: selectorLabels(depName),
					},
				},
			},
		})
	}

	// Allow DNS egress (kube-dns on port 53 UDP/TCP)
	udp := corev1.ProtocolUDP
	tcp := corev1.ProtocolTCP
	dnsPort := intstr.FromInt32(53)
	egressRules = append(egressRules, networkingv1.NetworkPolicyEgressRule{
		Ports: []networkingv1.NetworkPolicyPort{
			{
				Protocol: &udp,
				Port:     &dnsPort,
			},
			{
				Protocol: &tcp,
				Port:     &dnsPort,
			},
		},
	})

	// Build ingress rules: allow traffic from kompoze-managed pods on exposed ports
	var ingressRules []networkingv1.NetworkPolicyIngressRule
	if len(svc.Ports) > 0 {
		var ports []networkingv1.NetworkPolicyPort
		for _, p := range svc.Ports {
			proto := corev1.ProtocolTCP
			npPort := intstr.FromInt32(int32(p.ContainerPort))
			ports = append(ports, networkingv1.NetworkPolicyPort{
				Protocol: &proto,
				Port:     &npPort,
			})
		}
		ingressRules = append(ingressRules, networkingv1.NetworkPolicyIngressRule{
			From: []networkingv1.NetworkPolicyPeer{
				{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/managed-by": "kompoze",
						},
					},
				},
			},
			Ports: ports,
		})
	}

	np := &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.k8s.io/v1",
			Kind:       "NetworkPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: opts.Namespace,
			Labels:    labels,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: selector,
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			Ingress: ingressRules,
			Egress:  egressRules,
		},
	}

	return np
}
