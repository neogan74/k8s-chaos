package controller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	chaosv1alpha1 "github.com/neogan74/k8s-chaos/api/v1alpha1"
)

func TestCleanupExpiredHistory(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = chaosv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	now := time.Now()
	historyNamespace := "chaos-system"

	// 1. Define history records
	expiredRecord := &chaosv1alpha1.ChaosExperimentHistory{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "expired-record",
			Namespace:         historyNamespace,
			CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Hour)),
		},
	}
	recentRecord := &chaosv1alpha1.ChaosExperimentHistory{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "recent-record",
			Namespace:         historyNamespace,
			CreationTimestamp: metav1.NewTime(now.Add(-10 * time.Minute)),
		},
	}

	// 2. Setup fake client and reconciler
	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(expiredRecord, recentRecord).
		Build()

	reconciler := &ChaosExperimentReconciler{
		Client: k8sClient,
		HistoryConfig: HistoryConfig{
			Enabled:      true,
			Namespace:    historyNamespace,
			RetentionTTL: 1 * time.Hour,
		},
	}

	// 3. Run cleanup
	reconciler.cleanupExpiredHistory(context.Background())

	// 4. Verify results
	var historyList chaosv1alpha1.ChaosExperimentHistoryList
	err := k8sClient.List(context.Background(), &historyList)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(historyList.Items), "One record should remain")
	assert.Equal(t, "recent-record", historyList.Items[0].Name, "Recent record should be preserved")
}

func TestCleanupExpiredHistory_Disabled(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = chaosv1alpha1.AddToScheme(scheme)

	now := time.Now()
	historyNamespace := "chaos-system"

	expiredRecord := &chaosv1alpha1.ChaosExperimentHistory{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "expired-record",
			Namespace:         historyNamespace,
			CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Hour)),
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(expiredRecord).
		Build()

	reconciler := &ChaosExperimentReconciler{
		Client: k8sClient,
		HistoryConfig: HistoryConfig{
			RetentionTTL: 0, // Disabled
		},
	}

	reconciler.cleanupExpiredHistory(context.Background())

	var historyList chaosv1alpha1.ChaosExperimentHistoryList
	_ = k8sClient.List(context.Background(), &historyList)
	assert.Equal(t, 1, len(historyList.Items), "Record should NOT be deleted when TTL is 0")
}
