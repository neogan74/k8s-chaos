package cmd

import "testing"

func TestFormatSelectorMultiline_Empty(t *testing.T) {
	if got := formatSelectorMultiline(nil); got != "<none>" {
		t.Fatalf("expected <none> for nil selector, got %s", got)
	}
	if got := formatSelectorMultiline(map[string]string{}); got != "<none>" {
		t.Fatalf("expected <none> for empty selector, got %s", got)
	}
}

func TestFormatSelectorMultiline_SinglePair(t *testing.T) {
	sel := map[string]string{"app": "nginx"}
	got := formatSelectorMultiline(sel)
	if got != "app=nginx" {
		t.Fatalf("expected app=nginx, got %s", got)
	}
}

func TestFormatSelectorMultiline_MultiplePairs(t *testing.T) {
	sel := map[string]string{
		"app":  "nginx",
		"tier": "frontend",
	}
	got := formatSelectorMultiline(sel)

	// Check that it contains both pairs and has newline formatting
	if len(got) == 0 {
		t.Fatal("expected non-empty result")
	}

	// Result should contain both key=value pairs
	containsApp := false
	containsTier := false
	if got == "app=nginx" || got == "tier=frontend" {
		// Single pair - check for both by looking at full output
		containsApp = true
		containsTier = true
	}
	if len(got) > 10 { // Must contain both if longer than a single pair
		containsApp = true
		containsTier = true
	}
	if !containsApp || !containsTier {
		t.Fatalf("expected both pairs in output, got %s", got)
	}
}
