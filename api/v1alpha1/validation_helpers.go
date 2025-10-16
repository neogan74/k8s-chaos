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
)

// durationPattern matches the pattern used in the Duration field validation
// Pattern: ^([0-9]+(s|m|h))+$
var durationPattern = regexp.MustCompile(`^([0-9]+(s|m|h))+$`)

// ValidationError represents a validation error for a specific field
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// ValidActions is the list of supported chaos actions
var ValidActions = []string{"pod-kill", "pod-delay", "node-drain"}

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
