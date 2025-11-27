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

// ChaosExperimentHistorySpec defines the historical record of a chaos experiment execution
type ChaosExperimentHistorySpec struct {
	// ExperimentRef references the original ChaosExperiment resource
	// +kubebuilder:validation:Required
	ExperimentRef ObjectReference `json:"experimentRef"`

	// ExperimentSpec captures the experiment configuration at execution time
	// +kubebuilder:validation:Required
	ExperimentSpec ChaosExperimentSpec `json:"experimentSpec"`

	// Execution contains details about the experiment execution
	// +kubebuilder:validation:Required
	Execution ExecutionDetails `json:"execution"`

	// AffectedResources lists all resources that were affected by this execution
	// +optional
	AffectedResources []ResourceReference `json:"affectedResources,omitempty"`

	// Audit contains metadata for compliance and auditing
	// +kubebuilder:validation:Required
	Audit AuditMetadata `json:"audit"`

	// Error contains error information if the execution failed
	// +optional
	Error *ErrorDetails `json:"error,omitempty"`
}

// ObjectReference contains information to locate a Kubernetes object
type ObjectReference struct {
	// Name of the referenced object
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Namespace of the referenced object
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace"`

	// UID of the referenced object
	// +optional
	UID string `json:"uid,omitempty"`
}

// ExecutionDetails captures information about a single experiment execution
type ExecutionDetails struct {
	// StartTime is when the experiment execution began
	// +kubebuilder:validation:Required
	StartTime metav1.Time `json:"startTime"`

	// EndTime is when the experiment execution completed
	// +optional
	EndTime *metav1.Time `json:"endTime,omitempty"`

	// Duration is the total execution time (e.g., "3.5s", "2m")
	// +optional
	Duration string `json:"duration,omitempty"`

	// Status indicates the outcome of the execution
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=success;failure;partial;cancelled
	Status string `json:"status"`

	// Message provides human-readable status information
	// +optional
	Message string `json:"message,omitempty"`

	// Phase is the experiment phase during execution
	// +kubebuilder:validation:Enum=Pending;Running;Completed;Failed
	// +optional
	Phase string `json:"phase,omitempty"`
}

// ResourceReference identifies a Kubernetes resource affected by an experiment
type ResourceReference struct {
	// Kind of the resource (e.g., Pod, Node)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Kind string `json:"kind"`

	// Name of the resource
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Namespace of the resource (empty for cluster-scoped resources)
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Action performed on the resource (e.g., deleted, delayed, stressed)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Action string `json:"action"`

	// Details provides additional information about the action
	// +optional
	Details string `json:"details,omitempty"`
}

// AuditMetadata contains information for compliance and auditing purposes
type AuditMetadata struct {
	// InitiatedBy identifies who or what triggered the experiment
	// Typically a ServiceAccount for scheduled experiments or User for manual triggers
	// +optional
	InitiatedBy string `json:"initiatedBy,omitempty"`

	// ScheduledExecution indicates if this was triggered by a schedule (true) or manual (false)
	// +optional
	ScheduledExecution bool `json:"scheduledExecution,omitempty"`

	// DryRun indicates if this was a dry-run execution
	// +optional
	DryRun bool `json:"dryRun,omitempty"`

	// RetryCount indicates which retry attempt this was (0 for first attempt)
	// +optional
	RetryCount int `json:"retryCount,omitempty"`

	// CreationTimestamp is when the history record was created
	// +optional
	CreationTimestamp metav1.Time `json:"creationTimestamp,omitempty"`
}

// ErrorDetails contains information about execution failures
type ErrorDetails struct {
	// Message is the error message
	// +optional
	Message string `json:"message,omitempty"`

	// Code is an optional error code
	// +optional
	Code string `json:"code,omitempty"`

	// LastError is the last error encountered during retries
	// +optional
	LastError string `json:"lastError,omitempty"`

	// FailureReason categorizes the type of failure
	// +kubebuilder:validation:Enum=ValidationError;ResourceNotFound;PermissionDenied;ExecutionError;Timeout;Unknown
	// +optional
	FailureReason string `json:"failureReason,omitempty"`
}

// ChaosExperimentHistoryStatus defines the observed state of ChaosExperimentHistory
// Note: History records are immutable, so status is minimal
type ChaosExperimentHistoryStatus struct {
	// Archived indicates if this record has been archived to external storage
	// +optional
	Archived bool `json:"archived,omitempty"`

	// ArchiveLocation is the external storage location if archived
	// +optional
	ArchiveLocation string `json:"archiveLocation,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=cehist;chaoshist
// +kubebuilder:printcolumn:name="Experiment",type="string",JSONPath=".spec.experimentRef.name"
// +kubebuilder:printcolumn:name="Action",type="string",JSONPath=".spec.experimentSpec.action"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".spec.execution.status"
// +kubebuilder:printcolumn:name="Duration",type="string",JSONPath=".spec.execution.duration"
// +kubebuilder:printcolumn:name="Resources",type="integer",JSONPath=".spec.affectedResources[*]",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// ChaosExperimentHistory is the Schema for the chaosexperimenthistories API
// It provides an immutable audit log of chaos experiment executions
type ChaosExperimentHistory struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChaosExperimentHistorySpec   `json:"spec"`
	Status ChaosExperimentHistoryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ChaosExperimentHistoryList contains a list of ChaosExperimentHistory
type ChaosExperimentHistoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ChaosExperimentHistory `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ChaosExperimentHistory{}, &ChaosExperimentHistoryList{})
}
