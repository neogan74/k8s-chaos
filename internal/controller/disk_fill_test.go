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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	chaosv1alpha1 "github.com/neogan74/k8s-chaos/api/v1alpha1"
)

// ---------------------------------------------------------------------------
// resolveDiskFillTarget
// ---------------------------------------------------------------------------

func TestResolveDiskFillTarget(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "default"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "app",
					VolumeMounts: []corev1.VolumeMount{
						{Name: "data-vol", MountPath: "/data"},
						{Name: "log-vol", MountPath: "/var/log/app"},
					},
				},
			},
			InitContainers: []corev1.Container{
				{
					Name: "init",
					VolumeMounts: []corev1.VolumeMount{
						{Name: "init-vol", MountPath: "/init-data"},
					},
				},
			},
		},
	}

	tests := []struct {
		name       string
		volumeName string
		targetPath string
		wantPath   string
		wantErr    bool
	}{
		{
			name:       "returns targetPath when volumeName is empty",
			volumeName: "",
			targetPath: "/tmp",
			wantPath:   "/tmp",
		},
		{
			name:       "resolves volume from regular container",
			volumeName: "data-vol",
			targetPath: "",
			wantPath:   "/data",
		},
		{
			name:       "resolves second volume from regular container",
			volumeName: "log-vol",
			targetPath: "",
			wantPath:   "/var/log/app",
		},
		{
			name:       "resolves volume from init container",
			volumeName: "init-vol",
			targetPath: "",
			wantPath:   "/init-data",
		},
		{
			name:       "returns error when volume not found",
			volumeName: "missing-vol",
			targetPath: "",
			wantErr:    true,
		},
		{
			name:       "returns error when both volumeName and targetPath are empty",
			volumeName: "",
			targetPath: "",
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveDiskFillTarget(pod, tc.volumeName, tc.targetPath)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantPath, got)
		})
	}
}

// ---------------------------------------------------------------------------
// handlePodDiskFill — validation-only tests (no envtest needed)
// ---------------------------------------------------------------------------

func TestHandlePodDiskFill_Validation(t *testing.T) {
	tests := []struct {
		name        string
		spec        func(*chaosv1alpha1.ChaosExperimentSpec)
		wantMessage string
	}{
		{
			name: "fails when duration is empty",
			spec: func(s *chaosv1alpha1.ChaosExperimentSpec) {
				s.Duration = ""
				s.FillPercentage = 80
				s.TargetPath = "/tmp"
			},
			wantMessage: "duration is required",
		},
		{
			name: "fails when duration is invalid",
			spec: func(s *chaosv1alpha1.ChaosExperimentSpec) {
				s.Duration = "not-a-duration"
				s.FillPercentage = 80
				s.TargetPath = "/tmp"
			},
			wantMessage: "invalid duration format",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			exp := makeDiskFillExperiment("disk-fill-validation", "default")
			tc.spec(&exp.Spec)

			r := newReconcilerWithObjects(t, exp)
			_, err := r.handlePodDiskFill(ctx, exp)
			require.NoError(t, err)

			updated := fetchExperiment(t, r, exp.Name, exp.Namespace)
			assert.Contains(t, updated.Status.Message, tc.wantMessage,
				"message should contain %q, got %q", tc.wantMessage, updated.Status.Message)
		})
	}
}

func TestHandlePodDiskFill_NoEligiblePods(t *testing.T) {
	ctx := context.Background()

	exp := makeDiskFillExperiment("disk-fill-no-pods", "default")

	r := newReconcilerWithObjects(t, exp)
	result, err := r.handlePodDiskFill(ctx, exp)
	require.NoError(t, err)
	assert.Greater(t, result.RequeueAfter.Seconds(), 0.0)

	updated := fetchExperiment(t, r, exp.Name, exp.Namespace)
	assert.Contains(t, updated.Status.Message, "No eligible pods")
}

func TestHandlePodDiskFill_DryRun(t *testing.T) {
	ctx := context.Background()

	exp := makeDiskFillExperiment("disk-fill-dryrun", "default")
	exp.Spec.DryRun = true

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "target-pod",
			Namespace: "default",
			Labels:    map[string]string{"app": "demo"},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}

	r := newReconcilerWithObjects(t, exp, pod)
	_, err := r.handlePodDiskFill(ctx, exp)
	require.NoError(t, err)

	updated := fetchExperiment(t, r, exp.Name, exp.Namespace)
	assert.Equal(t, "Completed", updated.Status.Phase)
	assert.Contains(t, updated.Status.Message, "DRY RUN")
}

func TestHandlePodDiskFill_DefaultFillPercentage(t *testing.T) {
	ctx := context.Background()

	exp := makeDiskFillExperiment("disk-fill-default-pct", "default")
	exp.Spec.FillPercentage = 0 // should default to 80

	// No pods — just verifies the function handles the default properly
	r := newReconcilerWithObjects(t, exp)
	result, err := r.handlePodDiskFill(ctx, exp)
	require.NoError(t, err)
	// With no pods, the message reflects "No eligible pods"; important is no error/panic.
	assert.Greater(t, result.RequeueAfter.Seconds(), 0.0)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func makeDiskFillExperiment(name, namespace string) *chaosv1alpha1.ChaosExperiment {
	return &chaosv1alpha1.ChaosExperiment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: chaosv1alpha1.ChaosExperimentSpec{
			Action:         "pod-disk-fill",
			Namespace:      namespace,
			Selector:       map[string]string{"app": "demo"},
			Count:          1,
			Duration:       "2m",
			FillPercentage: 80,
			TargetPath:     "/tmp",
		},
	}
}
