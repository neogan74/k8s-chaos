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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	chaosv1alpha1 "github.com/neogan74/k8s-chaos/api/v1alpha1"
)

// ---------------------------------------------------------------------------
// taintNode
// ---------------------------------------------------------------------------

func TestTaintNode(t *testing.T) {
	tests := []struct {
		name              string
		existingTaints    []corev1.Taint
		taintKey          string
		taintValue        string
		taintEffect       string
		wantAlreadyTainted bool
		wantTaintCount    int
		wantErr           bool
	}{
		{
			name:              "adds new taint to untainted node",
			existingTaints:    nil,
			taintKey:          "chaos/test",
			taintValue:        "true",
			taintEffect:       "NoSchedule",
			wantAlreadyTainted: false,
			wantTaintCount:    1,
		},
		{
			name: "detects already tainted node (same key + effect)",
			existingTaints: []corev1.Taint{
				{Key: "chaos/test", Value: "old", Effect: corev1.TaintEffectNoSchedule},
			},
			taintKey:           "chaos/test",
			taintValue:         "new",
			taintEffect:        "NoSchedule",
			wantAlreadyTainted: true,
			wantTaintCount:     1, // unchanged
		},
		{
			name: "adds taint to node with a different existing taint",
			existingTaints: []corev1.Taint{
				{Key: "other/key", Value: "val", Effect: corev1.TaintEffectNoExecute},
			},
			taintKey:           "chaos/test",
			taintValue:         "true",
			taintEffect:        "NoSchedule",
			wantAlreadyTainted: false,
			wantTaintCount:     2, // original + new
		},
		{
			name: "same key but different effect is treated as new taint",
			existingTaints: []corev1.Taint{
				{Key: "chaos/test", Value: "true", Effect: corev1.TaintEffectNoExecute},
			},
			taintKey:           "chaos/test",
			taintValue:         "true",
			taintEffect:        "NoSchedule",
			wantAlreadyTainted: false,
			wantTaintCount:     2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			node := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
				},
				Spec: corev1.NodeSpec{
					Taints: tc.existingTaints,
				},
			}

			r := newReconcilerWithObjects(t, node)

			alreadyTainted, err := r.taintNode(ctx, node, tc.taintKey, tc.taintValue, tc.taintEffect)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantAlreadyTainted, alreadyTainted)

			// Verify the node in the fake store
			updated := &corev1.Node{}
			require.NoError(t, r.Get(ctx, types.NamespacedName{Name: "test-node"}, updated))
			assert.Len(t, updated.Spec.Taints, tc.wantTaintCount)

			if !tc.wantAlreadyTainted {
				// The new taint must be present
				found := false
				for _, t_ := range updated.Spec.Taints {
					if t_.Key == tc.taintKey && string(t_.Effect) == tc.taintEffect {
						found = true
						assert.Equal(t, tc.taintValue, t_.Value)
					}
				}
				assert.True(t, found, "expected new taint %s to be present on the node", tc.taintKey)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// cordonNode
// ---------------------------------------------------------------------------

func TestCordonNode(t *testing.T) {
	tests := []struct {
		name                string
		alreadyUnschedulable bool
		wantCordoned        bool // wasAlreadyCordoned returned
	}{
		{
			name:                "cordons a schedulable node",
			alreadyUnschedulable: false,
			wantCordoned:        false,
		},
		{
			name:                "detects already cordoned node",
			alreadyUnschedulable: true,
			wantCordoned:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			node := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
				},
				Spec: corev1.NodeSpec{
					Unschedulable: tc.alreadyUnschedulable,
				},
			}

			r := newReconcilerWithObjects(t, node)

			wasCordoned, err := r.cordonNode(ctx, node)
			require.NoError(t, err)
			assert.Equal(t, tc.wantCordoned, wasCordoned)

			// Verify state in fake store
			updated := &corev1.Node{}
			require.NoError(t, r.Get(ctx, types.NamespacedName{Name: "test-node"}, updated))
			assert.True(t, updated.Spec.Unschedulable, "node should be unschedulable after cordon")
		})
	}
}

