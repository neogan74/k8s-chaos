package controller

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			name:     "500 milliseconds",
			duration: "500ms",
			want:     500,
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
			name: "Static pod with mirror annotation",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
					Annotations: map[string]string{
						"kubernetes.io/config.mirror": "abc123",
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