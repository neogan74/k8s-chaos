package cmd

import (
	"testing"
	"time"
)

func TestFormatSelector(t *testing.T) {
	if got := formatSelector(nil); got != "<none>" {
		t.Fatalf("expected <none> for nil selector, got %s", got)
	}
	if got := formatSelector(map[string]string{}); got != "<none>" {
		t.Fatalf("expected <none> for empty selector, got %s", got)
	}
	if got := formatSelector(map[string]string{"app": "demo"}); got != "app=demo" {
		t.Fatalf("expected app=demo, got %s", got)
	}

	// Long selector should be truncated to 30 chars and end with "..."
	longSel := map[string]string{
		"very-long-key-1": "longvalue1",
		"very-long-key-2": "longvalue2",
		"role":            "frontend",
	}
	got := formatSelector(longSel)
	if len(got) > 30 {
		t.Fatalf("expected truncated selector length <=30, got %d (%s)", len(got), got)
	}
	if got[len(got)-3:] != "..." {
		t.Fatalf("expected truncated selector to end with ..., got %s", got)
	}
}

func TestFormatAge(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name     string
		then     time.Time
		expected string
	}{
		{"seconds", now.Add(-10 * time.Second), "s"},
		{"minutes", now.Add(-5 * time.Minute), "m"},
		{"hours", now.Add(-2 * time.Hour), "h"},
		{"days", now.Add(-48 * time.Hour), "d"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatAge(tc.then)
			if len(got) < 2 {
				t.Fatalf("expected formatted age with unit, got %s", got)
			}
			unit := got[len(got)-1:]
			if unit != tc.expected {
				t.Fatalf("expected unit %s, got %s (value %s)", tc.expected, unit, got)
			}
		})
	}
}
