package controller

import (
	"fmt"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/pointer"
)

const (
	serverNamespace        = "minecraft-server"
	dataVolumeName         = "data"
	dataMountPath          = "/data"
	cacheVolumeName        = "restic-cache"
	cacheMountPath         = "/cache"
	cacheSizeLimit         = "1Gi"
	saveGracePeriodSeconds = int64(180)
	saveFinalizer          = "servers.blamedevs.com/save"
	finalizeRetryInterval  = 5 * time.Second
)

type SaveConfig struct {
	Endpoint      string
	Bucket        string
	Region        string
	SecretName    string
	Image         string
	PullSecret    string
	Interval      time.Duration
	KeepSnapshots int
}

func (c SaveConfig) Configured() bool {
	return c.Endpoint != "" && c.Bucket != "" && c.Region != "" && c.SecretName != "" && c.Image != ""
}

func (c SaveConfig) repository(saveID string) string {
	return fmt.Sprintf("s3:%s/%s/saves/%s", c.Endpoint, c.Bucket, saveID)
}

func (c SaveConfig) env(saveID string) []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: "RESTIC_REPOSITORY", Value: c.repository(saveID)},
		{Name: "RESTIC_CACHE_DIR", Value: cacheMountPath},
		{Name: "AWS_DEFAULT_REGION", Value: c.Region},
		{Name: "HOME", Value: "/tmp"},
		{Name: "SNAPSHOT_INTERVAL_SECONDS", Value: strconv.Itoa(int(c.Interval.Seconds()))},
		{Name: "KEEP_SNAPSHOTS", Value: strconv.Itoa(c.KeepSnapshots)},
	}
}

func (c SaveConfig) container(name string, command string, saveID string, cpu string, memory string) corev1.Container {
	return corev1.Container{
		Name:    name,
		Image:   c.Image,
		Command: []string{command},
		Env:     c.env(saveID),
		EnvFrom: []corev1.EnvFromSource{
			{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: c.SecretName}}},
		},
		ImagePullPolicy: corev1.PullAlways,
		SecurityContext: containerSecurityContext(),
		VolumeMounts: []corev1.VolumeMount{
			{Name: dataVolumeName, MountPath: dataMountPath},
			{Name: cacheVolumeName, MountPath: cacheMountPath},
		},
		Resources: corev1.ResourceRequirements{
			Requests: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceCPU:    resource.MustParse(cpu),
				corev1.ResourceMemory: resource.MustParse(memory),
			},
			Limits: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceMemory: resource.MustParse(memory),
			},
		},
	}
}

func (c SaveConfig) initContainers(saveID string) []corev1.Container {
	restore := c.container("restore", "restore.sh", saveID, "100m", "256Mi")

	saver := c.container("saver", "save.sh", saveID, "50m", "512Mi")
	saver.RestartPolicy = (*corev1.ContainerRestartPolicy)(pointer.String(string(corev1.ContainerRestartPolicyAlways)))

	return []corev1.Container{restore, saver}
}

func cacheVolume() corev1.Volume {
	limit := resource.MustParse(cacheSizeLimit)

	return corev1.Volume{
		Name: cacheVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{SizeLimit: &limit},
		},
	}
}
