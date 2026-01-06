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
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

// durationPattern matches the pattern used in the Duration field validation
// Pattern: ^([0-9]+(s|m|h))+$
var durationPattern = regexp.MustCompile(`^([0-9]+(s|m|h))+$`)

// memorySizePattern matches the pattern used in the MemorySize field validation
// Pattern: ^[0-9]+[MG]$
var memorySizePattern = regexp.MustCompile(`^[0-9]+[MG]$`)

// timeWindowClockPattern matches 24h HH:MM format.
var timeWindowClockPattern = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

// ValidationError represents a validation error for a specific field
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// ValidActions is the list of supported chaos actions
var ValidActions = []string{"pod-kill", "pod-delay", "node-drain", "pod-cpu-stress", "pod-memory-stress", "pod-failure", "pod-network-loss", "pod-disk-fill", "pod-restart"}

// IsValidAction checks if the given action is valid
func IsValidAction(action string) bool {
	for _, valid := range ValidActions {
		if action == valid {
			return true
		}
	}
	return false
}

// ValidateDurationFormat validates that a duration string matches the expected pattern
func ValidateDurationFormat(duration string) error {
	if duration == "" {
		return nil // Duration is optional
	}
	if !durationPattern.MatchString(duration) {
		return fmt.Errorf("duration must match pattern ^([0-9]+(s|m|h))+$, got: %s", duration)
	}
	return nil
}

// ValidateMemorySize validates that a memory size string matches the expected pattern
func ValidateMemorySize(memorySize string) error {
	if memorySize == "" {
		return nil // MemorySize is optional
	}
	if !memorySizePattern.MatchString(memorySize) {
		return fmt.Errorf("memorySize must match pattern ^[0-9]+[MG]$, got: %s", memorySize)
	}
	return nil
}

// ValidateSchedule validates that a cron schedule expression is valid
func ValidateSchedule(schedule string) error {
	if schedule == "" {
		return nil // Schedule is optional
	}

	// Create a cron parser that supports standard cron format and special strings (@hourly, @daily, etc.)
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

	// Try to parse the schedule
	_, err := parser.Parse(schedule)
	if err != nil {
		return fmt.Errorf("invalid cron schedule %q: %w", schedule, err)
	}

	return nil
}

// ValidateTimeWindows validates the time window configuration.
func ValidateTimeWindows(windows []TimeWindow) error {
	for i, window := range windows {
		if err := validateTimeWindow(window); err != nil {
			return fmt.Errorf("timeWindows[%d]: %w", i, err)
		}
	}
	return nil
}

func validateTimeWindow(window TimeWindow) error {
	if window.Type != TimeWindowRecurring && window.Type != TimeWindowAbsolute {
		return fmt.Errorf("type must be Recurring or Absolute")
	}

	switch window.Type {
	case TimeWindowRecurring:
		if window.Start == "" || window.End == "" {
			return fmt.Errorf("start and end are required for recurring windows")
		}
		if !timeWindowClockPattern.MatchString(window.Start) {
			return fmt.Errorf("start must be HH:MM for recurring windows")
		}
		if !timeWindowClockPattern.MatchString(window.End) {
			return fmt.Errorf("end must be HH:MM for recurring windows")
		}
		if window.Start == window.End {
			return fmt.Errorf("start and end cannot be the same for recurring windows")
		}
		if window.Timezone != "" {
			if _, err := time.LoadLocation(window.Timezone); err != nil {
				return fmt.Errorf("invalid timezone %q", window.Timezone)
			}
		}
		for _, day := range window.DaysOfWeek {
			if _, ok := normalizeWeekday(day); !ok {
				return fmt.Errorf("invalid dayOfWeek %q", day)
			}
		}
	case TimeWindowAbsolute:
		if window.Start == "" || window.End == "" {
			return fmt.Errorf("start and end are required for absolute windows")
		}
		startTime, err := time.Parse(time.RFC3339, window.Start)
		if err != nil {
			return fmt.Errorf("start must be RFC3339 for absolute windows")
		}
		endTime, err := time.Parse(time.RFC3339, window.End)
		if err != nil {
			return fmt.Errorf("end must be RFC3339 for absolute windows")
		}
		if !endTime.After(startTime) {
			return fmt.Errorf("end must be after start for absolute windows")
		}
		if window.Timezone != "" {
			return fmt.Errorf("timezone is not supported for absolute windows")
		}
		if len(window.DaysOfWeek) > 0 {
			return fmt.Errorf("daysOfWeek is not supported for absolute windows")
		}
	}

	return nil
}

func normalizeWeekday(day string) (time.Weekday, bool) {
	switch strings.ToLower(day) {
	case "mon":
		return time.Monday, true
	case "tue":
		return time.Tuesday, true
	case "wed":
		return time.Wednesday, true
	case "thu":
		return time.Thursday, true
	case "fri":
		return time.Friday, true
	case "sat":
		return time.Saturday, true
	case "sun":
		return time.Sunday, true
	default:
		return time.Sunday, false
	}
}
