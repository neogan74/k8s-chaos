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

import (
	"testing"
	"time"
)

func TestFormatSelector(t *testing.T) {
	if got := formatSelector(nil); got != selectorNone {
		t.Fatalf("expected <none> for nil selector, got %s", got)
	}
	if got := formatSelector(map[string]string{}); got != selectorNone {
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
