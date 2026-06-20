package controller

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	chaosv1alpha1 "github.com/neogan74/k8s-chaos/api/v1alpha1"
)

func TestHandleExperimentFailureRetries(t *testing.T) {
	ctx := context.Background()
	exp := &chaosv1alpha1.ChaosExperiment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fail-retry",
			Namespace: "default",
		},
		Spec: chaosv1alpha1.ChaosExperimentSpec{
			Action:     "pod-kill",
			MaxRetries: 2,
		},
	}

	r := newReconcilerWithObjects(t, exp)

	chaosErr := &ChaosError{
		Original: fmt.Errorf("boom"),
		Type:     ErrorTypeExecution,
	}
	result, err := r.handleExperimentFailure(ctx, exp, chaosErr)
	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, result.RequeueAfter) // defaultRetryDelay * 2^0 = 30s

	refreshed := &chaosv1alpha1.ChaosExperiment{}
	require.NoError(t, r.Get(ctx, clientKey(exp), refreshed))
	assert.Equal(t, phasePending, refreshed.Status.Phase)
	assert.Equal(t, 1, refreshed.Status.RetryCount)
	assert.NotNil(t, refreshed.Status.NextRetryTime)
	assert.Contains(t, refreshed.Status.Message, "Retry 1/2")
}

func TestHandleExperimentFailureExhaustsRetries(t *testing.T) {
	ctx := context.Background()
	exp := &chaosv1alpha1.ChaosExperiment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fail-max",
			Namespace: "default",
		},
		Spec: chaosv1alpha1.ChaosExperimentSpec{
			Action:     "pod-kill",
			MaxRetries: 2,
		},
		Status: chaosv1alpha1.ChaosExperimentStatus{
			RetryCount: 2,
		},
	}

	r := newReconcilerWithObjects(t, exp)

	chaosErr := &ChaosError{
		Original: fmt.Errorf("boom"),
		Type:     ErrorTypeExecution,
	}
	result, err := r.handleExperimentFailure(ctx, exp, chaosErr)
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), result.RequeueAfter)

	refreshed := &chaosv1alpha1.ChaosExperiment{}
	require.NoError(t, r.Get(ctx, clientKey(exp), refreshed))
	assert.Equal(t, phaseFailed, refreshed.Status.Phase)
	assert.Nil(t, refreshed.Status.NextRetryTime)
	assert.Contains(t, refreshed.Status.Message, "Failed after 2 retries")
}

func TestHandleDryRunUpdatesStatus(t *testing.T) {
	ctx := context.Background()
	exp := &chaosv1alpha1.ChaosExperiment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dry-run",
			Namespace: "default",
		},
		Spec: chaosv1alpha1.ChaosExperimentSpec{
			Action: "pod-kill",
			Count:  3,
		},
	}

	pods := []corev1.Pod{
		{ObjectMeta: metav1.ObjectMeta{Name: "p1", Namespace: "default"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "p2", Namespace: "default"}},
	}

	r := newReconcilerWithObjects(t, exp)

	result, err := r.handleDryRun(ctx, exp, pods, "delete")
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), result.RequeueAfter)

	refreshed := &chaosv1alpha1.ChaosExperiment{}
	require.NoError(t, r.Get(ctx, clientKey(exp), refreshed))
	assert.Equal(t, "Completed", refreshed.Status.Phase)
	assert.NotNil(t, refreshed.Status.LastRunTime)
	assert.Contains(t, refreshed.Status.Message, "DRY RUN")
	assert.Contains(t, refreshed.Status.Message, "delete 2 pod(s)")
}
