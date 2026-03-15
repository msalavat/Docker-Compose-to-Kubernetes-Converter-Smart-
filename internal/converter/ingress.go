package converter

import (
	"github.com/compositor/kompoze/internal/parser"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// generateIngress creates an Ingress for services with published HTTP ports.
// Returns nil if the service has no HTTP ports.
func generateIngress(name string, svc *parser.ServiceConfig, opts ConvertOptions) *networkingv1.Ingress {
	var httpPorts []int32
	for _, p := range svc.Ports {
		if isHTTPPort(p.ContainerPort) {
			httpPorts = append(httpPorts, int32(p.ContainerPort))
		}
	}
	if len(httpPorts) == 0 {
		return nil
	}

	labels := standardLabels(name, opts.AppName)
	host := name + ".example.com"
	ingressClassName := "nginx"
	pathType := networkingv1.PathTypePrefix

	// Build one path per HTTP port
	var paths []networkingv1.HTTPIngressPath
	for _, port := range httpPorts {
		paths = append(paths, networkingv1.HTTPIngressPath{
			Path:     "/",
			PathType: &pathType,
			Backend: networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: name,
					Port: networkingv1.ServiceBackendPort{
						Number: port,
					},
				},
			},
		})
	}

	ingress := &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.k8s.io/v1",
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: opts.Namespace,
			Labels:    labels,
			Annotations: map[string]string{
				"cert-manager.io/cluster-issuer": "letsencrypt-prod",
			},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &ingressClassName,
			TLS: []networkingv1.IngressTLS{
				{
					Hosts:      []string{host},
					SecretName: name + "-tls",
				},
			},
			Rules: []networkingv1.IngressRule{
				{
					Host: host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: paths,
						},
					},
				},
			},
		},
	}

	return ingress
}

// generateIngressWithHost creates an Ingress with a custom hostname (from wizard).
// Uses all service ports (not just HTTP). TLS is optional.
func generateIngressWithHost(name string, svc *parser.ServiceConfig, opts ConvertOptions, host string, addTLS bool) *networkingv1.Ingress {
	if len(svc.Ports) == 0 {
		return nil
	}

	labels := standardLabels(name, opts.AppName)
	ingressClassName := "nginx"
	pathType := networkingv1.PathTypePrefix

	// Use first port as the backend
	backendPort := int32(svc.Ports[0].ContainerPort)

	paths := []networkingv1.HTTPIngressPath{
		{
			Path:     "/",
			PathType: &pathType,
			Backend: networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: name,
					Port: networkingv1.ServiceBackendPort{
						Number: backendPort,
					},
				},
			},
		},
	}

	annotations := map[string]string{}
	if addTLS {
		annotations["cert-manager.io/cluster-issuer"] = "letsencrypt-prod"
	}

	ingress := &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.k8s.io/v1",
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   opts.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &ingressClassName,
			Rules: []networkingv1.IngressRule{
				{
					Host: host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: paths,
						},
					},
				},
			},
		},
	}

	if addTLS {
		ingress.Spec.TLS = []networkingv1.IngressTLS{
			{
				Hosts:      []string{host},
				SecretName: name + "-tls",
			},
		}
	}

	return ingress
}
