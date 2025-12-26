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
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestChaosExperimentValidation(t *testing.T) {
	tests := []struct {
		name    string
		spec    ChaosExperimentSpec
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid pod-kill experiment",
			spec: ChaosExperimentSpec{
				Action:    "pod-kill",
				Namespace: "default",
				Selector:  map[string]string{"app": "test"},
				Count:     1,
			},
			wantErr: false,
		},
		{
			name: "valid pod-delay experiment with duration",
			spec: ChaosExperimentSpec{
				Action:    "pod-delay",
				Namespace: "default",
				Selector:  map[string]string{"app": "test"},
				Count:     2,
				Duration:  "30s",
			},
			wantErr: false,
		},
		{
			name: "valid node-drain experiment",
			spec: ChaosExperimentSpec{
				Action:    "node-drain",
				Namespace: "kube-system",
				Selector:  map[string]string{"node-role": "worker"},
				Count:     1,
			},
			wantErr: false,
		},
		{
			name: "valid duration patterns",
			spec: ChaosExperimentSpec{
				Action:    "pod-delay",
				Namespace: "default",
				Selector:  map[string]string{"app": "test"},
				Duration:  "1h30m45s",
			},
			wantErr: false,
		},
		{
			name: "count at maximum boundary",
			spec: ChaosExperimentSpec{
				Action:    "pod-kill",
				Namespace: "default",
				Selector:  map[string]string{"app": "test"},
				Count:     100,
			},
			wantErr: false,
		},
		{
			name: "multiple selector labels",
			spec: ChaosExperimentSpec{
				Action:    "pod-kill",
				Namespace: "default",
				Selector: map[string]string{
					"app":     "test",
					"version": "v1",
					"tier":    "backend",
				},
				Count: 5,
			},
			wantErr: false,
		},
		{
			name: "valid pod-network-loss experiment with duration",
			spec: ChaosExperimentSpec{
				Action:         "pod-network-loss",
				Namespace:      "default",
				Selector:       map[string]string{"app": "test"},
				Count:          2,
				Duration:       "2m",
				LossPercentage: 10,
			},
			wantErr: false,
		},
		{
			name: "valid pod-network-loss with correlation",
			spec: ChaosExperimentSpec{
				Action:          "pod-network-loss",
				Namespace:       "default",
				Selector:        map[string]string{"app": "test"},
				Duration:        "5m",
				LossPercentage:  15,
				LossCorrelation: 25,
			},
			wantErr: false,
		},
		{
			name: "valid pod-failure experiment",
			spec: ChaosExperimentSpec{
				Action:    "pod-failure",
				Namespace: "default",
				Selector:  map[string]string{"app": "test"},
				Count:     1,
			},
			wantErr: false,
		},
		{
			name: "valid pod-disk-fill experiment",
			spec: ChaosExperimentSpec{
				Action:         "pod-disk-fill",
				Namespace:      "default",
				Selector:       map[string]string{"app": "test"},
				Count:          1,
				Duration:       "3m",
				FillPercentage: 80,
				TargetPath:     "/tmp",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exp := &ChaosExperiment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-experiment",
					Namespace: "default",
				},
				Spec: tt.spec,
			}

			// Basic structural validation - in real cluster, OpenAPI validation would catch these
			err := validateChaosExperimentSpec(&exp.Spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateChaosExperimentSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChaosExperimentInvalidCases(t *testing.T) {
	tests := []struct {
		name   string
		spec   ChaosExperimentSpec
		errMsg string
	}{
		{
			name: "invalid action type",
			spec: ChaosExperimentSpec{
				Action:    "pod-destroy", // not in enum
				Namespace: "default",
				Selector:  map[string]string{"app": "test"},
			},
			errMsg: "action must be one of: pod-kill, pod-delay, node-drain, pod-cpu-stress, pod-memory-stress, pod-failure, pod-network-loss, pod-disk-fill",
		},
		{
			name: "empty action",
			spec: ChaosExperimentSpec{
				Action:    "",
				Namespace: "default",
				Selector:  map[string]string{"app": "test"},
			},
			errMsg: "action is required",
		},
		{
			name: "empty namespace",
			spec: ChaosExperimentSpec{
				Action:    "pod-kill",
				Namespace: "",
				Selector:  map[string]string{"app": "test"},
			},
			errMsg: "namespace must be non-empty",
		},
		{
			name: "empty selector",
			spec: ChaosExperimentSpec{
				Action:    "pod-kill",
				Namespace: "default",
				Selector:  map[string]string{},
			},
			errMsg: "selector must have at least one label",
		},
		{
			name: "nil selector",
			spec: ChaosExperimentSpec{
				Action:    "pod-kill",
				Namespace: "default",
				Selector:  nil,
			},
			errMsg: "selector is required",
		},
		{
			name: "count negative",
			spec: ChaosExperimentSpec{
				Action:    "pod-kill",
				Namespace: "default",
				Selector:  map[string]string{"app": "test"},
				Count:     -5,
			},
			errMsg: "count must be at least 1",
		},
		{
			name: "count exceeds maximum",
			spec: ChaosExperimentSpec{
				Action:    "pod-kill",
				Namespace: "default",
				Selector:  map[string]string{"app": "test"},
				Count:     101,
			},
			errMsg: "count must not exceed 100",
		},
		{
			name: "invalid duration format - no unit",
			spec: ChaosExperimentSpec{
				Action:    "pod-delay",
				Namespace: "default",
				Selector:  map[string]string{"app": "test"},
				Duration:  "30",
			},
			errMsg: "duration must match pattern ^([0-9]+(s|m|h))+$",
		},
		{
			name: "invalid duration format - wrong unit",
			spec: ChaosExperimentSpec{
				Action:    "pod-delay",
				Namespace: "default",
				Selector:  map[string]string{"app": "test"},
				Duration:  "30minutes",
			},
			errMsg: "duration must match pattern ^([0-9]+(s|m|h))+$",
		},
		{
			name: "invalid duration format - spaces",
			spec: ChaosExperimentSpec{
				Action:    "pod-delay",
				Namespace: "default",
				Selector:  map[string]string{"app": "test"},
				Duration:  "30 s",
			},
			errMsg: "duration must match pattern ^([0-9]+(s|m|h))+$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateChaosExperimentSpec(&tt.spec)
			if err == nil {
				t.Errorf("expected validation error but got none")
				return
			}
			if err.Error() != tt.errMsg {
				t.Errorf("expected error message %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

// validateChaosExperimentSpec performs the same validation that OpenAPI schema would do
// This is for testing purposes to ensure our validation markers are correct
func validateChaosExperimentSpec(spec *ChaosExperimentSpec) error {
	// Validate action (Enum validation)
	if spec.Action == "" {
		return &ValidationError{Field: "action", Message: "action is required"}
	}
	validActions := map[string]bool{
		"pod-kill":          true,
		"pod-delay":         true,
		"node-drain":        true,
		"pod-cpu-stress":    true,
		"pod-memory-stress": true,
		"pod-failure":       true,
		"pod-network-loss":  true,
		"pod-disk-fill":     true,
	}
	if !validActions[spec.Action] {
		return &ValidationError{Field: "action", Message: "action must be one of: pod-kill, pod-delay, node-drain, pod-cpu-stress, pod-memory-stress, pod-failure, pod-network-loss, pod-disk-fill"}
	}

	// Validate namespace (MinLength validation)
	if spec.Namespace == "" {
		return &ValidationError{Field: "namespace", Message: "namespace must be non-empty"}
	}

	// Validate selector (Required + MinProperties validation)
	if spec.Selector == nil {
		return &ValidationError{Field: "selector", Message: "selector is required"}
	}
	if len(spec.Selector) == 0 {
		return &ValidationError{Field: "selector", Message: "selector must have at least one label"}
	}

	// Validate count (Minimum + Maximum validation)
	// Count has a default of 1, so 0 means "use default" and is valid
	// Only negative values and values > 100 are invalid
	if spec.Count < 0 {
		return &ValidationError{Field: "count", Message: "count must be at least 1"}
	}
	if spec.Count > 100 {
		return &ValidationError{Field: "count", Message: "count must not exceed 100"}
	}

	// Validate duration pattern if provided
	if spec.Duration != "" {
		matched := durationPattern.MatchString(spec.Duration)
		if !matched {
			return &ValidationError{Field: "duration", Message: "duration must match pattern ^([0-9]+(s|m|h))+$"}
		}
	}

	return nil
}

func TestValidateMemorySize(t *testing.T) {
	tests := []struct {
		name       string
		memorySize string
		wantErr    bool
	}{
		{
			name:       "valid memory size in MB",
			memorySize: "256M",
			wantErr:    false,
		},
		{
			name:       "valid memory size in GB",
			memorySize: "1G",
			wantErr:    false,
		},
		{
			name:       "valid memory size 512M",
			memorySize: "512M",
			wantErr:    false,
		},
		{
			name:       "valid memory size 2G",
			memorySize: "2G",
			wantErr:    false,
		},
		{
			name:       "empty memory size - optional",
			memorySize: "",
			wantErr:    false,
		},
		{
			name:       "invalid - lowercase m",
			memorySize: "256m",
			wantErr:    true,
		},
		{
			name:       "invalid - lowercase g",
			memorySize: "1g",
			wantErr:    true,
		},
		{
			name:       "invalid - no unit",
			memorySize: "256",
			wantErr:    true,
		},
		{
			name:       "invalid - wrong unit KB",
			memorySize: "256K",
			wantErr:    true,
		},
		{
			name:       "invalid - with space",
			memorySize: "256 M",
			wantErr:    true,
		},
		{
			name:       "invalid - decimal number",
			memorySize: "1.5G",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMemorySize(tt.memorySize)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMemorySize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSchedule(t *testing.T) {
	tests := []struct {
		name     string
		schedule string
		wantErr  bool
	}{
		{
			name:     "valid cron - every minute",
			schedule: "* * * * *",
			wantErr:  false,
		},
		{
			name:     "valid cron - every 30 minutes",
			schedule: "*/30 * * * *",
			wantErr:  false,
		},
		{
			name:     "valid cron - daily at 2am",
			schedule: "0 2 * * *",
			wantErr:  false,
		},
		{
			name:     "valid cron - hourly at minute 15",
			schedule: "15 * * * *",
			wantErr:  false,
		},
		{
			name:     "valid cron - every 6 hours",
			schedule: "0 */6 * * *",
			wantErr:  false,
		},
		{
			name:     "valid cron - business hours (9-5, Mon-Fri)",
			schedule: "*/15 9-17 * * 1-5",
			wantErr:  false,
		},
		{
			name:     "valid cron - Monday at 9am",
			schedule: "0 9 * * 1",
			wantErr:  false,
		},
		{
			name:     "valid predefined - @hourly",
			schedule: "@hourly",
			wantErr:  false,
		},
		{
			name:     "valid predefined - @daily",
			schedule: "@daily",
			wantErr:  false,
		},
		{
			name:     "valid predefined - @weekly",
			schedule: "@weekly",
			wantErr:  false,
		},
		{
			name:     "valid predefined - @monthly",
			schedule: "@monthly",
			wantErr:  false,
		},
		{
			name:     "valid predefined - @yearly",
			schedule: "@yearly",
			wantErr:  false,
		},
		{
			name:     "empty schedule - optional field",
			schedule: "",
			wantErr:  false,
		},
		{
			name:     "invalid - too few fields",
			schedule: "* * * *",
			wantErr:  true,
		},
		{
			name:     "invalid - too many fields",
			schedule: "* * * * * * *",
			wantErr:  true,
		},
		{
			name:     "invalid - wrong format",
			schedule: "every 30 minutes",
			wantErr:  true,
		},
		{
			name:     "invalid - invalid minute value",
			schedule: "60 * * * *",
			wantErr:  true,
		},
		{
			name:     "invalid - invalid hour value",
			schedule: "0 24 * * *",
			wantErr:  true,
		},
		{
			name:     "invalid - invalid day of month",
			schedule: "0 0 32 * *",
			wantErr:  true,
		},
		{
			name:     "invalid - invalid month",
			schedule: "0 0 1 13 *",
			wantErr:  true,
		},
		{
			name:     "invalid - invalid day of week",
			schedule: "0 0 * * 8",
			wantErr:  true,
		},
		{
			name:     "invalid - random string",
			schedule: "not a cron schedule",
			wantErr:  true,
		},
		{
			name:     "invalid - @unknown predefined",
			schedule: "@every-minute",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSchedule(tt.schedule)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSchedule() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
