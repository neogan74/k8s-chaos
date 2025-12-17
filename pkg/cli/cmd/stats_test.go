package cmd

import (
	"testing"

	chaosv1alpha1 "github.com/neogan74/k8s-chaos/api/v1alpha1"
)

func TestCalculateStats_Empty(t *testing.T) {
	s := calculateStats(nil)
	if s.Total != 0 {
		t.Fatalf("expected 0 total, got %d", s.Total)
	}
	if len(s.ByAction) != 0 {
		t.Fatalf("expected empty ByAction map, got %v", s.ByAction)
	}
}

func TestCalculateStats_CountsByPhase(t *testing.T) {
	experiments := []chaosv1alpha1.ChaosExperiment{
		{
			Spec:   chaosv1alpha1.ChaosExperimentSpec{Action: "pod-kill"},
			Status: chaosv1alpha1.ChaosExperimentStatus{Phase: "Running"},
		},
		{
			Spec:   chaosv1alpha1.ChaosExperimentSpec{Action: "pod-kill"},
			Status: chaosv1alpha1.ChaosExperimentStatus{Phase: "Running"},
		},
		{
			Spec:   chaosv1alpha1.ChaosExperimentSpec{Action: "pod-delay"},
			Status: chaosv1alpha1.ChaosExperimentStatus{Phase: "Completed"},
		},
		{
			Spec:   chaosv1alpha1.ChaosExperimentSpec{Action: "node-drain"},
			Status: chaosv1alpha1.ChaosExperimentStatus{Phase: "Failed"},
		},
		{
			Spec:   chaosv1alpha1.ChaosExperimentSpec{Action: "pod-failure"},
			Status: chaosv1alpha1.ChaosExperimentStatus{Phase: "Pending"},
		},
	}

	s := calculateStats(experiments)

	if s.Total != 5 {
		t.Fatalf("expected 5 total, got %d", s.Total)
	}
	if s.Running != 2 {
		t.Fatalf("expected 2 running, got %d", s.Running)
	}
	if s.Completed != 1 {
		t.Fatalf("expected 1 completed, got %d", s.Completed)
	}
	if s.Failed != 1 {
		t.Fatalf("expected 1 failed, got %d", s.Failed)
	}
	if s.Pending != 1 {
		t.Fatalf("expected 1 pending, got %d", s.Pending)
	}
}

func TestCalculateStats_CountsByAction(t *testing.T) {
	experiments := []chaosv1alpha1.ChaosExperiment{
		{Spec: chaosv1alpha1.ChaosExperimentSpec{Action: "pod-kill"}},
		{Spec: chaosv1alpha1.ChaosExperimentSpec{Action: "pod-kill"}},
		{Spec: chaosv1alpha1.ChaosExperimentSpec{Action: "pod-kill"}},
		{Spec: chaosv1alpha1.ChaosExperimentSpec{Action: "pod-delay"}},
		{Spec: chaosv1alpha1.ChaosExperimentSpec{Action: "node-drain"}},
	}

	s := calculateStats(experiments)

	if s.ByAction["pod-kill"] != 3 {
		t.Fatalf("expected 3 pod-kill, got %d", s.ByAction["pod-kill"])
	}
	if s.ByAction["pod-delay"] != 1 {
		t.Fatalf("expected 1 pod-delay, got %d", s.ByAction["pod-delay"])
	}
	if s.ByAction["node-drain"] != 1 {
		t.Fatalf("expected 1 node-drain, got %d", s.ByAction["node-drain"])
	}
}

func TestCalculateStats_WithRetry(t *testing.T) {
	experiments := []chaosv1alpha1.ChaosExperiment{
		{Spec: chaosv1alpha1.ChaosExperimentSpec{Action: "pod-kill", MaxRetries: 3}},
		{Spec: chaosv1alpha1.ChaosExperimentSpec{Action: "pod-kill", MaxRetries: 0}},
		{Spec: chaosv1alpha1.ChaosExperimentSpec{Action: "pod-delay", MaxRetries: 5}},
	}

	s := calculateStats(experiments)

	if s.WithRetry != 2 {
		t.Fatalf("expected 2 with retry, got %d", s.WithRetry)
	}
}

func TestCalculateStats_TimeLimited(t *testing.T) {
	experiments := []chaosv1alpha1.ChaosExperiment{
		{Spec: chaosv1alpha1.ChaosExperimentSpec{Action: "pod-kill", ExperimentDuration: "1h"}},
		{Spec: chaosv1alpha1.ChaosExperimentSpec{Action: "pod-kill", ExperimentDuration: ""}},
		{Spec: chaosv1alpha1.ChaosExperimentSpec{Action: "pod-delay", ExperimentDuration: "30m"}},
		{Spec: chaosv1alpha1.ChaosExperimentSpec{Action: "node-drain", ExperimentDuration: "2h"}},
	}

	s := calculateStats(experiments)

	if s.TimeLimited != 3 {
		t.Fatalf("expected 3 time-limited, got %d", s.TimeLimited)
	}
}
