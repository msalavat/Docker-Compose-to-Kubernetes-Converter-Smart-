package converter

import (
	"fmt"

	"github.com/compositor/kompoze/internal/parser"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func generateService(name string, svc *parser.ServiceConfig, opts ConvertOptions) corev1.Service {
	labels := standardLabels(name, opts.AppName)
	selector := selectorLabels(name)

	var ports []corev1.ServicePort
	for i, p := range svc.Ports {
		proto := corev1.ProtocolTCP
		if p.Protocol == "udp" {
			proto = corev1.ProtocolUDP
		}

		port := corev1.ServicePort{
			Name:       fmt.Sprintf("port-%d", i),
			Port:       int32(p.ContainerPort),
			TargetPort: intstr.FromInt32(int32(p.ContainerPort)),
			Protocol:   proto,
		}

		// If host port is specified and different, note it
		if p.HostPort > 0 && p.HostPort != p.ContainerPort {
			port.Port = int32(p.HostPort)
		}

		ports = append(ports, port)
	}

	return corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: opts.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: selector,
			Ports:    ports,
		},
	}
}
