package converter

import (
	"fmt"
	"strings"

	"github.com/compositor/kompoze/internal/parser"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VolumeConvertResult holds the generated PVCs and warnings for a service.
type VolumeConvertResult struct {
	PVCs         []corev1.PersistentVolumeClaim
	PodVolumes   []corev1.Volume
	VolumeMounts []corev1.VolumeMount
	Warnings     []string
}

// generateVolumes processes compose volumes and returns PVCs, pod volumes, and mounts.
func generateVolumes(name string, svc *parser.ServiceConfig, compose *parser.ComposeFile, opts ConvertOptions) VolumeConvertResult {
	var result VolumeConvertResult

	for _, vol := range svc.Volumes {
		switch vol.Type {
		case "volume":
			pvcName := sanitizeName(name + "-" + vol.Source)
			if vol.Source == "" {
				pvcName = sanitizeName(name + "-data")
			}

			// Check if top-level volume has a driver
			storageClass := ""
			if compose != nil {
				if topVol, ok := compose.Volumes[vol.Source]; ok && topVol.Driver != "" {
					storageClass = topVol.Driver
				}
			}

			pvc := generatePVC(pvcName, storageClass, opts)
			result.PVCs = append(result.PVCs, pvc)

			result.PodVolumes = append(result.PodVolumes, corev1.Volume{
				Name: pvcName,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvcName,
					},
				},
			})
			result.VolumeMounts = append(result.VolumeMounts, corev1.VolumeMount{
				Name:      pvcName,
				MountPath: vol.Target,
				ReadOnly:  vol.ReadOnly,
			})

		case "bind":
			// Bind mounts can't be directly converted
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Bind mount '%s:%s' cannot be automatically converted. Use a PVC or ConfigMap.", vol.Source, vol.Target))

		case "tmpfs":
			tmpName := sanitizeName(name + "-tmpfs-" + strings.ReplaceAll(vol.Target, "/", "-"))
			result.PodVolumes = append(result.PodVolumes, corev1.Volume{
				Name: tmpName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{
						Medium: corev1.StorageMediumMemory,
					},
				},
			})
			result.VolumeMounts = append(result.VolumeMounts, corev1.VolumeMount{
				Name:      tmpName,
				MountPath: vol.Target,
			})
		}
	}

	return result
}

func generatePVC(name string, storageClass string, opts ConvertOptions) corev1.PersistentVolumeClaim {
	labels := standardLabels(name, opts.AppName)

	pvc := corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "PersistentVolumeClaim",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: opts.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}

	if storageClass != "" {
		pvc.Spec.StorageClassName = &storageClass
	}

	return pvc
}

func sanitizeName(name string) string {
	name = strings.ToLower(name)
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '-'
	}, name)
	// Trim leading/trailing dashes
	name = strings.Trim(name, "-")
	// Truncate to 63 chars (K8s name limit)
	if len(name) > 63 {
		name = name[:63]
	}
	return name
}
