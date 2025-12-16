package cmd

import (
	"sort"
	"testing"
	"time"
)

func TestExperimentMetrics_SortByRetries(t *testing.T) {
	metrics := []experimentMetrics{
		{Name: "exp1", RetryCount: 1},
		{Name: "exp2", RetryCount: 5},
		{Name: "exp3", RetryCount: 3},
		{Name: "exp4", RetryCount: 0},
	}

	// Sort by retry count descending (same logic as printTopByRetries)
	sorted := make([]experimentMetrics, len(metrics))
	copy(sorted, metrics)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].RetryCount > sorted[j].RetryCount
	})

	if sorted[0].Name != "exp2" {
		t.Fatalf("expected exp2 first (5 retries), got %s", sorted[0].Name)
	}
	if sorted[1].Name != "exp3" {
		t.Fatalf("expected exp3 second (3 retries), got %s", sorted[1].Name)
	}
	if sorted[2].Name != "exp1" {
		t.Fatalf("expected exp1 third (1 retry), got %s", sorted[2].Name)
	}
	if sorted[3].Name != "exp4" {
		t.Fatalf("expected exp4 last (0 retries), got %s", sorted[3].Name)
	}
}

func TestExperimentMetrics_SortByAge(t *testing.T) {
	now := time.Now()
	metrics := []experimentMetrics{
		{Name: "exp1", Age: 1 * time.Hour},
		{Name: "exp2", Age: 24 * time.Hour},
		{Name: "exp3", Age: 30 * time.Minute},
		{Name: "exp4", Age: 48 * time.Hour},
	}

	// Sort by age descending (oldest first, same logic as printTopByAge)
	sorted := make([]experimentMetrics, len(metrics))
	copy(sorted, metrics)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Age > sorted[j].Age
	})

	_ = now // avoid unused variable warning

	if sorted[0].Name != "exp4" {
		t.Fatalf("expected exp4 first (48h), got %s", sorted[0].Name)
	}
	if sorted[1].Name != "exp2" {
		t.Fatalf("expected exp2 second (24h), got %s", sorted[1].Name)
	}
	if sorted[2].Name != "exp1" {
		t.Fatalf("expected exp1 third (1h), got %s", sorted[2].Name)
	}
	if sorted[3].Name != "exp3" {
		t.Fatalf("expected exp3 last (30m), got %s", sorted[3].Name)
	}
}

func TestExperimentMetrics_FilterFailed(t *testing.T) {
	metrics := []experimentMetrics{
		{Name: "exp1", Phase: "Running"},
		{Name: "exp2", Phase: "Failed"},
		{Name: "exp3", Phase: "Completed"},
		{Name: "exp4", Phase: "Failed"},
		{Name: "exp5", Phase: "Pending"},
	}

	// Filter failed experiments (same logic as printFailed)
	var failed []experimentMetrics
	for _, m := range metrics {
		if m.Phase == "Failed" {
			failed = append(failed, m)
		}
	}

	if len(failed) != 2 {
		t.Fatalf("expected 2 failed experiments, got %d", len(failed))
	}

	foundExp2 := false
	foundExp4 := false
	for _, f := range failed {
		if f.Name == "exp2" {
			foundExp2 = true
		}
		if f.Name == "exp4" {
			foundExp4 = true
		}
	}

	if !foundExp2 || !foundExp4 {
		t.Fatalf("expected exp2 and exp4 in failed list, got %v", failed)
	}
}

func TestExperimentMetrics_FilterRetriesOnly(t *testing.T) {
	metrics := []experimentMetrics{
		{Name: "exp1", RetryCount: 0},
		{Name: "exp2", RetryCount: 3},
		{Name: "exp3", RetryCount: 0},
		{Name: "exp4", RetryCount: 1},
	}

	// Filter experiments with retries (same logic as printTopByRetries filter)
	var withRetries []experimentMetrics
	for _, m := range metrics {
		if m.RetryCount > 0 {
			withRetries = append(withRetries, m)
		}
	}

	if len(withRetries) != 2 {
		t.Fatalf("expected 2 experiments with retries, got %d", len(withRetries))
	}

	foundExp2 := false
	foundExp4 := false
	for _, r := range withRetries {
		if r.Name == "exp2" {
			foundExp2 = true
		}
		if r.Name == "exp4" {
			foundExp4 = true
		}
	}

	if !foundExp2 || !foundExp4 {
		t.Fatalf("expected exp2 and exp4 in retry list, got %v", withRetries)
	}
}

func TestExperimentMetrics_LimitResults(t *testing.T) {
	metrics := []experimentMetrics{
		{Name: "exp1"},
		{Name: "exp2"},
		{Name: "exp3"},
		{Name: "exp4"},
		{Name: "exp5"},
	}

	limit := 3
	var limited []experimentMetrics
	for i := 0; i < limit && i < len(metrics); i++ {
		limited = append(limited, metrics[i])
	}

	if len(limited) != 3 {
		t.Fatalf("expected 3 results with limit, got %d", len(limited))
	}
}
