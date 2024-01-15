/*
Copyright 2024.

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
	"fmt"
	mydomainv1 "operator/api/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	metrics "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// BankInstanceReconciler reconciles a BankInstance object
type BankInstanceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=my.domain,resources=bankinstances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=my.domain,resources=bankinstances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=my.domain,resources=bankinstances/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the BankInstance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *BankInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var pod corev1.Pod
	var podmetrics metrics.PodMetrics
	var nodemetrics metrics.NodeMetrics

	r.Client.Get(ctx, req.NamespacedName, &pod)
	fmt.Println("**********************************************")
	if pod.Name == "" {
		fmt.Println("POD: " + req.NamespacedName.Name + " has been deleted")
	} else {
		r.Client.Get(ctx, req.NamespacedName, &podmetrics)
		r.Client.Get(ctx, types.NamespacedName{Namespace: "", Name: pod.Spec.NodeName}, &nodemetrics)

		fmt.Println("POD: " + pod.Name + ", Phase: " + string(pod.Status.Phase))

		for _, condition := range pod.Status.Conditions {

			fmt.Println("	condition: " + condition.Type + ", status: " + corev1.PodConditionType(condition.Status) + ", message: " + corev1.PodConditionType(condition.Message))
		}

		var cpuUsage float64
		var memoryUsage float64
		var cpuUsagePerc float64
		var memoryUsagePerc float64

		fmt.Println("CONTAINERS:")

		for _, container := range podmetrics.Containers {
			fmt.Println("	-container: " + container.Name)
			fmt.Println("		-CPU: " + container.Usage.Cpu().String())
			fmt.Println("		-Memory: " + container.Usage.Memory().String())
			cpuUsage = cpuUsage + container.Usage.Cpu().ToDec().AsApproximateFloat64()
			memoryUsage = memoryUsage + container.Usage.Memory().AsApproximateFloat64()
		}

		cpuUsagePerc = (cpuUsage / nodemetrics.Usage.Cpu().AsApproximateFloat64()) * 100
		memoryUsagePerc = (memoryUsage / nodemetrics.Usage.Memory().ToDec().AsApproximateFloat64()) * 100
		fmt.Println("NODE: " + nodemetrics.Name)
		fmt.Println("		-CPU: " + nodemetrics.Usage.Cpu().String())
		fmt.Println("		-Memory: " + nodemetrics.Usage.Memory().String())

		fmt.Println("USAGE: ")
		fmt.Printf("	-CPU: %v %%\n", cpuUsagePerc)
		fmt.Printf("	-Memory: %v %%\n", memoryUsagePerc)
	}

	fmt.Println("********************************************")

	return ctrl.Result{}, nil
}

func namespaceFilters(namespace string) predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return e.Object.GetNamespace() == namespace
		},
		UpdateFunc: func(e event.UpdateEvent) bool {

			return e.ObjectOld.GetNamespace() == namespace || e.ObjectNew.GetNamespace() == namespace
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return e.Object.GetNamespace() == namespace
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return e.Object.GetNamespace() == namespace
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *BankInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mydomainv1.BankInstance{}).
		Watches(&corev1.Pod{}, &handler.EnqueueRequestForObject{}).
		Watches(&metrics.PodMetrics{}, &handler.EnqueueRequestForObject{}).
		WithEventFilter(namespaceFilters("bank")).
		Complete(r)
}
