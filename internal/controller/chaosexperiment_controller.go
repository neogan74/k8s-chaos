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
	"math/rand"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	chaosv1alpha1 "github.com/neogan74/k8s-chaos/api/v1alpha1"
)

// ChaosExperimentReconciler reconciles a ChaosExperiment object
type ChaosExperimentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=chaos.gushchin.dev,resources=chaosexperiments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chaos.gushchin.dev,resources=chaosexperiments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chaos.gushchin.dev,resources=chaosexperiments/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ChaosExperiment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *ChaosExperimentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	var exp chaosv1alpha1.ChaosExperiment
	if err := r.Get(ctx, req.NamespacedName, &exp); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if exp.Spec.Action == "pod-kill" {
		log.Info("Unsupported action", "action", exp.Spec.Action)
		return ctrl.Result{}, nil
	}

	// Choose Pods by selector
	podList := &corev1.PodList{}
	selector := labels.SelectorFromSet(exp.Spec.Selector)
	if err := r.List(ctx, podList, client.InNamespace(exp.Spec.Namespace),
		client.MatchingLabelsSelector{Selector: selector}); err != nil {
		return ctrl.Result{}, err
	}

	if len(podList.Items) == 0 {
		log.Info("No pods found for selector", "selector", exp.Spec.Selector)
		return ctrl.Result{}, nil
	}

	// Перемешаем список Pod-ов
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(podList.Items), func(i, j int) {
		podList.Items[i], podList.Items[j] = podList.Items[j], podList.Items[i]
	})

	// Удалим нужное количество Pod-ов
	killCount := exp.Spec.Count
	if killCount > len(podList.Items) {
		killCount = len(podList.Items)
	}

	for i := 0; i < killCount; i++ {
		pod := podList.Items[i]
		log.Info("Deleting pod", "pod", pod.Name)
		if err := r.Delete(ctx, &pod); err != nil {
			log.Error(err, "Failed to delete pod", "pod", pod.Name)
		}
	}

	// Update status
	now := metav1.Now()
	exp.Status.LastRunTime = &now
	exp.Status.Message = "Killed pods"
	if err := r.Status().Update(ctx, &exp); err != nil {
		log.Error(err, "Failed to update ChaosExperiment status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChaosExperimentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1alpha1.ChaosExperiment{}).
		Named("chaosexperiment").
		Complete(r)
}
