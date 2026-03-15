package converter

import (
	"github.com/compositor/kompoze/internal/parser"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func generateStatefulSet(name string, svc *parser.ServiceConfig, opts ConvertOptions) appsv1.StatefulSet {
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
		Containers:         []corev1.Container{container},
		InitContainers:     initContainers,
		ServiceAccountName: name,
	}

	if opts.AddSecurity {
		podSpec.SecurityContext = buildPodSecurityContext(svc)
	}

	ss := appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: opts.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: name,
			Replicas:    &replicas,
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

	// Add VolumeClaimTemplates for named volumes
	for _, vol := range svc.Volumes {
		if vol.Type == "volume" {
			ss.Spec.VolumeClaimTemplates = append(ss.Spec.VolumeClaimTemplates,
				corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name: sanitizeName(vol.Source),
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("10Gi"),
							},
						},
					},
				},
			)
			ss.Spec.Template.Spec.Containers[0].VolumeMounts = append(
				ss.Spec.Template.Spec.Containers[0].VolumeMounts,
				corev1.VolumeMount{
					Name:      sanitizeName(vol.Source),
					MountPath: vol.Target,
				},
			)
		}
	}

	return ss
}
