package controller

import (
	"testing"

	serversv1alpha1 "github.com/VaynerAkaWalo/mc-server-operator/api/v1alpha1"
)

func TestServerPodHasNoClusterAccess(t *testing.T) {
	server := &serversv1alpha1.McServer{
		Spec: serversv1alpha1.McServerSpec{
			Name:   "test-server",
			Image:  "itzg/minecraft-server:latest",
			Memory: "3000Mi",
			Volume: "save",
		},
	}

	pod := (&McServerReconciler{}).createDeployment(server).Spec.Template.Spec

	if *pod.AutomountServiceAccountToken || *pod.EnableServiceLinks {
		t.Error("pod exposes cluster credentials or service addresses")
	}
	if pod.DNSPolicy != "None" {
		t.Errorf("pod uses %s DNS policy, want None", pod.DNSPolicy)
	}
	if !*pod.SecurityContext.RunAsNonRoot || *pod.Containers[0].SecurityContext.AllowPrivilegeEscalation {
		t.Error("server container can run privileged")
	}
	if pod.Volumes[0].EmptyDir == nil || pod.Volumes[0].EmptyDir.SizeLimit == nil {
		t.Error("save volume has no size limit")
	}
}
