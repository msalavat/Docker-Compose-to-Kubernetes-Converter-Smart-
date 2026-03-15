package converter

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/compositor/kompoze/internal/parser"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func generateDeployment(name string, svc *parser.ServiceConfig, opts ConvertOptions) appsv1.Deployment {
	labels := standardLabels(name, opts.AppName)
	selector := selectorLabels(name)

	var replicas int32 = 1
	if svc.Deploy != nil && svc.Deploy.Replicas != nil {
		replicas = int32(*svc.Deploy.Replicas)
	}

	container := buildContainer(name, svc, opts)

	// Init containers for depends_on
	var initContainers []corev1.Container
	if len(svc.DependsOn) > 0 {
		initContainers = buildInitContainers(svc.DependsOn)
	}

	podSpec := corev1.PodSpec{
		Containers:     []corev1.Container{container},
		InitContainers: initContainers,
	}

	// Security context at pod level
	if opts.AddSecurity {
		podSpec.SecurityContext = buildPodSecurityContext(svc)
	}

	deployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: opts.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: selector,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: mergeLabels(labels, selector),
				},
				Spec: podSpec,
			},
		},
	}

	return deployment
}

func buildContainer(name string, svc *parser.ServiceConfig, opts ConvertOptions) corev1.Container {
	container := corev1.Container{
		Name:  name,
		Image: svc.Image,
	}

	// Command
	if len(svc.Command) > 0 {
		if len(svc.Command) == 1 {
			// Shell form: split into shell -c "cmd"
			container.Command = []string{"/bin/sh", "-c", svc.Command[0]}
		} else {
			container.Command = svc.Command
		}
	}

	// Entrypoint
	if len(svc.Entrypoint) > 0 {
		if len(svc.Entrypoint) == 1 && strings.Contains(svc.Entrypoint[0], " ") {
			container.Command = []string{"/bin/sh", "-c", svc.Entrypoint[0]}
		} else {
			container.Command = svc.Entrypoint
		}
		// If both entrypoint and command, entrypoint=Command, command=Args
		if len(svc.Command) > 0 {
			container.Args = svc.Command
		}
	}

	// Ports
	for _, p := range svc.Ports {
		proto := corev1.ProtocolTCP
		if p.Protocol == "udp" {
			proto = corev1.ProtocolUDP
		}
		container.Ports = append(container.Ports, corev1.ContainerPort{
			ContainerPort: int32(p.ContainerPort),
			Protocol:      proto,
		})
	}

	// Environment - from ConfigMap ref + Secret refs
	if len(svc.Environment) > 0 {
		for k, v := range svc.Environment {
			if isSensitiveKey(k) {
				// Secret reference (placeholder)
				container.Env = append(container.Env, corev1.EnvVar{
					Name: k,
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: name + "-secret",
							},
							Key: k,
						},
					},
				})
			} else {
				container.EnvFrom = append(container.EnvFrom, corev1.EnvFromSource{
					ConfigMapRef: &corev1.ConfigMapEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: name + "-config",
						},
					},
				})
				_ = v
				break // Only need one EnvFrom for the ConfigMap
			}
		}
	}

	// Working directory
	if svc.WorkingDir != "" {
		container.WorkingDir = svc.WorkingDir
	}

	// Probes
	if opts.AddProbes {
		liveness, readiness := buildProbes(svc)
		container.LivenessProbe = liveness
		container.ReadinessProbe = readiness
	}

	// Resources
	if opts.AddResources {
		container.Resources = buildResources(svc)
	}

	// Security context at container level
	if opts.AddSecurity {
		container.SecurityContext = buildContainerSecurityContext(svc)
	}

	return container
}

func buildProbes(svc *parser.ServiceConfig) (*corev1.Probe, *corev1.Probe) {
	if svc.Healthcheck != nil && !svc.Healthcheck.Disable && len(svc.Healthcheck.Test) > 0 {
		return buildProbesFromHealthcheck(svc.Healthcheck)
	}

	// Smart defaults based on ports
	if len(svc.Ports) > 0 {
		port := svc.Ports[0].ContainerPort
		if isHTTPPort(port) {
			probe := &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/",
						Port: intstr.FromInt32(int32(port)),
					},
				},
				InitialDelaySeconds: 10,
				PeriodSeconds:       15,
				TimeoutSeconds:      5,
				FailureThreshold:    3,
			}
			readiness := probe.DeepCopy()
			readiness.InitialDelaySeconds = 5
			return probe, readiness
		}
		// TCP probe for non-HTTP ports
		probe := &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt32(int32(port)),
				},
			},
			InitialDelaySeconds: 10,
			PeriodSeconds:       15,
			TimeoutSeconds:      5,
			FailureThreshold:    3,
		}
		readiness := probe.DeepCopy()
		readiness.InitialDelaySeconds = 5
		return probe, readiness
	}

	return nil, nil
}

