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

package v1alpha1

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var chaosexperimentlog = logf.Log.WithName("chaosexperiment-resource")

// ChaosExperimentWebhook implements webhook.CustomValidator
// +kubebuilder:object:generate=false
type ChaosExperimentWebhook struct {
	Client client.Client
}

// SetupWebhookWithManager sets up the webhook with the Manager.
func (r *ChaosExperiment) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(&ChaosExperimentWebhook{Client: mgr.GetClient()}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-chaos-gushchin-dev-v1alpha1-chaosexperiment,mutating=false,failurePolicy=fail,sideEffects=None,groups=chaos.gushchin.dev,resources=chaosexperiments,verbs=create;update,versions=v1alpha1,name=vchaosexperiment.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &ChaosExperimentWebhook{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (w *ChaosExperimentWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	exp, ok := obj.(*ChaosExperiment)
	if !ok {
		return nil, fmt.Errorf("expected a ChaosExperiment but got a %T", obj)
	}

	chaosexperimentlog.Info("validate create", "name", exp.Name)

	var warnings admission.Warnings

	// Validate namespace exists
	if err := w.validateNamespaceExists(ctx, exp.Spec.Namespace); err != nil {
		return warnings, err
	}

	// Validate selector matches at least one pod
	matchedPods, err := w.validateSelectorEffectiveness(ctx, exp.Spec.Namespace, exp.Spec.Selector)
	if err != nil {
		return warnings, err
	}

	// Warning if count exceeds available pods
	if exp.Spec.Count > len(matchedPods) {
		warnings = append(warnings, fmt.Sprintf(
			"Count (%d) exceeds number of pods matching selector (%d). Experiment will only affect %d pods.",
			exp.Spec.Count, len(matchedPods), len(matchedPods),
		))
	}

	// Validate cross-field constraints
	if err := w.validateCrossFieldConstraints(&exp.Spec); err != nil {
		return warnings, err
	}

	return warnings, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (w *ChaosExperimentWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	exp, ok := newObj.(*ChaosExperiment)
	if !ok {
		return nil, fmt.Errorf("expected a ChaosExperiment but got a %T", newObj)
	}

	chaosexperimentlog.Info("validate update", "name", exp.Name)

	// Perform the same validations as create
	return w.ValidateCreate(ctx, newObj)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (w *ChaosExperimentWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	exp, ok := obj.(*ChaosExperiment)
	if !ok {
		return nil, fmt.Errorf("expected a ChaosExperiment but got a %T", obj)
	}

	chaosexperimentlog.Info("validate delete", "name", exp.Name)

	// No validation needed for delete
	return nil, nil
}

// validateNamespaceExists checks if the target namespace exists
func (w *ChaosExperimentWebhook) validateNamespaceExists(ctx context.Context, namespace string) error {
	ns := &corev1.Namespace{}
	err := w.Client.Get(ctx, types.NamespacedName{Name: namespace}, ns)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("target namespace %q does not exist", namespace)
		}
		return fmt.Errorf("failed to validate namespace existence: %w", err)
	}
	return nil
}

// validateSelectorEffectiveness checks if the selector matches at least one pod
func (w *ChaosExperimentWebhook) validateSelectorEffectiveness(ctx context.Context, namespace string, selector map[string]string) ([]corev1.Pod, error) {
	podList := &corev1.PodList{}
	err := w.Client.List(ctx, podList, client.InNamespace(namespace), client.MatchingLabels(selector))
	if err != nil {
		return nil, fmt.Errorf("failed to list pods with selector: %w", err)
	}

	if len(podList.Items) == 0 {
		return nil, fmt.Errorf("selector does not match any pods in namespace %q", namespace)
	}

	return podList.Items, nil
}

// validateCrossFieldConstraints validates dependencies between fields
func (w *ChaosExperimentWebhook) validateCrossFieldConstraints(spec *ChaosExperimentSpec) error {
	// pod-delay action requires duration
	if spec.Action == "pod-delay" && spec.Duration == "" {
		return fmt.Errorf("duration is required for pod-delay action")
	}

	// pod-cpu-stress action requires duration and cpuLoad
	if spec.Action == "pod-cpu-stress" {
		if spec.Duration == "" {
			return fmt.Errorf("duration is required for pod-cpu-stress action")
		}
		if spec.CPULoad <= 0 {
			return fmt.Errorf("cpuLoad must be specified and greater than 0 for pod-cpu-stress action")
		}
	}

	// Validate duration format if provided
	if spec.Duration != "" {
		if err := ValidateDurationFormat(spec.Duration); err != nil {
			return err
		}
	}

	// Validate experimentDuration format if provided
	if spec.ExperimentDuration != "" {
		if err := ValidateDurationFormat(spec.ExperimentDuration); err != nil {
			return fmt.Errorf("invalid experimentDuration format: %w", err)
		}
	}

	return nil
}
