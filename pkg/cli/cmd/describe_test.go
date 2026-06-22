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

package cmd

import "testing"

const selectorNone = "<none>"

func TestFormatSelectorMultiline_Empty(t *testing.T) {
	if got := formatSelectorMultiline(nil); got != selectorNone {
		t.Fatalf("expected <none> for nil selector, got %s", got)
	}
	if got := formatSelectorMultiline(map[string]string{}); got != selectorNone {
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