// ---------------------------------------------------------------------------
// untaintNode
// ---------------------------------------------------------------------------

func TestUntaintNode(t *testing.T) {
	tests := []struct {
		name           string
		existingTaints []corev1.Taint
		removeKey      string
		removeEffect   string
		wantTaintCount int
		wantErr        bool
		emptyNodeName  bool
	}{
		{
			name: "removes the matching taint",
			existingTaints: []corev1.Taint{
				{Key: "chaos/test", Value: "true", Effect: corev1.TaintEffectNoSchedule},
			},
			removeKey:      "chaos/test",
			removeEffect:   "NoSchedule",
			wantTaintCount: 0,
		},
		{
			name: "removes only the matching taint, leaves others",
			existingTaints: []corev1.Taint{
				{Key: "chaos/test", Value: "true", Effect: corev1.TaintEffectNoSchedule},
				{Key: "other/key", Value: "val", Effect: corev1.TaintEffectNoExecute},
			},
			removeKey:      "chaos/test",
			removeEffect:   "NoSchedule",
			wantTaintCount: 1,
		},
		{
			name:           "no-op when taint is not present",
			existingTaints: nil,
			removeKey:      "chaos/test",
			removeEffect:   "NoSchedule",
			wantTaintCount: 0,
		},
		{
			name:          "returns error for empty node name",
			emptyNodeName: true,
			wantErr:       true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			node := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
				},
				Spec: corev1.NodeSpec{
					Taints: tc.existingTaints,
				},
			}

			var r *ChaosExperimentReconciler
			var nodeName string

			if tc.emptyNodeName {
				r = newReconcilerWithObjects(t)
				nodeName = ""
			} else {
				r = newReconcilerWithObjects(t, node)
				nodeName = "test-node"
			}

			err := r.untaintNode(ctx, nodeName, tc.removeKey, tc.removeEffect)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Verify remaining taints
			updated := &corev1.Node{}
			require.NoError(t, r.Get(ctx, types.NamespacedName{Name: "test-node"}, updated))
			assert.Len(t, updated.Spec.Taints, tc.wantTaintCount)
		})
	}
}

// ---------------------------------------------------------------------------
// uncordonNode
// ---------------------------------------------------------------------------

func TestUncordonNode(t *testing.T) {
	tests := []struct {
		name              string
		startUnschedulable bool
		wantUnschedulable  bool
		nodeMissing       bool
		wantErr           bool
	}{
		{
			name:               "uncordons a cordoned node",
			startUnschedulable: true,
			wantUnschedulable:  false,
		},
		{
			name:               "no-op on already schedulable node",
			startUnschedulable: false,
			wantUnschedulable:  false,
		},
		{
			name:        "returns error when node is not found",
			nodeMissing: true,
			wantErr:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			var r *ChaosExperimentReconciler

			if tc.nodeMissing {
				r = newReconcilerWithObjects(t)
			} else {
				node := &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-node",
					},
					Spec: corev1.NodeSpec{
						Unschedulable: tc.startUnschedulable,
					},
				}
				r = newReconcilerWithObjects(t, node)
			}

			err := r.uncordonNode(ctx, "test-node")
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			updated := &corev1.Node{}
			require.NoError(t, r.Get(ctx, types.NamespacedName{Name: "test-node"}, updated))
			assert.Equal(t, tc.wantUnschedulable, updated.Spec.Unschedulable)
		})
	}
}

// ---------------------------------------------------------------------------
// handleNodeCPUStress — validation-only tests (no envtest needed)
// ---------------------------------------------------------------------------

