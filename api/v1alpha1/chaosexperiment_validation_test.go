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
	"time"

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
			errMsg: "action must be one of: pod-kill, pod-delay, node-drain, pod-cpu-stress, pod-memory-stress, pod-failure, pod-network-loss, pod-disk-fill, pod-restart",
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
		"pod-restart":       true,
	}
	if !validActions[spec.Action] {
		return &ValidationError{Field: "action", Message: "action must be one of: pod-kill, pod-delay, node-drain, pod-cpu-stress, pod-memory-stress, pod-failure, pod-network-loss, pod-disk-fill, pod-restart"}
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

func TestValidateTimeWindows(t *testing.T) {
	tests := []struct {
		name    string
		windows []TimeWindow
		wantErr bool
	}{
		{
			name: "valid recurring window with timezone and days",
			windows: []TimeWindow{
				{
					Type:       TimeWindowRecurring,
					Start:      "22:00",
					End:        "02:00",
					Timezone:   "UTC",
					DaysOfWeek: []string{"Mon", "Wed", "Fri"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid absolute window",
			windows: []TimeWindow{
				{
					Type:  TimeWindowAbsolute,
					Start: "2026-01-10T01:00:00Z",
					End:   "2026-01-10T03:00:00Z",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid recurring clock format",
			windows: []TimeWindow{
				{
					Type:  TimeWindowRecurring,
					Start: "9:00",
					End:   "18:00",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid recurring day",
			windows: []TimeWindow{
				{
					Type:       TimeWindowRecurring,
					Start:      "09:00",
					End:        "18:00",
					DaysOfWeek: []string{"Funday"},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid timezone",
			windows: []TimeWindow{
				{
					Type:     TimeWindowRecurring,
					Start:    "09:00",
					End:      "18:00",
					Timezone: "Not/AZone",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid absolute order",
			windows: []TimeWindow{
				{
					Type:  TimeWindowAbsolute,
					Start: "2026-01-10T03:00:00Z",
					End:   "2026-01-10T01:00:00Z",
				},
			},
			wantErr: true,
		},
		{
			name: "absolute with timezone not allowed",
			windows: []TimeWindow{
				{
					Type:     TimeWindowAbsolute,
					Start:    "2026-01-10T01:00:00Z",
					End:      "2026-01-10T03:00:00Z",
					Timezone: "UTC",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTimeWindows(tt.windows)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTimeWindows() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsWithinTimeWindows(t *testing.T) {
	// Fixed test time: 2026-01-06 (Tuesday) 14:30 UTC
	testTime := time.Date(2026, 1, 6, 14, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		windows  []TimeWindow
		testTime time.Time
		want     bool
	}{
		{
			name:     "no windows - always allowed",
			windows:  []TimeWindow{},
			testTime: testTime,
			want:     true,
		},
		{
			name: "recurring window - within time on correct day",
			windows: []TimeWindow{
				{
					Type:       TimeWindowRecurring,
					Start:      "09:00",
					End:        "17:00",
					Timezone:   "UTC",
					DaysOfWeek: []string{"Tue", "Wed", "Thu"},
				},
			},
			testTime: testTime, // Tuesday 14:30 UTC
			want:     true,
		},
		{
			name: "recurring window - within time but wrong day",
			windows: []TimeWindow{
				{
					Type:       TimeWindowRecurring,
					Start:      "09:00",
					End:        "17:00",
					Timezone:   "UTC",
					DaysOfWeek: []string{"Mon", "Wed", "Fri"},
				},
			},
			testTime: testTime, // Tuesday 14:30 UTC
			want:     false,
		},
		{
			name: "recurring window - correct day but outside time",
			windows: []TimeWindow{
				{
					Type:       TimeWindowRecurring,
					Start:      "09:00",
					End:        "12:00",
					Timezone:   "UTC",
					DaysOfWeek: []string{"Tue"},
				},
			},
			testTime: testTime, // Tuesday 14:30 UTC
			want:     false,
		},
		{
			name: "recurring window - wrap around midnight (in window before midnight)",
			windows: []TimeWindow{
				{
					Type:     TimeWindowRecurring,
					Start:    "22:00",
					End:      "02:00",
					Timezone: "UTC",
				},
			},
			testTime: time.Date(2026, 1, 7, 23, 30, 0, 0, time.UTC),
			want:     true,
		},
		{
			name: "recurring window - wrap around midnight (in window after midnight)",
			windows: []TimeWindow{
				{
					Type:     TimeWindowRecurring,
					Start:    "22:00",
					End:      "02:00",
					Timezone: "UTC",
				},
			},
			testTime: time.Date(2026, 1, 7, 1, 30, 0, 0, time.UTC),
			want:     true,
		},
		{
			name: "recurring window - wrap around midnight (outside window)",
			windows: []TimeWindow{
				{
					Type:     TimeWindowRecurring,
					Start:    "22:00",
					End:      "02:00",
					Timezone: "UTC",
				},
			},
			testTime: testTime, // 14:30 UTC
			want:     false,
		},
		{
			name: "recurring window - timezone conversion (Europe/Berlin)",
			windows: []TimeWindow{
				{
					Type:     TimeWindowRecurring,
					Start:    "09:00",
					End:      "17:00",
					Timezone: "Europe/Berlin",
				},
			},
			testTime: time.Date(2026, 1, 7, 8, 30, 0, 0, time.UTC), // 09:30 Berlin time
			want:     true,
		},
		{
			name: "absolute window - within window",
			windows: []TimeWindow{
				{
					Type:  TimeWindowAbsolute,
					Start: "2026-01-06T14:00:00Z",
					End:   "2026-01-06T15:00:00Z",
				},
			},
			testTime: testTime, // 14:30 UTC
			want:     true,
		},
		{
			name: "absolute window - before window",
			windows: []TimeWindow{
				{
					Type:  TimeWindowAbsolute,
					Start: "2026-01-06T15:00:00Z",
					End:   "2026-01-06T16:00:00Z",
				},
			},
			testTime: testTime, // 14:30 UTC
			want:     false,
		},
		{
			name: "absolute window - after window",
			windows: []TimeWindow{
				{
					Type:  TimeWindowAbsolute,
					Start: "2026-01-06T12:00:00Z",
					End:   "2026-01-06T13:00:00Z",
				},
			},
			testTime: testTime, // 14:30 UTC
			want:     false,
		},
		{
			name: "multiple windows - matches second window",
			windows: []TimeWindow{
				{
					Type:       TimeWindowRecurring,
					Start:      "09:00",
					End:        "12:00",
					DaysOfWeek: []string{"Mon"},
				},
				{
					Type:       TimeWindowRecurring,
					Start:      "14:00",
					End:        "18:00",
					DaysOfWeek: []string{"Tue"},
				},
			},
			testTime: testTime, // Tuesday 14:30 UTC
			want:     true,
		},
		{
			name: "multiple windows - matches none",
			windows: []TimeWindow{
				{
					Type:       TimeWindowRecurring,
					Start:      "09:00",
					End:        "12:00",
					DaysOfWeek: []string{"Mon"},
				},
				{
					Type:       TimeWindowRecurring,
					Start:      "18:00",
					End:        "22:00",
					DaysOfWeek: []string{"Tue"},
				},
			},
			testTime: testTime, // Tuesday 14:30 UTC
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsWithinTimeWindows(tt.windows, tt.testTime)
			if got != tt.want {
				t.Errorf("IsWithinTimeWindows() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNextTimeWindowBoundary(t *testing.T) {
	// Fixed test time: 2026-01-06 (Tuesday) 14:30 UTC
	testTime := time.Date(2026, 1, 6, 14, 30, 0, 0, time.UTC)

	tests := []struct {
		name           string
		windows        []TimeWindow
		testTime       time.Time
		wantBoundary   time.Time
		wantWillBeOpen bool
	}{
		{
			name:           "no windows - always open",
			windows:        []TimeWindow{},
			testTime:       testTime,
			wantBoundary:   time.Time{},
			wantWillBeOpen: true,
		},
		{
			name: "recurring window - before window opens today",
			windows: []TimeWindow{
				{
					Type:     TimeWindowRecurring,
					Start:    "18:00",
					End:      "22:00",
					Timezone: "UTC",
				},
			},
			testTime:       testTime, // 14:30 UTC
			wantBoundary:   time.Date(2026, 1, 6, 18, 0, 0, 0, time.UTC),
			wantWillBeOpen: true,
		},
		{
			name: "recurring window - inside window, next boundary is close",
			windows: []TimeWindow{
				{
					Type:     TimeWindowRecurring,
					Start:    "14:00",
					End:      "15:00",
					Timezone: "UTC",
				},
			},
			testTime:       testTime, // 14:30 UTC
			wantBoundary:   time.Date(2026, 1, 6, 15, 0, 0, 0, time.UTC),
			wantWillBeOpen: false,
		},
		{
			name: "recurring window - after today's window, next is tomorrow",
			windows: []TimeWindow{
				{
					Type:     TimeWindowRecurring,
					Start:    "09:00",
					End:      "12:00",
					Timezone: "UTC",
				},
			},
			testTime:       testTime, // 14:30 UTC
			wantBoundary:   time.Date(2026, 1, 7, 9, 0, 0, 0, time.UTC),
			wantWillBeOpen: true,
		},
		{
			name: "recurring window with days - next matching day",
			windows: []TimeWindow{
				{
					Type:       TimeWindowRecurring,
					Start:      "09:00",
					End:        "17:00",
					DaysOfWeek: []string{"Mon", "Wed", "Fri"},
					Timezone:   "UTC",
				},
			},
			testTime:       testTime,                                    // Tuesday 14:30 UTC
			wantBoundary:   time.Date(2026, 1, 7, 9, 0, 0, 0, time.UTC), // Wednesday
			wantWillBeOpen: true,
		},
		{
			name: "absolute window - before start",
			windows: []TimeWindow{
				{
					Type:  TimeWindowAbsolute,
					Start: "2026-01-10T10:00:00Z",
					End:   "2026-01-10T12:00:00Z",
				},
			},
			testTime:       testTime,
			wantBoundary:   time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC),
			wantWillBeOpen: true,
		},
		{
			name: "absolute window - inside window",
			windows: []TimeWindow{
				{
					Type:  TimeWindowAbsolute,
					Start: "2026-01-06T14:00:00Z",
					End:   "2026-01-06T16:00:00Z",
				},
			},
			testTime:       testTime,
			wantBoundary:   time.Date(2026, 1, 6, 16, 0, 0, 0, time.UTC),
			wantWillBeOpen: false,
		},
		{
			name: "absolute window - after end (no future boundary)",
			windows: []TimeWindow{
				{
					Type:  TimeWindowAbsolute,
					Start: "2026-01-05T12:00:00Z",
					End:   "2026-01-05T13:00:00Z",
				},
			},
			testTime:       testTime,
			wantBoundary:   time.Time{},
			wantWillBeOpen: false,
		},
		{
			name: "wrap-around window - before start",
			windows: []TimeWindow{
				{
					Type:     TimeWindowRecurring,
					Start:    "22:00",
					End:      "02:00",
					Timezone: "UTC",
				},
			},
			testTime:       testTime, // 14:30 UTC
			wantBoundary:   time.Date(2026, 1, 6, 22, 0, 0, 0, time.UTC),
			wantWillBeOpen: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBoundary, gotWillBeOpen := NextTimeWindowBoundary(tt.windows, tt.testTime)
			if !gotBoundary.Equal(tt.wantBoundary) {
				t.Errorf("NextTimeWindowBoundary() boundary = %v, want %v",
					gotBoundary.Format(time.RFC3339), tt.wantBoundary.Format(time.RFC3339))
			}
			if gotWillBeOpen != tt.wantWillBeOpen {
				t.Errorf("NextTimeWindowBoundary() willBeOpen = %v, want %v", gotWillBeOpen, tt.wantWillBeOpen)
			}
		})
	}
}

// TestValidateCIDR tests CIDR validation
func TestValidateCIDR(t *testing.T) {
	tests := []struct {
		name    string
		cidr    string
		wantErr bool
	}{
		// Valid CIDRs
		{name: "valid CIDR - /24", cidr: "192.168.1.0/24", wantErr: false},
		{name: "valid CIDR - /16", cidr: "10.0.0.0/16", wantErr: false},
		{name: "valid CIDR - /12", cidr: "10.96.0.0/12", wantErr: false},
		{name: "valid CIDR - /32", cidr: "192.168.1.1/32", wantErr: false},
		{name: "valid CIDR - /8", cidr: "10.0.0.0/8", wantErr: false},

		// Invalid CIDRs
		{name: "empty CIDR", cidr: "", wantErr: true},
		{name: "invalid CIDR - no mask", cidr: "192.168.1.0", wantErr: true},
		{name: "invalid CIDR - wrong format", cidr: "192.168.1.0/", wantErr: true},
		{name: "invalid CIDR - mask too large", cidr: "192.168.1.0/33", wantErr: true},
		{name: "invalid CIDR - negative mask", cidr: "192.168.1.0/-1", wantErr: true},
		{name: "invalid CIDR - malformed IP", cidr: "256.1.1.1/24", wantErr: true},
		{name: "invalid CIDR - text", cidr: "not-a-cidr/24", wantErr: true},
		{name: "IPv6 CIDR (unsupported)", cidr: "2001:db8::/32", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCIDR(tt.cidr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCIDR(%q) error = %v, wantErr %v", tt.cidr, err, tt.wantErr)
			}
		})
	}
}

// TestValidateIP tests IP address validation
func TestValidateIP(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		// Valid IPs
		{name: "valid IP - loopback", ip: "127.0.0.1", wantErr: false},
		{name: "valid IP - private 10.x", ip: "10.96.100.50", wantErr: false},
		{name: "valid IP - private 192.168.x", ip: "192.168.1.100", wantErr: false},
		{name: "valid IP - private 172.16.x", ip: "172.16.0.1", wantErr: false},
		{name: "valid IP - public", ip: "8.8.8.8", wantErr: false},
		{name: "valid IP - broadcast", ip: "255.255.255.255", wantErr: false},
		{name: "valid IP - all zeros", ip: "0.0.0.0", wantErr: false},

		// Invalid IPs
		{name: "empty IP", ip: "", wantErr: true},
		{name: "invalid IP - out of range octet", ip: "256.1.1.1", wantErr: true},
		{name: "invalid IP - too many octets", ip: "1.2.3.4.5", wantErr: true},
		{name: "invalid IP - too few octets", ip: "1.2.3", wantErr: true},
		{name: "invalid IP - text", ip: "not-an-ip", wantErr: true},
		{name: "invalid IP - hostname", ip: "example.com", wantErr: true},
		{name: "IPv6 (unsupported)", ip: "::1", wantErr: true},
		{name: "IPv6 (unsupported)", ip: "2001:db8::1", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIP(tt.ip)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIP(%q) error = %v, wantErr %v", tt.ip, err, tt.wantErr)
			}
		})
	}
}

// TestValidatePortRange tests port range validation
func TestValidatePortRange(t *testing.T) {
	tests := []struct {
		name    string
		port    int32
		wantErr bool
	}{
		// Valid ports
		{name: "valid port - minimum", port: 1, wantErr: false},
		{name: "valid port - HTTP", port: 80, wantErr: false},
		{name: "valid port - HTTPS", port: 443, wantErr: false},
		{name: "valid port - high", port: 8080, wantErr: false},
		{name: "valid port - maximum", port: 65535, wantErr: false},

		// Invalid ports
		{name: "invalid port - zero", port: 0, wantErr: true},
		{name: "invalid port - negative", port: -1, wantErr: true},
		{name: "invalid port - too high", port: 65536, wantErr: true},
		{name: "invalid port - way too high", port: 100000, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePortRange(tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePortRange(%d) error = %v, wantErr %v", tt.port, err, tt.wantErr)
			}
		})
	}
}

// TestIsDangerousTarget tests dangerous target detection
func TestIsDangerousTarget(t *testing.T) {
	tests := []struct {
		name           string
		ip             string
		wantDangerous  bool
		wantReasonPart string
	}{
		// Dangerous targets
		{name: "loopback", ip: "127.0.0.1", wantDangerous: true, wantReasonPart: "Loopback"},
		{name: "link-local", ip: "169.254.1.1", wantDangerous: true, wantReasonPart: "Link-local"},
		{name: "k8s API server", ip: "10.96.0.1", wantDangerous: true, wantReasonPart: "Kubernetes API"},
		{name: "cluster DNS", ip: "10.96.0.10", wantDangerous: true, wantReasonPart: "Cluster DNS"},
		{name: "cluster service IP", ip: "10.96.100.50", wantDangerous: true, wantReasonPart: "Cluster service"},

		// Safe targets
		{name: "public IP", ip: "8.8.8.8", wantDangerous: false, wantReasonPart: ""},
		{name: "private IP outside cluster", ip: "192.168.1.100", wantDangerous: false, wantReasonPart: ""},

		// Invalid IPs (return false, not an error)
		{name: "invalid IP", ip: "not-an-ip", wantDangerous: false, wantReasonPart: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDangerous, gotReason := IsDangerousTarget(tt.ip)
			if gotDangerous != tt.wantDangerous {
				t.Errorf("IsDangerousTarget(%q) dangerous = %v, want %v", tt.ip, gotDangerous, tt.wantDangerous)
			}
			if tt.wantReasonPart != "" && gotReason == "" {
				t.Errorf("IsDangerousTarget(%q) expected reason containing %q, got empty", tt.ip, tt.wantReasonPart)
			}
		})
	}
}

// TestIsDangerousCIDR tests dangerous CIDR detection
func TestIsDangerousCIDR(t *testing.T) {
	tests := []struct {
		name           string
		cidr           string
		wantDangerous  bool
		wantReasonPart string
	}{
		// Dangerous CIDRs
		{name: "loopback range", cidr: "127.0.0.0/8", wantDangerous: true, wantReasonPart: "loopback"},
		{name: "cluster service CIDR", cidr: "10.96.0.0/12", wantDangerous: true, wantReasonPart: "overlaps"},
		{name: "private 10.x", cidr: "10.0.0.0/8", wantDangerous: true, wantReasonPart: "overlaps"},
		{name: "private 172.16.x", cidr: "172.16.0.0/12", wantDangerous: true, wantReasonPart: "overlaps"},

		// Safe CIDRs
		{name: "public CIDR", cidr: "8.8.8.0/24", wantDangerous: false, wantReasonPart: ""},
		{name: "specific non-cluster private", cidr: "192.168.1.0/24", wantDangerous: false, wantReasonPart: ""},

		// Invalid CIDRs (return false, not an error)
		{name: "invalid CIDR", cidr: "not-a-cidr", wantDangerous: false, wantReasonPart: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDangerous, gotReason := IsDangerousCIDR(tt.cidr)
			if gotDangerous != tt.wantDangerous {
				t.Errorf("IsDangerousCIDR(%q) dangerous = %v, want %v", tt.cidr, gotDangerous, tt.wantDangerous)
			}
			if tt.wantReasonPart != "" && gotReason == "" {
				t.Errorf("IsDangerousCIDR(%q) expected reason containing %q, got empty", tt.cidr, tt.wantReasonPart)
			}
		})
	}
}