func buildProbesFromHealthcheck(hc *parser.HealthcheckConfig) (*corev1.Probe, *corev1.Probe) {
	var handler corev1.ProbeHandler

	test := hc.Test
	if len(test) == 0 {
		return nil, nil
	}

	// Parse CMD or CMD-SHELL
	if test[0] == "CMD" {
		handler = corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: test[1:],
			},
		}
	} else if test[0] == "CMD-SHELL" {
		handler = corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{"/bin/sh", "-c", strings.Join(test[1:], " ")},
			},
		}
	} else {
		// Plain string command
		handler = corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{"/bin/sh", "-c", strings.Join(test, " ")},
			},
		}
	}

	periodSeconds := int32(30)
	if hc.Interval != "" {
		if secs := parseDurationSeconds(hc.Interval); secs > 0 {
			periodSeconds = secs
		}
	}
	timeoutSeconds := int32(10)
	if hc.Timeout != "" {
		if secs := parseDurationSeconds(hc.Timeout); secs > 0 {
			timeoutSeconds = secs
		}
	}
	retries := int32(3)
	if hc.Retries > 0 {
		retries = int32(hc.Retries)
	}
	startPeriod := int32(0)
	if hc.StartPeriod != "" {
		if secs := parseDurationSeconds(hc.StartPeriod); secs > 0 {
			startPeriod = secs
		}
	}

	liveness := &corev1.Probe{
		ProbeHandler:        handler,
		InitialDelaySeconds: startPeriod,
		PeriodSeconds:       periodSeconds,
		TimeoutSeconds:      timeoutSeconds,
		FailureThreshold:    retries,
	}
	readiness := liveness.DeepCopy()
	readiness.InitialDelaySeconds = startPeriod / 2

	return liveness, readiness
}

func buildResources(svc *parser.ServiceConfig) corev1.ResourceRequirements {
	if svc.Deploy != nil && svc.Deploy.Resources != nil {
		return convertComposeResources(svc.Deploy.Resources)
	}

	// Smart defaults
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		},
	}
}

func convertComposeResources(res *parser.Resources) corev1.ResourceRequirements {
	rr := corev1.ResourceRequirements{}

	if res.Limits != nil {
		rr.Limits = corev1.ResourceList{}
		if res.Limits.CPUs != "" {
			cpuMillis := parseCPUs(res.Limits.CPUs)
			rr.Limits[corev1.ResourceCPU] = resource.MustParse(fmt.Sprintf("%dm", cpuMillis))
		}
		if res.Limits.Memory != "" {
			rr.Limits[corev1.ResourceMemory] = resource.MustParse(normalizeMemory(res.Limits.Memory))
		}
	}

	if res.Reservations != nil {
		rr.Requests = corev1.ResourceList{}
		if res.Reservations.CPUs != "" {
			cpuMillis := parseCPUs(res.Reservations.CPUs)
			rr.Requests[corev1.ResourceCPU] = resource.MustParse(fmt.Sprintf("%dm", cpuMillis))
		}
		if res.Reservations.Memory != "" {
			rr.Requests[corev1.ResourceMemory] = resource.MustParse(normalizeMemory(res.Reservations.Memory))
		}
	}

	return rr
}

func buildPodSecurityContext(svc *parser.ServiceConfig) *corev1.PodSecurityContext {
	fsGroup := int64(1000)
	runAsNonRoot := true

	if svc.Privileged {
		runAsNonRoot = false
	}

	return &corev1.PodSecurityContext{
		RunAsNonRoot: &runAsNonRoot,
		FSGroup:      &fsGroup,
	}
}

func buildContainerSecurityContext(svc *parser.ServiceConfig) *corev1.SecurityContext {
	allowPrivEsc := false
	readOnly := !svc.Privileged

	if svc.ReadOnly {
		readOnly = true
	}

	sc := &corev1.SecurityContext{
		AllowPrivilegeEscalation: &allowPrivEsc,
		ReadOnlyRootFilesystem:   &readOnly,
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}

	// Map cap_add from compose
	for _, cap := range svc.CapAdd {
		sc.Capabilities.Add = append(sc.Capabilities.Add, corev1.Capability(cap))
	}

	return sc
}

func buildInitContainers(deps parser.DependsOn) []corev1.Container {
	var initContainers []corev1.Container

	for depName := range deps {
		initContainers = append(initContainers, corev1.Container{
			Name:    "wait-for-" + depName,
			Image:   "busybox:1.36",
			Command: []string{"sh", "-c", fmt.Sprintf("until nc -z %s 1; do echo waiting for %s; sleep 2; done", depName, depName)},
		})
	}

	return initContainers
}

// --- Helpers ---

func isHTTPPort(port uint32) bool {
	httpPorts := map[uint32]bool{80: true, 443: true, 8080: true, 3000: true, 5000: true, 8000: true, 8443: true}
	return httpPorts[port]
}

func isSensitiveKey(key string) bool {
	upper := strings.ToUpper(key)
	sensitivePatterns := []string{"PASSWORD", "SECRET", "TOKEN", "KEY", "CREDENTIALS", "API_KEY", "PRIVATE"}
	for _, pattern := range sensitivePatterns {
		if strings.Contains(upper, pattern) {
			return true
		}
	}
	return false
}

func parseDurationSeconds(s string) int32 {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "s") {
		val, err := strconv.Atoi(strings.TrimSuffix(s, "s"))
		if err == nil {
			return int32(val)
		}
	}
	if strings.HasSuffix(s, "m") {
		val, err := strconv.Atoi(strings.TrimSuffix(s, "m"))
		if err == nil {
			return int32(val * 60)
		}
	}
	return 0
}

func parseCPUs(s string) int64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 500
	}
	return int64(f * 1000)
}

func normalizeMemory(s string) string {
	// Docker uses M, G; K8s uses Mi, Gi
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "M") && !strings.HasSuffix(s, "Mi") {
		return strings.TrimSuffix(s, "M") + "Mi"
	}
	if strings.HasSuffix(s, "G") && !strings.HasSuffix(s, "Gi") {
		return strings.TrimSuffix(s, "G") + "Gi"
	}
	if strings.HasSuffix(s, "K") && !strings.HasSuffix(s, "Ki") {
		return strings.TrimSuffix(s, "K") + "Ki"
	}
	return s
}

func mergeLabels(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}
