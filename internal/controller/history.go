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
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	chaosv1alpha1 "github.com/neogan74/k8s-chaos/api/v1alpha1"
	chaosmetrics "github.com/neogan74/k8s-chaos/internal/metrics"
)

// HistoryConfig holds configuration for history recording
type HistoryConfig struct {
	Enabled        bool
	Namespace      string
	RetentionLimit int
	RetentionTTL   time.Duration
	SamplingRate   int // Record every Nth execution (1 = all, 10 = every 10th)
}

// DefaultHistoryConfig returns default history configuration
func DefaultHistoryConfig() HistoryConfig {
	return HistoryConfig{
		Enabled:        true,
		Namespace:      "chaos-system",
		RetentionLimit: 100,
		RetentionTTL:   30 * 24 * time.Hour, // 30 days
		SamplingRate:   1,                   // Record all executions
	}
}

// createHistoryRecord creates an immutable history record for an experiment execution
func (r *ChaosExperimentReconciler) createHistoryRecord(
	ctx context.Context,
	exp *chaosv1alpha1.ChaosExperiment,
	executionStatus string,
	affectedResources []chaosv1alpha1.ResourceReference,
	startTime time.Time,
	errorDetails *chaosv1alpha1.ErrorDetails,
) error {
	log := ctrl.LoggerFrom(ctx)

	// Check if history recording is enabled
	if !r.HistoryConfig.Enabled {
		log.V(1).Info("History recording is disabled, skipping")
		return nil
	}

	// Generate unique name for history record
	timestamp := time.Now().Format("20060102-150405")
	historyName := fmt.Sprintf("%s-%s-%s", exp.Name, timestamp, generateShortUID())

	endTime := metav1.Now()
	duration := time.Since(startTime)

	// Build history record
	// Determine history namespace (use configured namespace or experiment namespace as fallback)
	historyNamespace := r.HistoryConfig.Namespace
	if historyNamespace == "" {
		historyNamespace = exp.Namespace
	}

	history := &chaosv1alpha1.ChaosExperimentHistory{
		ObjectMeta: metav1.ObjectMeta{
			Name:      historyName,
			Namespace: historyNamespace,
			Labels: map[string]string{
				"chaos.gushchin.dev/experiment":       exp.Name,
				"chaos.gushchin.dev/action":           exp.Spec.Action,
				"chaos.gushchin.dev/target-namespace": exp.Spec.Namespace,
				"chaos.gushchin.dev/status":           executionStatus,
			},
		},
		Spec: chaosv1alpha1.ChaosExperimentHistorySpec{
			ExperimentRef: chaosv1alpha1.ObjectReference{
				Name:      exp.Name,
				Namespace: exp.Namespace,
				UID:       string(exp.UID),
			},
			ExperimentSpec: exp.Spec,
			Execution: chaosv1alpha1.ExecutionDetails{
				StartTime: metav1.NewTime(startTime),
				EndTime:   &endTime,
				Duration:  duration.String(),
				Status:    executionStatus,
				Message:   exp.Status.Message,
				Phase:     exp.Status.Phase,
			},
			AffectedResources: affectedResources,
			Audit: chaosv1alpha1.AuditMetadata{
				InitiatedBy:        getInitiator(ctx),
				ScheduledExecution: exp.Spec.Schedule != "",
				DryRun:             exp.Spec.DryRun,
				RetryCount:         exp.Status.RetryCount,
				CreationTimestamp:  metav1.Now(),
			},
			Error: errorDetails,
		},
	}

	// Create the history record
	if err := r.Create(ctx, history); err != nil {
		log.Error(err, "Failed to create history record",
			"experiment", exp.Name,
			"historyName", historyName)
		return fmt.Errorf("failed to create history record: %w", err)
	}

	log.Info("Created history record",
		"experiment", exp.Name,
		"historyName", historyName,
		"status", executionStatus,
		"affectedResources", len(affectedResources))

	// Record metrics for history creation
	chaosmetrics.HistoryRecordsTotal.WithLabelValues(exp.Spec.Action, executionStatus).Inc()

	// Trigger retention cleanup asynchronously
	go r.cleanupOldHistoryRecords(context.Background(), exp)

	return nil
}

