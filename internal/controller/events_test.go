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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	chaosv1alpha1 "github.com/neogan74/k8s-chaos/api/v1alpha1"
)

func TestPodKillEmitsEvent(t *testing.T) {
	ctx := context.Background()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "target-pod",
			Namespace: "default",
			Labels: map[string]string{
				"app": "demo",
			},
		},
	}
	exp := &chaosv1alpha1.ChaosExperiment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-exp",
			Namespace: "default",
		},
		Spec: chaosv1alpha1.ChaosExperimentSpec{
			Action:    "pod-kill",
			Namespace: "default",
			Selector: map[string]string{
				"app": "demo",
			},
			Count: 1,
		},
	}

	r := newReconcilerWithObjects(t, pod)

	// Inject the experiment into the client
	require.NoError(t, r.Create(ctx, exp))

	// Run handler
	_, err := r.handlePodKill(ctx, exp)
	require.NoError(t, err)

	// Verify event
	fakeRecorder, ok := r.Recorder.(*record.FakeRecorder)
	require.True(t, ok)

	select {
	case event := <-fakeRecorder.Events:
		// Expected event: warning ChaosPodKill Pod killed by chaos experiment test-exp
		assert.Contains(t, event, "Pod killed by chaos experiment test-exp")
		assert.Contains(t, event, "Warning ChaosPodKill")
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for event")
	}
}
