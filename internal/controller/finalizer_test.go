package controller

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	serversv1alpha1 "github.com/VaynerAkaWalo/mc-server-operator/api/v1alpha1"
)

func fakeReconciler(t *testing.T, saves SaveConfig, objects ...client.Object) *McServerReconciler {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := serversv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	cluster := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()

	return &McServerReconciler{Client: cluster, Scheme: scheme, PodReader: cluster, Saves: saves}
}

func request() ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-server", Namespace: serverNamespace}}
}

func storedServer(name string, finalizers ...string) *serversv1alpha1.McServer {
	server := savedServer()
	server.Name = name
	server.Namespace = serverNamespace
	server.Finalizers = finalizers

	return server
}

func serverPod() *corev1.Pod {
	return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:      "test-server-abc",
		Namespace: serverNamespace,
		Labels:    map[string]string{"app": "test-server"},
	}}
}

func TestSavedServerIsProtectedByFinalizerBeforeAnythingIsCreated(t *testing.T) {
	reconciler := fakeReconciler(t, testSaves(), storedServer("test-server"))

	if _, err := reconciler.Reconcile(context.Background(), request()); err != nil {
		t.Fatal(err)
	}

	stored := &serversv1alpha1.McServer{}
	if err := reconciler.Get(context.Background(), request().NamespacedName, stored); err != nil {
		t.Fatal(err)
	}
	if len(stored.Finalizers) != 1 || stored.Finalizers[0] != saveFinalizer {
		t.Errorf("finalizers = %v, want %s", stored.Finalizers, saveFinalizer)
	}

	deployment := &appsv1.Deployment{}
	key := types.NamespacedName{Name: "test-server", Namespace: serverNamespace}
	if err := reconciler.Get(context.Background(), key, deployment); err != nil {
		t.Errorf("deployment was not created: %v", err)
	}
}

func TestSaveRequestWithoutConfiguredSavesFailsInsteadOfRunningWithoutPersistence(t *testing.T) {
	reconciler := fakeReconciler(t, SaveConfig{}, storedServer("test-server"))

	if _, err := reconciler.Reconcile(context.Background(), request()); err == nil {
		t.Fatal("reconcile succeeded and would have started a server that never saves")
	}

	deployment := &appsv1.Deployment{}
	key := types.NamespacedName{Name: "test-server", Namespace: serverNamespace}
	if err := reconciler.Get(context.Background(), key, deployment); !errors.IsNotFound(err) {
		t.Errorf("deployment exists or lookup failed: %v", err)
	}
}

func TestDeletedServerStaysUntilItsPodsAreGoneSoTheFinalSnapshotCanFinish(t *testing.T) {
	deleting := storedServer("test-server", saveFinalizer)
	deleting.DeletionTimestamp = &metav1.Time{Time: metav1.Now().Time}
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "test-server", Namespace: serverNamespace}}
	reconciler := fakeReconciler(t, testSaves(), deleting, deployment, serverPod())

	result, err := reconciler.Reconcile(context.Background(), request())
	if err != nil {
		t.Fatal(err)
	}
	if result.RequeueAfter != finalizeRetryInterval {
		t.Errorf("requeue = %v, want %v", result.RequeueAfter, finalizeRetryInterval)
	}

	stored := &serversv1alpha1.McServer{}
	if err := reconciler.Get(context.Background(), request().NamespacedName, stored); err != nil {
		t.Fatalf("server disappeared while its pod was still running: %v", err)
	}

	key := types.NamespacedName{Name: "test-server", Namespace: serverNamespace}
	if err := reconciler.Get(context.Background(), key, &appsv1.Deployment{}); !errors.IsNotFound(err) {
		t.Errorf("deployment was not deleted: %v", err)
	}
}

func TestDeletedServerIsReleasedOnceItsPodsAreGone(t *testing.T) {
	deleting := storedServer("test-server", saveFinalizer)
	deleting.DeletionTimestamp = &metav1.Time{Time: metav1.Now().Time}
	reconciler := fakeReconciler(t, testSaves(), deleting)

	if _, err := reconciler.Reconcile(context.Background(), request()); err != nil {
		t.Fatal(err)
	}

	err := reconciler.Get(context.Background(), request().NamespacedName, &serversv1alpha1.McServer{})
	if !errors.IsNotFound(err) {
		t.Errorf("server still exists after its pods are gone: %v", err)
	}
}
