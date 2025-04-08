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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=tcproutes,verbs=get;list;watch;create;update;patch;delete

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
	log := log.FromContext(ctx)

	serverDefinition := &serversv1alpha1.McServer{}
	err := r.Get(ctx, req.NamespacedName, serverDefinition)
	if err != nil {
		return ctrl.Result{}, nil
	}

	deployment := r.createDeployment(serverDefinition)
	if err := controllerutil.SetControllerReference(serverDefinition, deployment, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	currentDeployment := &appsv1.Deployment{}
	err = r.Get(ctx, client.ObjectKeyFromObject(deployment), currentDeployment)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Deployment not found creating new one")
		err := r.Create(ctx, deployment)
		if err != nil {
			log.Error(err, "Failed to create new deployment")
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get server deployment")
		return ctrl.Result{}, nil
	}

	service := r.createService(serverDefinition)
	if err := controllerutil.SetControllerReference(serverDefinition, service, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	currentService := &corev1.Service{}
	err = r.Get(ctx, client.ObjectKeyFromObject(service), currentService)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Service not found creating new one")
		err := r.Create(ctx, service)
		if err != nil {
			log.Error(err, "Failed to create new service")
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get server service")
		return ctrl.Result{}, nil
	}

	route := r.createRoute(serverDefinition)
	if err := controllerutil.SetControllerReference(serverDefinition, route, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	currentRoute := &networkingv1alpha2.TCPRoute{}
	err = r.Get(ctx, client.ObjectKeyFromObject(route), currentRoute)
	if err != nil && errors.IsNotFound(err) {
		log.Info("TCPRoute not found creating new one")
		err := r.Create(ctx, route)
		if err != nil {
			log.Error(err, "Failed to create new TCPRoute")
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get server TCPRoute")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *McServerReconciler) createDeployment(McServer *serversv1alpha1.McServer) *appsv1.Deployment {
	var envs []corev1.EnvVar
	for key, value := range McServer.Spec.Env {
		envs = append(envs, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      McServer.Spec.Name,
			Namespace: "minecraft-server",
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
							Resources: corev1.ResourceRequirements{
								Requests: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    resource.MustParse("3"),
									corev1.ResourceMemory: resource.MustParse("7000Mi"),
								},
								Limits: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    resource.MustParse("4"),
									corev1.ResourceMemory: resource.MustParse("7000Mi"),
								},
							},
						},
					},
				},
			},
		},
	}
	return deployment
}

func (r *McServerReconciler) createService(McServer *serversv1alpha1.McServer) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      McServer.Spec.Name,
			Namespace: "minecraft-server",
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

func (r *McServerReconciler) createRoute(McServer *serversv1alpha1.McServer) *networkingv1alpha2.TCPRoute {
	route := &networkingv1alpha2.TCPRoute{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "gateway.networking.k8s.io/v1alpha2",
			Kind:       "TCPRoute",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      McServer.Spec.RouteName,
			Namespace: "minecraft-server",
			Labels: map[string]string{
				"app": McServer.Spec.RouteName,
			},
		},
		Spec: networkingv1alpha2.TCPRouteSpec{
			CommonRouteSpec: networkingv1alpha2.CommonRouteSpec{
				ParentRefs: []networkingv1alpha2.ParentReference{{
					Name:        "envoy-gateway",
					Namespace:   ptr.To(networkingv1alpha2.Namespace("infra")),
					SectionName: ptr.To(networkingv1alpha2.SectionName(McServer.Spec.RouteName)),
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
