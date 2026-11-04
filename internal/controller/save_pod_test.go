package controller

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"

	serversv1alpha1 "github.com/VaynerAkaWalo/mc-server-operator/api/v1alpha1"
)

const testSaveID = "11111111-2222-3333-4444-555555555555"

func testSaves() SaveConfig {
	return SaveConfig{
		Endpoint:      "https://ns.compat.objectstorage.eu-frankfurt-1.oraclecloud.com",
		Bucket:        "production-mc-saves",
		Region:        "eu-frankfurt-1",
		SecretName:    "mc-saves",
		Image:         "ghcr.io/vaynerakawalo/mc-saver:latest",
		PullSecret:    "ghcr-pull",
		Interval:      5 * time.Minute,
		KeepSnapshots: 10,
	}
}

func savedServer() *serversv1alpha1.McServer {
	return &serversv1alpha1.McServer{
		Spec: serversv1alpha1.McServerSpec{
			Name:   "test-server",
			Image:  "itzg/minecraft-server:latest",
			Memory: "3000Mi",
			Save:   &serversv1alpha1.SaveSpec{ID: testSaveID},
		},
	}
}

func TestSavedServerRestoresBeforeStartAndKeepsSavingAlongsideIt(t *testing.T) {
	pod := (&McServerReconciler{Saves: testSaves()}).createDeployment(savedServer()).Spec.Template.Spec

	if len(pod.InitContainers) != 2 || pod.InitContainers[0].Name != "restore" || pod.InitContainers[1].Name != "saver" {
		t.Fatalf("init containers = %v, want restore then saver", pod.InitContainers)
	}

	saver := pod.InitContainers[1]
	if saver.RestartPolicy == nil || *saver.RestartPolicy != corev1.ContainerRestartPolicyAlways {
		t.Error("saver is not a native sidecar, so it would not outlive the server during shutdown")
	}
}

func TestSavedServerKeepsBucketCredentialsAwayFromTheServerContainer(t *testing.T) {
	pod := (&McServerReconciler{Saves: testSaves()}).createDeployment(savedServer()).Spec.Template.Spec

	if len(pod.Containers[0].EnvFrom) != 0 {
		t.Error("server container receives the saves secret")
	}

	for _, container := range pod.InitContainers {
		if len(container.EnvFrom) != 1 || container.EnvFrom[0].SecretRef.Name != "mc-saves" {
			t.Errorf("%s does not receive the saves secret", container.Name)
		}
	}
}

func TestSavedServerUsesRepositoryOfItsSaveAndSharesTheWorldVolume(t *testing.T) {
	pod := (&McServerReconciler{Saves: testSaves()}).createDeployment(savedServer()).Spec.Template.Spec

	want := "s3:https://ns.compat.objectstorage.eu-frankfurt-1.oraclecloud.com/production-mc-saves/saves/" + testSaveID
	for _, container := range pod.InitContainers {
		if got := envValue(container, "RESTIC_REPOSITORY"); got != want {
			t.Errorf("%s repository = %s, want %s", container.Name, got, want)
		}
		if !mounts(container, "data", "/data") {
			t.Errorf("%s does not mount the world volume", container.Name)
		}
	}

	if !mounts(pod.Containers[0], "data", "/data") {
		t.Error("server does not mount the world volume")
	}
}

func TestSavedServerGetsTimeToUploadTheFinalSnapshot(t *testing.T) {
	spec := (&McServerReconciler{Saves: testSaves()}).createDeployment(savedServer()).Spec.Template.Spec

	if spec.TerminationGracePeriodSeconds == nil || *spec.TerminationGracePeriodSeconds != saveGracePeriodSeconds {
		t.Errorf("grace period = %v, want %d", spec.TerminationGracePeriodSeconds, saveGracePeriodSeconds)
	}
	if len(spec.ImagePullSecrets) != 1 || spec.ImagePullSecrets[0].Name != "ghcr-pull" {
		t.Errorf("pull secrets = %v, want ghcr-pull", spec.ImagePullSecrets)
	}
}

func TestServerWithoutSaveGetsNoSaveContainers(t *testing.T) {
	server := savedServer()
	server.Spec.Save = nil
	server.Spec.Volume = "scratch"

	pod := (&McServerReconciler{Saves: testSaves()}).createDeployment(server).Spec.Template.Spec

	if len(pod.InitContainers) != 0 || pod.TerminationGracePeriodSeconds != nil || len(pod.ImagePullSecrets) != 0 {
		t.Error("server without a save is shaped for saving")
	}
	if !mounts(pod.Containers[0], "scratch", "/data") {
		t.Error("volume of a server without a save changed")
	}
}

func envValue(container corev1.Container, name string) string {
	for _, env := range container.Env {
		if env.Name == name {
			return env.Value
		}
	}

	return ""
}

func mounts(container corev1.Container, volume string, path string) bool {
	for _, mount := range container.VolumeMounts {
		if mount.Name == volume && mount.MountPath == path {
			return true
		}
	}

	return false
}
