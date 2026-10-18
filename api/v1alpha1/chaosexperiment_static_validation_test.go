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
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

const (
	countField     = "spec.count"
	actionPodDelay = "pod-delay"
)

func validExperiment() *ChaosExperiment {
	exp := &ChaosExperiment{}
	exp.Name = "test"
	exp.Spec = ChaosExperimentSpec{
		Action:    "pod-kill",
		Namespace: "default",
		Selector:  map[string]string{"app": "nginx"},
		Count:     1,
	}
	return exp
}

func TestValidateStatic(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*ChaosExperiment)
		wantErr []string // substrings that must all appear; nil means valid
	}{
		{"valid pod-kill", func(*ChaosExperiment) {}, nil},
		{"missing action", func(e *ChaosExperiment) { e.Spec.Action = "" }, []string{"spec.action: required"}},
		{"unknown action", func(e *ChaosExperiment) { e.Spec.Action = "pod-explode" }, []string{`unsupported value "pod-explode"`}},
		{"missing namespace", func(e *ChaosExperiment) { e.Spec.Namespace = "" }, []string{"spec.namespace: required"}},
		{"empty selector", func(e *ChaosExperiment) { e.Spec.Selector = nil }, []string{"spec.selector: required"}},
		{"count too high", func(e *ChaosExperiment) { e.Spec.Count = 101 }, []string{countField, "between 1 and 100"}},
		{"negative count", func(e *ChaosExperiment) { e.Spec.Count = -1 }, []string{countField}},
		{"maxPercentage out of range", func(e *ChaosExperiment) { e.Spec.MaxPercentage = 150 }, []string{"spec.maxPercentage"}},
		{"bad duration format", func(e *ChaosExperiment) { e.Spec.Duration = "ten minutes" }, []string{"spec.duration"}},
		{"bad memory size", func(e *ChaosExperiment) { e.Spec.MemorySize = "1GB" }, []string{"spec.memorySize"}},
		{"bad retryBackoff", func(e *ChaosExperiment) { e.Spec.RetryBackoff = "random" }, []string{"spec.retryBackoff"}},
		{"bad taintEffect", func(e *ChaosExperiment) { e.Spec.TaintEffect = "Nope" }, []string{"spec.taintEffect"}},
		{"bad protocol", func(e *ChaosExperiment) { e.Spec.TargetProtocols = []string{"tcp", "sctp"} }, []string{"targetProtocols[1]"}},
		{
			"reports every schema problem together",
			func(e *ChaosExperiment) { e.Spec.Action = "x"; e.Spec.Namespace = ""; e.Spec.Count = 500 },
			[]string{"spec.action", "spec.namespace", countField},
		},
		// cross-field / action-specific rules shared with the webhook
		{"pod-delay needs duration", func(e *ChaosExperiment) { e.Spec.Action = actionPodDelay }, []string{"duration is required"}},
		{"pod-delay with duration", func(e *ChaosExperiment) { e.Spec.Action = actionPodDelay; e.Spec.Duration = "30s" }, nil},
		{"self dependency", func(e *ChaosExperiment) { e.Spec.DependsOn = []string{"test"} }, []string{"cannot depend on itself"}},
		{"bad schedule", func(e *ChaosExperiment) { e.Spec.Schedule = "not a cron" }, []string{"schedule"}},
		{"valid schedule", func(e *ChaosExperiment) { e.Spec.Schedule = "*/5 * * * *" }, nil},
		{
			"node-disk-fill needs fillPercentage",
			func(e *ChaosExperiment) { e.Spec.Action = "node-disk-fill"; e.Spec.Duration = "1m" },
			[]string{"fillPercentage"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			exp := validExperiment()
			tc.mutate(exp)
			err := exp.ValidateStatic()

			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("expected valid, got error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %v, got nil", tc.wantErr)
			}
			for _, want := range tc.wantErr {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err.Error(), want)
				}
			}
		})
	}
}

// TestSupportedActionsMatchesCRDEnum keeps SupportedActions in sync with the
// kubebuilder Enum marker on Spec.Action, which is what the API server enforces.
func TestSupportedActionsMatchesCRDEnum(t *testing.T) {
	src, err := os.ReadFile("chaosexperiment_types.go")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^\s*// \+kubebuilder:validation:Enum=([a-z;-]+)\s*\n\s*Action string`).FindSubmatch(src)
	if m == nil {
		t.Fatal("could not find the Enum marker on Action")
	}
	want := strings.Split(string(m[1]), ";")
	got := append([]string(nil), SupportedActions...)
	sort.Strings(want)
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("SupportedActions out of sync with CRD enum:\n got  %v\n want %v", got, want)
	}
}
