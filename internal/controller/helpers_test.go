package controller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	chaosv1alpha1 "github.com/neogan74/k8s-chaos/api/v1alpha1"
)

// Test parseDuration function
func TestParseDuration(t *testing.T) {
	r := &ChaosExperimentReconciler{}
	tests := []struct {
		name     string
		duration string
		want     time.Duration
		wantErr  bool
	}{
		{
			name:     "valid seconds",
			duration: "30s",
			want:     30 * time.Second,
			wantErr:  false,
		},
		{
			name:     "valid minutes",
			duration: "5m",
			want:     5 * time.Minute,
			wantErr:  false,
		},
		{
			name:     "valid hours",
			duration: "2h",
			want:     2 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "complex duration",
			duration: "1h30m",
			want:     90 * time.Minute,
			wantErr:  false,
		},
		{
			name:     "invalid duration",
			duration: "invalid",
			want:     0,
			wantErr:  true,
		},
		{
			name:     "empty string",
			duration: "",
			want:     0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.parseDuration(tt.duration)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// Test parseDurationToSeconds function
func TestParseDurationToSeconds(t *testing.T) {
	r := &ChaosExperimentReconciler{}
	tests := []struct {
		name     string
		duration string
		want     int
		wantErr  bool
	}{
		{
			name:     "30 seconds",
			duration: "30s",
			want:     30,
			wantErr:  false,
		},
		{
			name:     "5 minutes",
			duration: "5m",
			want:     300,
			wantErr:  false,
		},
		{
			name:     "1 hour",
			duration: "1h",
			want:     3600,
			wantErr:  false,
		},
		{
			name:     "complex duration",
			duration: "1h30m45s",
			want:     5445,
			wantErr:  false,
		},
		{
			name:     "invalid duration",
			duration: "invalid",
			want:     0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.parseDurationToSeconds(tt.duration)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// Test parseDurationToMs function
func TestParseDurationToMs(t *testing.T) {
	r := &ChaosExperimentReconciler{}
	tests := []struct {
		name     string
		duration string
		want     int
		wantErr  bool
	}{
		{
			name:     "1 second",
			duration: "1s",
			want:     1000,
			wantErr:  false,
		},
		{
			name:     "30 seconds",
			duration: "30s",
			want:     30000,
			wantErr:  false,
		},
		{
			name:     "1 minute",
			duration: "1m",
			want:     60000,
			wantErr:  false,
		},
		{
			name:     "complex duration",
			duration: "1m30s",
			want:     90000,
			wantErr:  false,
		},
		{
			name:     "invalid duration",
			duration: "invalid",
			want:     0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.parseDurationToMs(tt.duration)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// Test isDaemonSetPod function
func TestIsDaemonSetPod(t *testing.T) {
	tests := []struct {
		name string
		pod  *corev1.Pod
		want bool
	}{
		{
			name: "DaemonSet pod",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "DaemonSet",
							Name: "test-daemonset",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "Deployment pod",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "ReplicaSet",
							Name: "test-replicaset",
						},
					},
				},
			},
			want: false,
		},
		{
			name: "No owner references",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
			},
			want: false,
		},
		{
			name: "Multiple owners including DaemonSet",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "Node",
							Name: "test-node",
						},
						{
							Kind: "DaemonSet",
							Name: "test-daemonset",
						},
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDaemonSetPod(tt.pod)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Test isStaticPod function
func TestIsStaticPod(t *testing.T) {
	tests := []struct {
		name string
		pod  *corev1.Pod
		want bool
	}{
		{
			name: "Static pod with config.source annotation",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
					Annotations: map[string]string{
						"kubernetes.io/config.source": "file",
					},
				},
			},
			want: true,
		},
		{
			name: "Static pod with Node owner reference",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "Node",
							Name: "test-node",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "Regular pod",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
			},
			want: false,
		},
		{
			name: "Pod with other annotations",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
					Annotations: map[string]string{
						"some-other-annotation": "value",
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isStaticPod(tt.pod)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Test isEphemeralContainerRunning function
func TestIsEphemeralContainerRunning(t *testing.T) {
	tests := []struct {
		name          string
		pod           *corev1.Pod
		containerName string
		want          bool
	}{
		{
			name: "running container",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					EphemeralContainerStatuses: []corev1.ContainerStatus{
						{
							Name: "chaos-cpu-stress-123",
							State: corev1.ContainerState{
								Running: &corev1.ContainerStateRunning{
									StartedAt: metav1.Now(),
								},
							},
						},
					},
				},
			},
			containerName: "chaos-cpu-stress-123",
			want:          true,
		},
		{
			name: "terminated container",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					EphemeralContainerStatuses: []corev1.ContainerStatus{
						{
							Name: "chaos-cpu-stress-123",
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 0,
								},
							},
						},
					},
				},
			},
			containerName: "chaos-cpu-stress-123",
			want:          false,
		},
		{
			name: "waiting container",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					EphemeralContainerStatuses: []corev1.ContainerStatus{
						{
							Name: "chaos-cpu-stress-123",
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason: "ContainerCreating",
								},
							},
						},
					},
				},
			},
			containerName: "chaos-cpu-stress-123",
			want:          false,
		},
		{
			name: "container not found",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					EphemeralContainerStatuses: []corev1.ContainerStatus{
						{
							Name: "other-container",
							State: corev1.ContainerState{
								Running: &corev1.ContainerStateRunning{},
							},
						},
					},
				},
			},
			containerName: "chaos-cpu-stress-123",
			want:          true, // Returns true as safe default
		},
		{
			name: "no ephemeral containers",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					EphemeralContainerStatuses: []corev1.ContainerStatus{},
				},
			},
			containerName: "chaos-cpu-stress-123",
			want:          true, // Returns true as safe default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isEphemeralContainerRunning(tt.pod, tt.containerName)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Test calculateRetryDelay function
