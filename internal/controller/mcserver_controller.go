/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/gateway-api/apis/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	serversv1alpha1 "github.com/VaynerAkaWalo/mc-server-operator/api/v1alpha1"
	networkingv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// McServerReconciler reconciles a McServer object
type McServerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=servers.blamedevs.com,resources=mcservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=servers.blamedevs.com,resources=mcservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=servers.blamedevs.com,resources=mcservers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the McServer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *McServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	serverController := &serversv1alpha1.McServer{}
	err := r.Get(ctx, req.NamespacedName, serverController)
	if err != nil {
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *McServerReconciler) createDeployment(McServer serversv1alpha1.McServer) *appsv1.Deployment {
	var envs []corev1.EnvVar
	for key, value := range McServer.Spec.Env {
		envs = append(envs, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: McServer.Spec.Name,
			Labels: map[string]string{
				"app": McServer.Spec.Name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": McServer.Spec.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": McServer.Spec.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "minecraft-server",
							Image: McServer.Spec.Image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: McServer.Spec.Port,
								},
							},
							Env: envs,
						},
					},
				},
			},
		},
	}
	return deployment
}

func (r *McServerReconciler) createService(McServer serversv1alpha1.McServer) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: McServer.Spec.Name,
			Labels: map[string]string{
				"app": McServer.Spec.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": McServer.Spec.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Protocol: corev1.ProtocolTCP,
					Port:     McServer.Spec.Port,
				},
			},
			Type: corev1.ServiceTypeNodePort,
		},
	}
	return service
}

func (r *McServerReconciler) createRoute(McServer serversv1alpha1.McServer) *networkingv1alpha2.TCPRoute {
	route := &networkingv1alpha2.TCPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name: McServer.Spec.Name,
			Labels: map[string]string{
				"app": McServer.Spec.Name,
			},
		},
		Spec: networkingv1alpha2.TCPRouteSpec{
			CommonRouteSpec: networkingv1alpha2.CommonRouteSpec{
				ParentRefs: []networkingv1alpha2.ParentReference{{
					Name:      "envoy-gateway",
					Namespace: ptr.To(networkingv1alpha2.Namespace("infra")),
				}},
			},
			Rules: []networkingv1alpha2.TCPRouteRule{{
				BackendRefs: []networkingv1alpha2.BackendRef{{
					BackendObjectReference: networkingv1alpha2.BackendObjectReference{
						Name: v1.ObjectName(McServer.Spec.Name),
						Port: ptr.To(networkingv1alpha2.PortNumber(McServer.Spec.Port)),
					},
				}},
			}},
		},
	}
	return route
}

// SetupWithManager sets up the controller with the Manager.
func (r *McServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&serversv1alpha1.McServer{}).
		Complete(r)
}
