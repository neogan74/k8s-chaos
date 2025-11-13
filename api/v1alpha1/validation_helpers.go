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

	"github.com/robfig/cron/v3"
)

// durationPattern matches the pattern used in the Duration field validation
// Pattern: ^([0-9]+(s|m|h))+$
var durationPattern = regexp.MustCompile(`^([0-9]+(s|m|h))+$`)

// memorySizePattern matches the pattern used in the MemorySize field validation
// Pattern: ^[0-9]+[MG]$
var memorySizePattern = regexp.MustCompile(`^[0-9]+[MG]$`)

// ValidationError represents a validation error for a specific field
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// ValidActions is the list of supported chaos actions
var ValidActions = []string{"pod-kill", "pod-delay", "node-drain", "pod-cpu-stress", "pod-memory-stress"}

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
