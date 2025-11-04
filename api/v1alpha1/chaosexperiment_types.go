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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	// ExclusionLabel is the label that protects resources from chaos experiments
	ExclusionLabel = "chaos.gushchin.dev/exclude"

	// ProductionAnnotation marks a namespace as production
	ProductionAnnotation = "chaos.gushchin.dev/production"

	// ProductionLabel alternative way to mark namespaces as production
	ProductionLabel = "environment"

	// ProductionLabelValue for environment label
	ProductionLabelValue = "production"
)

// ChaosExperimentSpec defines the desired state of ChaosExperiment
type ChaosExperimentSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// Action specifies the chaos action to perform
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=pod-kill;pod-delay;node-drain;pod-cpu-stress;pod-memory-stress
	Action string `json:"action"`

	// Namespace specifies the target namespace for chaos experiments
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace"`

	// Selector specifies the label selector for target resources
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinProperties=1
	Selector map[string]string `json:"selector"`

	// Count specifies the number of resources to affect
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=1
	// +optional
	Count int `json:"count,omitempty"`

	// Duration specifies how long the chaos action should last (for pod-delay)
	// +kubebuilder:validation:Pattern="^([0-9]+(s|m|h))+$"
	// +optional
	Duration string `json:"duration,omitempty"`

	// ExperimentDuration specifies how long the entire experiment should run before auto-stopping
	// If not set, the experiment runs indefinitely until manually stopped
	// +kubebuilder:validation:Pattern="^([0-9]+(s|m|h))+$"
	// +optional
	ExperimentDuration string `json:"experimentDuration,omitempty"`

	// MaxRetries specifies the maximum number of retry attempts for failed experiments
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=10
	// +kubebuilder:default=3
	// +optional
	MaxRetries int `json:"maxRetries,omitempty"`

	// RetryBackoff specifies the backoff strategy for retries (exponential or fixed)
	// +kubebuilder:validation:Enum=exponential;fixed
	// +kubebuilder:default=exponential
	// +optional
	RetryBackoff string `json:"retryBackoff,omitempty"`

	// RetryDelay specifies the initial delay between retries (e.g., "30s", "1m")
	// +kubebuilder:validation:Pattern="^([0-9]+(s|m|h))+$"
	// +kubebuilder:default="30s"
	// +optional
	RetryDelay string `json:"retryDelay,omitempty"`

	// CPULoad specifies the CPU load percentage for pod-cpu-stress action
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +optional
	CPULoad int `json:"cpuLoad,omitempty"`

	// CPUWorkers specifies the number of CPU workers for pod-cpu-stress action
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=32
	// +kubebuilder:default=1
	// +optional
	CPUWorkers int `json:"cpuWorkers,omitempty"`

	// MemorySize specifies the amount of memory to consume (for pod-memory-stress)
	// Format: "256M", "1G", "512M", etc.
	// +kubebuilder:validation:Pattern="^[0-9]+[MG]$"
	// +optional
	MemorySize string `json:"memorySize,omitempty"`

	// MemoryWorkers specifies the number of memory workers (for pod-memory-stress)
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=8
	// +kubebuilder:default=1
	// +optional
	MemoryWorkers int `json:"memoryWorkers,omitempty"`

	// DryRun enables preview mode without executing actual chaos
	// +optional
	DryRun bool `json:"dryRun,omitempty"`

	// MaxPercentage limits the maximum percentage of resources to affect
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +optional
	MaxPercentage int `json:"maxPercentage,omitempty"`

	// AllowProduction explicitly permits chaos experiments in production namespaces
	// +optional
	AllowProduction bool `json:"allowProduction,omitempty"`
}

// ChaosExperimentStatus defines the observed state of ChaosExperiment.
type ChaosExperimentStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// LastRunTime indicates when the experiment was last executed
	// +optional
	LastRunTime *metav1.Time `json:"lastRunTime,omitempty"`

	// Message provides human-readable status information
	// +optional
	Message string `json:"message,omitempty"`

	// Phase represents the current state of the experiment
	// +kubebuilder:validation:Enum=Pending;Running;Completed;Failed
	// +optional
	Phase string `json:"phase,omitempty"`

	// RetryCount tracks the current number of retry attempts
	// +optional
	RetryCount int `json:"retryCount,omitempty"`

	// LastError stores the last error message encountered
	// +optional
	LastError string `json:"lastError,omitempty"`

	// NextRetryTime indicates when the next retry will be attempted
	// +optional
	NextRetryTime *metav1.Time `json:"nextRetryTime,omitempty"`

	// StartTime indicates when the experiment started running
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletedAt indicates when the experiment completed (either by duration or manually)
	// +optional
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Action",type="string",JSONPath=".spec.action"
// +kubebuilder:printcolumn:name="Namespace",type="string",JSONPath=".spec.namespace"
// +kubebuilder:printcolumn:name="Count",type="integer",JSONPath=".spec.count"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Retries",type="integer",JSONPath=".status.retryCount"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// ChaosExperiment is the Schema for the chaosexperiments API
type ChaosExperiment struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of ChaosExperiment
	// +required
	Spec ChaosExperimentSpec `json:"spec"`

	// status defines the observed state of ChaosExperiment
	// +optional
	Status ChaosExperimentStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// ChaosExperimentList contains a list of ChaosExperiment
type ChaosExperimentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ChaosExperiment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ChaosExperiment{}, &ChaosExperimentList{})
}
