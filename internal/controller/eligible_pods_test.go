package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	chaosv1alpha1 "github.com/neogan74/k8s-chaos/api/v1alpha1"
)

// Build a reconciler with a fake client populated with the given objects.
func newReconcilerWithObjects(t *testing.T, objs ...client.Object) *ChaosExperimentReconciler {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, chaosv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithStatusSubresource(&chaosv1alpha1.ChaosExperiment{}).
		Build()

	return &ChaosExperimentReconciler{
		Client:        cl,
		Scheme:        scheme,
		HistoryConfig: DefaultHistoryConfig(),
	}
}

func TestGetEligiblePods_NamespaceExcluded(t *testing.T) {
	ctx := context.Background()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "blocked",
			Annotations: map[string]string{
				chaosv1alpha1.ExclusionLabel: "true",
			},
		},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-pod",
			Namespace: "blocked",
			Labels: map[string]string{
				"app": "demo",
			},
		},
	}
	exp := &chaosv1alpha1.ChaosExperiment{
		Spec: chaosv1alpha1.ChaosExperimentSpec{
			Action:    "pod-kill",
			Namespace: "blocked",
			Selector: map[string]string{
				"app": "demo",
			},
		},
	}

	r := newReconcilerWithObjects(t, ns, pod)

	eligible, err := r.getEligiblePods(ctx, exp)
	require.NoError(t, err)
	assert.Len(t, eligible, 0, "namespace exclusion should filter all pods")
}

func TestGetEligiblePods_PodLabelExcluded(t *testing.T) {
	ctx := context.Background()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ns",
		},
	}
	excludedPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "skip-me",
			Namespace: "test-ns",
			Labels: map[string]string{
				"app":                        "demo",
				chaosv1alpha1.ExclusionLabel: "true",
			},
		},
	}
	includedPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keep-me",
			Namespace: "test-ns",
			Labels: map[string]string{
				"app": "demo",
			},
		},
	}
	exp := &chaosv1alpha1.ChaosExperiment{
		Spec: chaosv1alpha1.ChaosExperimentSpec{
			Action:    "pod-kill",
			Namespace: "test-ns",
			Selector: map[string]string{
				"app": "demo",
			},
		},
	}

	r := newReconcilerWithObjects(t, ns, excludedPod, includedPod)

	eligible, err := r.getEligiblePods(ctx, exp)
	require.NoError(t, err)
	assert.Len(t, eligible, 1)
	assert.Equal(t, "keep-me", eligible[0].Name)
}
