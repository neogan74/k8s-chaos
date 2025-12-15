package controller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	chaosv1alpha1 "github.com/neogan74/k8s-chaos/api/v1alpha1"
)

func TestCheckSchedule_NoScheduleRunsImmediately(t *testing.T) {
	ctx := context.Background()
	exp := &chaosv1alpha1.ChaosExperiment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "immediate",
			Namespace: "default",
		},
		Spec: chaosv1alpha1.ChaosExperimentSpec{
			Action: "pod-kill",
		},
	}

	r := newReconcilerWithObjects(t, exp)

	shouldRun, requeueAfter, err := r.checkSchedule(ctx, exp)
	require.NoError(t, err)
	assert.True(t, shouldRun)
	assert.Equal(t, time.Minute, requeueAfter)
}

func TestCheckSchedule_InvalidScheduleErrors(t *testing.T) {
	ctx := context.Background()
	exp := &chaosv1alpha1.ChaosExperiment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bad-schedule",
			Namespace: "default",
		},
		Spec: chaosv1alpha1.ChaosExperimentSpec{
			Action:   "pod-kill",
			Schedule: "not-a-cron",
		},
	}

	r := newReconcilerWithObjects(t, exp)

	shouldRun, _, err := r.checkSchedule(ctx, exp)
	assert.Error(t, err)
	assert.False(t, shouldRun)
}

func TestCheckSchedule_CronUpdatesStatusAndRuns(t *testing.T) {
	ctx := context.Background()
	createdAt := metav1.NewTime(time.Now().Add(-6 * time.Minute))
	exp := &chaosv1alpha1.ChaosExperiment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "cron-exp",
			Namespace:         "default",
			CreationTimestamp: createdAt,
		},
		Spec: chaosv1alpha1.ChaosExperimentSpec{
			Action:   "pod-kill",
			Schedule: "*/5 * * * *", // every 5 minutes
		},
	}

	r := newReconcilerWithObjects(t, exp)

	shouldRun, requeueAfter, err := r.checkSchedule(ctx, exp)
	require.NoError(t, err)
	assert.True(t, shouldRun)
	assert.Equal(t, time.Duration(0), requeueAfter)

	// Status should have been updated with last/next scheduled times
	refreshed := &chaosv1alpha1.ChaosExperiment{}
	require.NoError(t, r.Get(ctx, clientKey(exp), refreshed))
	assert.NotNil(t, refreshed.Status.LastScheduledTime)
	assert.NotNil(t, refreshed.Status.NextScheduledTime)
}

func TestCheckExperimentLifecycle_StartsAndCompletes(t *testing.T) {
	ctx := context.Background()
	exp := &chaosv1alpha1.ChaosExperiment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "duration-exp",
			Namespace: "default",
		},
		Spec: chaosv1alpha1.ChaosExperimentSpec{
			Action:             "pod-kill",
			ExperimentDuration: "1s",
		},
	}

	r := newReconcilerWithObjects(t, exp)

	// First call initializes StartTime
	shouldContinue, err := r.checkExperimentLifecycle(ctx, exp)
	require.NoError(t, err)
	assert.True(t, shouldContinue)
	assert.Equal(t, phaseRunning, exp.Status.Phase)
	assert.NotNil(t, exp.Status.StartTime)

	// Simulate elapsed duration by moving StartTime into the past
	past := metav1.NewTime(time.Now().Add(-2 * time.Second))
	exp.Status.StartTime = &past
	exp.Status.Phase = phaseRunning
	require.NoError(t, r.Status().Update(ctx, exp))

	shouldContinue, err = r.checkExperimentLifecycle(ctx, exp)
	require.NoError(t, err)
	assert.False(t, shouldContinue)

	refreshed := &chaosv1alpha1.ChaosExperiment{}
	require.NoError(t, r.Get(ctx, clientKey(exp), refreshed))
	assert.Equal(t, phaseCompleted, refreshed.Status.Phase)
	assert.NotNil(t, refreshed.Status.CompletedAt)
	assert.Contains(t, refreshed.Status.Message, "Experiment completed")
}

// clientKey builds a NamespacedName-style key for fake client lookups.
func clientKey(exp *chaosv1alpha1.ChaosExperiment) client.ObjectKey {
	return client.ObjectKey{Name: exp.Name, Namespace: exp.Namespace}
}