func TestHandleNodeCPUStress_Validation(t *testing.T) {
	tests := []struct {
		name        string
		spec        func(*chaosv1alpha1.ChaosExperimentSpec)
		wantPhase   string
		wantMessage string
	}{
		{
			name: "fails when CPULoad is zero",
			spec: func(s *chaosv1alpha1.ChaosExperimentSpec) {
				s.CPULoad = 0
				s.Duration = "30s"
			},
			wantPhase:   "Failed",
			wantMessage: "CPULoad must be specified",
		},
		{
			name: "fails when CPULoad is negative",
			spec: func(s *chaosv1alpha1.ChaosExperimentSpec) {
				s.CPULoad = -5
				s.Duration = "30s"
			},
			wantPhase:   "Failed",
			wantMessage: "CPULoad must be specified",
		},
		{
			name: "fails when Duration is empty",
			spec: func(s *chaosv1alpha1.ChaosExperimentSpec) {
				s.CPULoad = 50
				s.Duration = ""
			},
			wantPhase:   "Failed",
			wantMessage: "Duration is required",
		},
		{
			name: "fails when Duration is invalid",
			spec: func(s *chaosv1alpha1.ChaosExperimentSpec) {
				s.CPULoad = 50
				s.Duration = "not-a-duration"
			},
			wantPhase:   "Failed",
			wantMessage: "Invalid duration format",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			exp := makeCPUStressExperiment("cpu-stress-validation", "default")
			tc.spec(&exp.Spec)

			r := newReconcilerWithObjects(t, exp)
			_, err := r.handleNodeCPUStress(ctx, exp)
			require.NoError(t, err)

			updated := fetchExperiment(t, r, exp.Name, exp.Namespace)
			assert.Contains(t, updated.Status.Message, tc.wantMessage,
				"message should contain %q, got %q", tc.wantMessage, updated.Status.Message)
		})
	}
}

// ---------------------------------------------------------------------------
// handleNodeTaint — validation-only tests (no envtest needed)
// ---------------------------------------------------------------------------

func TestHandleNodeTaint_Validation(t *testing.T) {
	tests := []struct {
		name        string
		spec        func(*chaosv1alpha1.ChaosExperimentSpec)
		wantMessage string
	}{
		{
			name: "fails when TaintKey is empty",
			spec: func(s *chaosv1alpha1.ChaosExperimentSpec) {
				s.TaintKey = ""
				s.TaintEffect = "NoSchedule"
			},
			wantMessage: "TaintKey and TaintEffect must be specified",
		},
		{
			name: "fails when TaintEffect is empty",
			spec: func(s *chaosv1alpha1.ChaosExperimentSpec) {
				s.TaintKey = "chaos/test"
				s.TaintEffect = ""
			},
			wantMessage: "TaintKey and TaintEffect must be specified",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			exp := makeTaintExperiment("taint-validation", "default")
			tc.spec(&exp.Spec)

			r := newReconcilerWithObjects(t, exp)
			_, err := r.handleNodeTaint(ctx, exp)
			require.NoError(t, err)

			updated := fetchExperiment(t, r, exp.Name, exp.Namespace)
			assert.Contains(t, updated.Status.Message, tc.wantMessage)
		})
	}
}

// ---------------------------------------------------------------------------
// helpers for node handler tests
// ---------------------------------------------------------------------------

// makeCPUStressExperiment builds a minimal ChaosExperiment for node-cpu-stress tests.
func makeCPUStressExperiment(name, namespace string) *chaosv1alpha1.ChaosExperiment {
	return &chaosv1alpha1.ChaosExperiment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: chaosv1alpha1.ChaosExperimentSpec{
			Action:   "node-cpu-stress",
			CPULoad:  50,
			Duration: "30s",
			Selector: map[string]string{"role": "worker"},
		},
	}
}

// makeTaintExperiment builds a minimal ChaosExperiment for node-taint tests.
func makeTaintExperiment(name, namespace string) *chaosv1alpha1.ChaosExperiment {
	return &chaosv1alpha1.ChaosExperiment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: chaosv1alpha1.ChaosExperimentSpec{
			Action:      "node-taint",
			TaintKey:    "chaos/test",
			TaintEffect: "NoSchedule",
			Selector:    map[string]string{"role": "worker"},
		},
	}
}

// fetchExperiment retrieves a ChaosExperiment from the fake store by name/namespace.
func fetchExperiment(t *testing.T, r *ChaosExperimentReconciler, name, namespace string) *chaosv1alpha1.ChaosExperiment {
	t.Helper()
	exp := &chaosv1alpha1.ChaosExperiment{}
	require.NoError(t, r.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, exp))
	return exp
}
