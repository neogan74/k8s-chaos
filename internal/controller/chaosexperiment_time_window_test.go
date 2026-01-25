package controller

import (
	"context"
	"testing"
	"time"

	chaosv1alpha1 "github.com/neogan74/k8s-chaos/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCheckTimeWindows_MaintenanceWindows(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = chaosv1alpha1.AddToScheme(scheme)

	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour).Format(time.RFC3339)
	oneHourLater := now.Add(1 * time.Hour).Format(time.RFC3339)
	twoHoursLater := now.Add(2 * time.Hour).Format(time.RFC3339)

	tests := []struct {
		name           string
		experiment     *chaosv1alpha1.ChaosExperiment
		expectedResult bool
		expectRequeue  bool
	}{
		{
			name: "No Windows - Allowed",
			experiment: &chaosv1alpha1.ChaosExperiment{
				ObjectMeta: metav1.ObjectMeta{Name: "test-exp", Namespace: "default"},
				Spec:       chaosv1alpha1.ChaosExperimentSpec{},
			},
			expectedResult: true,
			expectRequeue:  false,
		},
		{
			name: "Maintenance Window Active - Blocked",
			experiment: &chaosv1alpha1.ChaosExperiment{
				ObjectMeta: metav1.ObjectMeta{Name: "test-exp", Namespace: "default"},
				Spec: chaosv1alpha1.ChaosExperimentSpec{
					MaintenanceWindows: []chaosv1alpha1.TimeWindow{
						{
							Type:  chaosv1alpha1.TimeWindowAbsolute,
							Start: oneHourAgo,
							End:   oneHourLater,
						},
					},
				},
			},
			expectedResult: false,
			expectRequeue:  true,
		},
		{
			name: "Maintenance Window Future - Allowed",
			experiment: &chaosv1alpha1.ChaosExperiment{
				ObjectMeta: metav1.ObjectMeta{Name: "test-exp", Namespace: "default"},
				Spec: chaosv1alpha1.ChaosExperimentSpec{
					MaintenanceWindows: []chaosv1alpha1.TimeWindow{
						{
							Type:  chaosv1alpha1.TimeWindowAbsolute,
							Start: oneHourLater,
							End:   twoHoursLater,
						},
					},
				},
			},
			expectedResult: true,
			expectRequeue:  false,
		},
		{
			name: "Allowed Window Active - Allowed",
			experiment: &chaosv1alpha1.ChaosExperiment{
				ObjectMeta: metav1.ObjectMeta{Name: "test-exp", Namespace: "default"},
				Spec: chaosv1alpha1.ChaosExperimentSpec{
					TimeWindows: []chaosv1alpha1.TimeWindow{
						{
							Type:  chaosv1alpha1.TimeWindowAbsolute,
							Start: oneHourAgo,
							End:   oneHourLater,
						},
					},
				},
			},
			expectedResult: true,
			expectRequeue:  false,
		},
		{
			name: "Allowed Window Inactive - Blocked",
			experiment: &chaosv1alpha1.ChaosExperiment{
				ObjectMeta: metav1.ObjectMeta{Name: "test-exp", Namespace: "default"},
				Spec: chaosv1alpha1.ChaosExperimentSpec{
					TimeWindows: []chaosv1alpha1.TimeWindow{
						{
							Type:  chaosv1alpha1.TimeWindowAbsolute,
							Start: oneHourLater,
							End:   twoHoursLater,
						},
					},
				},
			},
			expectedResult: false,
			expectRequeue:  true,
		},
		{
			name: "Both Windows Active - Maintenance Wins (Blocked)",
			experiment: &chaosv1alpha1.ChaosExperiment{
				ObjectMeta: metav1.ObjectMeta{Name: "test-exp", Namespace: "default"},
				Spec: chaosv1alpha1.ChaosExperimentSpec{
					TimeWindows: []chaosv1alpha1.TimeWindow{
						{
							Type:  chaosv1alpha1.TimeWindowAbsolute,
							Start: oneHourAgo,
							End:   oneHourLater,
						},
					},
					MaintenanceWindows: []chaosv1alpha1.TimeWindow{
						{
							Type:  chaosv1alpha1.TimeWindowAbsolute,
							Start: oneHourAgo,
							End:   oneHourLater,
						},
					},
				},
			},
			expectedResult: false,
			expectRequeue:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ChaosExperimentReconciler{
				Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
				Scheme: scheme,
			}

			allowed, requeueTime, err := r.checkTimeWindows(context.TODO(), tt.experiment)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedResult, allowed)

			if tt.expectRequeue {
				assert.False(t, requeueTime.IsZero())
			}
		})
	}
}
