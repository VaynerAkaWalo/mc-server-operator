package controller

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	serversv1alpha1 "github.com/VaynerAkaWalo/mc-server-operator/api/v1alpha1"
)

func fakeReconciler(t *testing.T, saves SaveConfig, objects ...client.Object) *McServerReconciler {
	t.Helper()

	return fakeReconcilerWith(t, saves, interceptor.Funcs{}, objects...)
}

func fakeReconcilerWith(t *testing.T, saves SaveConfig, funcs interceptor.Funcs, objects ...client.Object) *McServerReconciler {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := serversv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	cluster := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).WithInterceptorFuncs(funcs).Build()

	return &McServerReconciler{Client: cluster, Scheme: scheme, Saves: saves}
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

func terminatingDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Name:       "test-server",
		Namespace:  serverNamespace,
		Finalizers: []string{metav1.FinalizerDeleteDependents},
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

func deletingServer() *serversv1alpha1.McServer {
	deleting := storedServer("test-server", saveFinalizer)
	deleting.DeletionTimestamp = &metav1.Time{Time: metav1.Now().Time}

	return deleting
}

func TestDeletedServerStaysUntilItsWorkloadIsGoneSoTheFinalSnapshotCanFinish(t *testing.T) {
	reconciler := fakeReconciler(t, testSaves(), deletingServer(), terminatingDeployment())

	result, err := reconciler.Reconcile(context.Background(), request())
	if err != nil {
		t.Fatal(err)
	}
	if result.RequeueAfter != finalizeRetryInterval {
		t.Errorf("requeue = %v, want %v", result.RequeueAfter, finalizeRetryInterval)
	}

	stored := &serversv1alpha1.McServer{}
	if err := reconciler.Get(context.Background(), request().NamespacedName, stored); err != nil {
		t.Fatalf("server disappeared while its workload was still terminating: %v", err)
	}
}

func TestServerWorkloadIsDeletedInTheForegroundSoPodsAreGoneBeforeItDisappears(t *testing.T) {
	var policy *metav1.DeletionPropagation
	observe := interceptor.Funcs{
		Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
			options := &client.DeleteOptions{}
			options.ApplyOptions(opts)
			policy = options.PropagationPolicy

			return c.Delete(ctx, obj, opts...)
		},
	}
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "test-server", Namespace: serverNamespace}}
	reconciler := fakeReconcilerWith(t, testSaves(), observe, deletingServer(), deployment)

	if _, err := reconciler.Reconcile(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
	if policy == nil || *policy != metav1.DeletePropagationForeground {
		t.Errorf("propagation = %v, want Foreground", policy)
	}
}

func TestDeletedServerIsReleasedOnceItsWorkloadIsGone(t *testing.T) {
	reconciler := fakeReconciler(t, testSaves(), deletingServer())

	if _, err := reconciler.Reconcile(context.Background(), request()); err != nil {
		t.Fatal(err)
	}

	err := reconciler.Get(context.Background(), request().NamespacedName, &serversv1alpha1.McServer{})
	if !errors.IsNotFound(err) {
		t.Errorf("server still exists after its workload is gone: %v", err)
	}
}