// cleanupOldHistoryRecords removes old history records based on retention policy
func (r *ChaosExperimentReconciler) cleanupOldHistoryRecords(
	ctx context.Context,
	exp *chaosv1alpha1.ChaosExperiment,
) {
	log := ctrl.LoggerFrom(ctx)

	// Get retention limit from config
	retentionLimit := r.HistoryConfig.RetentionLimit
	if retentionLimit <= 0 {
		retentionLimit = 100 // Default fallback
	}

	// List all history records for this experiment
	historyList := &chaosv1alpha1.ChaosExperimentHistoryList{}
	err := r.List(ctx, historyList,
		client.InNamespace(exp.Namespace),
		client.MatchingLabels{
			"chaos.gushchin.dev/experiment": exp.Name,
		})
	if err != nil {
		log.Error(err, "Failed to list history records for cleanup")
		return
	}

	// If under limit, nothing to clean up
	if len(historyList.Items) <= retentionLimit {
		return
	}

	// Sort by creation timestamp (oldest first)
	sortHistoryByAge(historyList.Items)

	// Delete oldest records exceeding the limit
	recordsToDelete := len(historyList.Items) - retentionLimit
	deletedCount := 0
	for i := 0; i < recordsToDelete && i < len(historyList.Items); i++ {
		record := &historyList.Items[i]
		log.Info("Deleting old history record due to retention policy",
			"record", record.Name,
			"age", time.Since(record.CreationTimestamp.Time))

		if err := r.Delete(ctx, record); err != nil {
			log.Error(err, "Failed to delete old history record", "record", record.Name)
		} else {
			deletedCount++
			// Record cleanup metric
			chaosmetrics.HistoryCleanupTotal.WithLabelValues("retention_limit").Inc()
		}
	}

	if deletedCount > 0 {
		log.Info("Cleaned up old history records",
			"experiment", exp.Name,
			"deletedCount", deletedCount,
			"retentionLimit", retentionLimit)
	}
}

// sortHistoryByAge sorts history records by creation timestamp (oldest first)
func sortHistoryByAge(items []chaosv1alpha1.ChaosExperimentHistory) {
	// Simple bubble sort (sufficient for typical history sizes)
	// For large datasets, consider using sort.Slice
	for i := 0; i < len(items)-1; i++ {
		for j := 0; j < len(items)-i-1; j++ {
			if items[j].CreationTimestamp.After(items[j+1].CreationTimestamp.Time) {
				items[j], items[j+1] = items[j+1], items[j]
			}
		}
	}
}

// buildResourceReferences creates ResourceReference objects from pod names
func buildResourceReferences(action string, namespace string, resourceNames []string, kind string) []chaosv1alpha1.ResourceReference {
	refs := make([]chaosv1alpha1.ResourceReference, 0, len(resourceNames))
	for _, name := range resourceNames {
		refs = append(refs, chaosv1alpha1.ResourceReference{
			Kind:      kind,
			Name:      name,
			Namespace: namespace,
			Action:    action,
		})
	}
	return refs
}

// getInitiator extracts the user/service account that initiated the request
func getInitiator(ctx context.Context) string {
	// TODO: Extract from request context
	// For now, return controller service account
	return "system:serviceaccount:chaos-system:chaos-controller"
}

// generateShortUID generates a short unique identifier for history records
func generateShortUID() string {
	// Simple implementation - use current nanosecond timestamp
	// In production, might want to use UUID or more sophisticated approach
	return fmt.Sprintf("%x", time.Now().UnixNano()%0xffffff)
}

// Example integration in handler functions:
//
// func (r *ChaosExperimentReconciler) handlePodKill(ctx context.Context, exp *chaosv1alpha1.ChaosExperiment) (ctrl.Result, error) {
//     startTime := time.Now()
//     // ... existing logic ...
//
//     // Build resource references
//     affectedResources := buildResourceReferences("deleted", exp.Spec.Namespace, killedPods, "Pod")
//
//     // Create history record
//     if err := r.createHistoryRecord(ctx, exp, statusSuccess, affectedResources, startTime, nil); err != nil {
//         log.Error(err, "Failed to create history record")
//         // Don't fail the experiment if history recording fails
//     }
//
//     return ctrl.Result{RequeueAfter: time.Minute}, nil
// }
