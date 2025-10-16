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
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestChaosExperimentWebhook_ValidateCreate(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = AddToScheme(scheme)

	tests := []struct {
		name        string
		experiment  *ChaosExperiment
		objects     []client.Object
		wantErr     bool
		errContains string
		wantWarning bool
	}{
		{
			name: "valid pod-kill experiment",
			experiment: &ChaosExperiment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-experiment",
					Namespace: "default",
				},
				Spec: ChaosExperimentSpec{
					Action:    "pod-kill",
					Namespace: "test-ns",
					Selector:  map[string]string{"app": "test"},
					Count:     1,
				},
			},
			objects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns",
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod-1",
						Namespace: "test-ns",
						Labels:    map[string]string{"app": "test"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid pod-delay with duration",
			experiment: &ChaosExperiment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-experiment",
					Namespace: "default",
				},
				Spec: ChaosExperimentSpec{
					Action:    "pod-delay",
					Namespace: "test-ns",
					Selector:  map[string]string{"app": "test"},
					Count:     1,
					Duration:  "30s",
				},
			},
			objects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns",
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod-1",
						Namespace: "test-ns",
						Labels:    map[string]string{"app": "test"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "namespace does not exist",
			experiment: &ChaosExperiment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-experiment",
					Namespace: "default",
				},
				Spec: ChaosExperimentSpec{
					Action:    "pod-kill",
					Namespace: "nonexistent",
					Selector:  map[string]string{"app": "test"},
					Count:     1,
				},
			},
			objects:     []client.Object{},
			wantErr:     true,
			errContains: "does not exist",
		},
		{
			name: "selector matches no pods",
			experiment: &ChaosExperiment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-experiment",
					Namespace: "default",
				},
				Spec: ChaosExperimentSpec{
					Action:    "pod-kill",
					Namespace: "test-ns",
					Selector:  map[string]string{"app": "nonexistent"},
					Count:     1,
				},
			},
			objects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns",
					},
				},
			},
			wantErr:     true,
			errContains: "does not match any pods",
		},
		{
			name: "pod-delay without duration",
			experiment: &ChaosExperiment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-experiment",
					Namespace: "default",
				},
				Spec: ChaosExperimentSpec{
					Action:    "pod-delay",
					Namespace: "test-ns",
					Selector:  map[string]string{"app": "test"},
					Count:     1,
				},
			},
			objects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns",
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod-1",
						Namespace: "test-ns",
						Labels:    map[string]string{"app": "test"},
					},
				},
			},
			wantErr:     true,
			errContains: "duration is required for pod-delay",
		},
		{
			name: "count exceeds available pods (warning)",
			experiment: &ChaosExperiment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-experiment",
					Namespace: "default",
				},
				Spec: ChaosExperimentSpec{
					Action:    "pod-kill",
					Namespace: "test-ns",
					Selector:  map[string]string{"app": "test"},
					Count:     5,
				},
			},
			objects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns",
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod-1",
						Namespace: "test-ns",
						Labels:    map[string]string{"app": "test"},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod-2",
						Namespace: "test-ns",
						Labels:    map[string]string{"app": "test"},
					},
				},
			},
			wantErr:     false,
			wantWarning: true,
		},
		{
			name: "invalid duration format",
			experiment: &ChaosExperiment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-experiment",
					Namespace: "default",
				},
				Spec: ChaosExperimentSpec{
					Action:    "pod-delay",
					Namespace: "test-ns",
					Selector:  map[string]string{"app": "test"},
					Count:     1,
					Duration:  "invalid",
				},
			},
			objects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns",
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod-1",
						Namespace: "test-ns",
						Labels:    map[string]string{"app": "test"},
					},
				},
			},
			wantErr:     true,
			errContains: "duration must match pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client with initial objects
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				Build()

			webhook := &ChaosExperimentWebhook{
				Client: fakeClient,
			}

			warnings, err := webhook.ValidateCreate(context.Background(), tt.experiment)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCreate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateCreate() error = %v, should contain %q", err, tt.errContains)
				}
			}

			if tt.wantWarning && len(warnings) == 0 {
				t.Errorf("ValidateCreate() expected warnings but got none")
			}

			if !tt.wantWarning && len(warnings) > 0 {
				t.Errorf("ValidateCreate() unexpected warnings: %v", warnings)
			}
		})
	}
}

func TestChaosExperimentWebhook_ValidateUpdate(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = AddToScheme(scheme)

	oldExperiment := &ChaosExperiment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-experiment",
			Namespace: "default",
		},
		Spec: ChaosExperimentSpec{
			Action:    "pod-kill",
			Namespace: "test-ns",
			Selector:  map[string]string{"app": "old"},
			Count:     1,
		},
	}

	newExperiment := &ChaosExperiment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-experiment",
			Namespace: "default",
		},
		Spec: ChaosExperimentSpec{
			Action:    "pod-kill",
			Namespace: "test-ns",
			Selector:  map[string]string{"app": "new"},
			Count:     2,
		},
	}

	objects := []client.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-ns",
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-1",
				Namespace: "test-ns",
				Labels:    map[string]string{"app": "new"},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		Build()

	webhook := &ChaosExperimentWebhook{
		Client: fakeClient,
	}

	_, err := webhook.ValidateUpdate(context.Background(), oldExperiment, newExperiment)
	if err != nil {
		t.Errorf("ValidateUpdate() error = %v, expected nil", err)
	}
}

func TestChaosExperimentWebhook_ValidateDelete(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = AddToScheme(scheme)

	experiment := &ChaosExperiment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-experiment",
			Namespace: "default",
		},
		Spec: ChaosExperimentSpec{
			Action:    "pod-kill",
			Namespace: "test-ns",
			Selector:  map[string]string{"app": "test"},
			Count:     1,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	webhook := &ChaosExperimentWebhook{
		Client: fakeClient,
	}

	_, err := webhook.ValidateDelete(context.Background(), experiment)
	if err != nil {
		t.Errorf("ValidateDelete() error = %v, expected nil", err)
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