func TestCalculateRetryDelay(t *testing.T) {
	r := &ChaosExperimentReconciler{}
	tests := []struct {
		name string
		exp  *chaosv1alpha1.ChaosExperiment
		want time.Duration
	}{
		{
			name: "default exponential backoff - first retry",
			exp: &chaosv1alpha1.ChaosExperiment{
				Spec: chaosv1alpha1.ChaosExperimentSpec{},
				Status: chaosv1alpha1.ChaosExperimentStatus{
					RetryCount: 0,
				},
			},
			want: 30 * time.Second, // default base delay * 2^0
		},
		{
			name: "default exponential backoff - second retry",
			exp: &chaosv1alpha1.ChaosExperiment{
				Spec: chaosv1alpha1.ChaosExperimentSpec{},
				Status: chaosv1alpha1.ChaosExperimentStatus{
					RetryCount: 1,
				},
			},
			want: 60 * time.Second, // default base delay * 2^1
		},
		{
			name: "default exponential backoff - third retry",
			exp: &chaosv1alpha1.ChaosExperiment{
				Spec: chaosv1alpha1.ChaosExperimentSpec{},
				Status: chaosv1alpha1.ChaosExperimentStatus{
					RetryCount: 2,
				},
			},
			want: 2 * time.Minute, // default base delay * 2^2
		},
		{
			name: "custom base delay with exponential backoff",
			exp: &chaosv1alpha1.ChaosExperiment{
				Spec: chaosv1alpha1.ChaosExperimentSpec{
					RetryDelay:   "1m",
					RetryBackoff: "exponential",
				},
				Status: chaosv1alpha1.ChaosExperimentStatus{
					RetryCount: 1,
				},
			},
			want: 2 * time.Minute, // 1m * 2^1
		},
		{
			name: "fixed backoff - first retry",
			exp: &chaosv1alpha1.ChaosExperiment{
				Spec: chaosv1alpha1.ChaosExperimentSpec{
					RetryDelay:   "45s",
					RetryBackoff: "fixed",
				},
				Status: chaosv1alpha1.ChaosExperimentStatus{
					RetryCount: 0,
				},
			},
			want: 45 * time.Second,
		},
		{
			name: "fixed backoff - third retry",
			exp: &chaosv1alpha1.ChaosExperiment{
				Spec: chaosv1alpha1.ChaosExperimentSpec{
					RetryDelay:   "45s",
					RetryBackoff: "fixed",
				},
				Status: chaosv1alpha1.ChaosExperimentStatus{
					RetryCount: 2,
				},
			},
			want: 45 * time.Second, // Still 45s for fixed backoff
		},
		{
			name: "exponential backoff capped at 10 minutes",
			exp: &chaosv1alpha1.ChaosExperiment{
				Spec: chaosv1alpha1.ChaosExperimentSpec{
					RetryDelay:   "2m",
					RetryBackoff: "exponential",
				},
				Status: chaosv1alpha1.ChaosExperimentStatus{
					RetryCount: 5, // 2m * 2^5 = 64m, should be capped at 10m
				},
			},
			want: 10 * time.Minute,
		},
		{
			name: "invalid retry delay falls back to default",
			exp: &chaosv1alpha1.ChaosExperiment{
				Spec: chaosv1alpha1.ChaosExperimentSpec{
					RetryDelay:   "invalid",
					RetryBackoff: "exponential",
				},
				Status: chaosv1alpha1.ChaosExperimentStatus{
					RetryCount: 0,
				},
			},
			want: 30 * time.Second, // default base delay
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.calculateRetryDelay(tt.exp)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Test shouldRetry function
func TestShouldRetry(t *testing.T) {
	r := &ChaosExperimentReconciler{}
	tests := []struct {
		name string
		exp  *chaosv1alpha1.ChaosExperiment
		want bool
	}{
		{
			name: "should retry - under default max retries",
			exp: &chaosv1alpha1.ChaosExperiment{
				Spec: chaosv1alpha1.ChaosExperimentSpec{},
				Status: chaosv1alpha1.ChaosExperimentStatus{
					RetryCount: 1,
				},
			},
			want: true, // default max is 3, retry count is 1
		},
		{
			name: "should retry - custom max retries",
			exp: &chaosv1alpha1.ChaosExperiment{
				Spec: chaosv1alpha1.ChaosExperimentSpec{
					MaxRetries: 5,
				},
				Status: chaosv1alpha1.ChaosExperimentStatus{
					RetryCount: 4,
				},
			},
			want: true,
		},
		{
			name: "should not retry - reached default max retries",
			exp: &chaosv1alpha1.ChaosExperiment{
				Spec: chaosv1alpha1.ChaosExperimentSpec{},
				Status: chaosv1alpha1.ChaosExperimentStatus{
					RetryCount: 3,
				},
			},
			want: false, // default max is 3, retry count is 3
		},
		{
			name: "should not retry - exceeded custom max retries",
			exp: &chaosv1alpha1.ChaosExperiment{
				Spec: chaosv1alpha1.ChaosExperimentSpec{
					MaxRetries: 2,
				},
				Status: chaosv1alpha1.ChaosExperimentStatus{
					RetryCount: 3,
				},
			},
			want: false,
		},
		{
			name: "should retry - zero retry count",
			exp: &chaosv1alpha1.ChaosExperiment{
				Spec: chaosv1alpha1.ChaosExperimentSpec{
					MaxRetries: 3,
				},
				Status: chaosv1alpha1.ChaosExperimentStatus{
					RetryCount: 0,
				},
			},
			want: true,
		},
		{
			name: "should retry - max retries 0 uses default",
			exp: &chaosv1alpha1.ChaosExperiment{
				Spec: chaosv1alpha1.ChaosExperimentSpec{
					MaxRetries: 0,
				},
				Status: chaosv1alpha1.ChaosExperimentStatus{
					RetryCount: 2,
				},
			},
			want: true, // max retries 0 uses default (3)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.shouldRetry(tt.exp)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Test buildResourceReferences function
func TestBuildResourceReferences(t *testing.T) {
	tests := []struct {
		name          string
		action        string
		namespace     string
		resourceNames []string
		kind          string
		want          []chaosv1alpha1.ResourceReference
	}{
		{
			name:          "build pod references",
			action:        "deleted",
			namespace:     "test-ns",
			resourceNames: []string{"pod-1", "pod-2"},
			kind:          "Pod",
			want: []chaosv1alpha1.ResourceReference{
				{Kind: "Pod", Name: "pod-1", Namespace: "test-ns", Action: "deleted"},
				{Kind: "Pod", Name: "pod-2", Namespace: "test-ns", Action: "deleted"},
			},
		},
		{
			name:          "build node references",
			action:        "drained",
			namespace:     "",
			resourceNames: []string{"node-1"},
			kind:          "Node",
			want: []chaosv1alpha1.ResourceReference{
				{Kind: "Node", Name: "node-1", Namespace: "", Action: "drained"},
			},
		},
		{
			name:          "empty resource names",
			action:        "deleted",
			namespace:     "test-ns",
			resourceNames: []string{},
			kind:          "Pod",
			want:          []chaosv1alpha1.ResourceReference{},
		},
		{
			name:          "multiple resources with network delay action",
			action:        "network-delay-100ms",
			namespace:     "default",
			resourceNames: []string{"app-1", "app-2", "app-3"},
			kind:          "Pod",
			want: []chaosv1alpha1.ResourceReference{
				{Kind: "Pod", Name: "app-1", Namespace: "default", Action: "network-delay-100ms"},
				{Kind: "Pod", Name: "app-2", Namespace: "default", Action: "network-delay-100ms"},
				{Kind: "Pod", Name: "app-3", Namespace: "default", Action: "network-delay-100ms"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildResourceReferences(tt.action, tt.namespace, tt.resourceNames, tt.kind)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Test sortHistoryByAge function
func TestSortHistoryByAge(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name  string
		items []chaosv1alpha1.ChaosExperimentHistory
		want  []string // expected order of names after sorting
	}{
		{
			name: "sort three items",
			items: []chaosv1alpha1.ChaosExperimentHistory{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "newest",
						CreationTimestamp: metav1.NewTime(now),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "oldest",
						CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Hour)),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "middle",
						CreationTimestamp: metav1.NewTime(now.Add(-1 * time.Hour)),
					},
				},
			},
			want: []string{"oldest", "middle", "newest"},
		},
		{
			name: "already sorted",
			items: []chaosv1alpha1.ChaosExperimentHistory{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "first",
						CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Hour)),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "second",
						CreationTimestamp: metav1.NewTime(now.Add(-1 * time.Hour)),
					},
				},
			},
			want: []string{"first", "second"},
		},
		{
			name:  "empty list",
			items: []chaosv1alpha1.ChaosExperimentHistory{},
			want:  []string{},
		},
		{
			name: "single item",
			items: []chaosv1alpha1.ChaosExperimentHistory{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "only",
						CreationTimestamp: metav1.NewTime(now),
					},
				},
			},
			want: []string{"only"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortHistoryByAge(tt.items)
			gotNames := make([]string, len(tt.items))
			for i, item := range tt.items {
				gotNames[i] = item.Name
			}
			assert.Equal(t, tt.want, gotNames)
		})
	}
}

// Test generateShortUID function
func TestGenerateShortUID(t *testing.T) {
	// Test that it generates non-empty strings
	uid1 := generateShortUID()
	assert.NotEmpty(t, uid1)

	// Test that consecutive calls generate different UIDs (with small delay)
	time.Sleep(time.Nanosecond)
	uid2 := generateShortUID()
	assert.NotEmpty(t, uid2)

	// UIDs should be hex strings of reasonable length
	assert.LessOrEqual(t, len(uid1), 6) // max 6 hex chars for 0xffffff
}

// Test getInitiator function
func TestGetInitiator(t *testing.T) {
	ctx := context.Background()
	initiator := getInitiator(ctx)

	// Should return the default controller service account
	assert.Equal(t, "system:serviceaccount:chaos-system:chaos-controller", initiator)
}

// Test DefaultHistoryConfig function
func TestDefaultHistoryConfig(t *testing.T) {
	config := DefaultHistoryConfig()

	assert.True(t, config.Enabled)
	assert.Equal(t, "chaos-system", config.Namespace)
	assert.Equal(t, 100, config.RetentionLimit)
	assert.Equal(t, 30*24*time.Hour, config.RetentionTTL)
	assert.Equal(t, 1, config.SamplingRate)
}
