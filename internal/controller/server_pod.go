package controller

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/pointer"
)

// Servers run user-chosen mods, so pods get no cluster credentials, no cluster DNS and no privileges.
const (
	serverUID         = 1000
	serverCPU         = "500m"
	serverDiskRequest = "2Gi"
	serverDiskLimit   = "8Gi"
)

func podSecurityContext() *corev1.PodSecurityContext {
	return &corev1.PodSecurityContext{
		RunAsNonRoot: pointer.Bool(true),
		RunAsUser:    pointer.Int64(serverUID),
		RunAsGroup:   pointer.Int64(serverUID),
		FSGroup:      pointer.Int64(serverUID),
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
}

func containerSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		AllowPrivilegeEscalation: pointer.Bool(false),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}
}

func publicDNS() *corev1.PodDNSConfig {
	return &corev1.PodDNSConfig{
		Nameservers: []string{"1.1.1.1", "8.8.8.8"},
	}
}

func diskLimit() *resource.Quantity {
	limit := resource.MustParse(serverDiskLimit)
	return &limit
}
