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

// IsWithinTimeWindows checks if the current time is within any of the configured time windows.
// Returns true if windows is empty (no restrictions) or if current time matches any window.
func IsWithinTimeWindows(windows []TimeWindow, now time.Time) bool {
	// No windows means no restrictions
	if len(windows) == 0 {
		return true
	}

	// Check if we're within any window
	for _, window := range windows {
		if isWithinTimeWindow(window, now) {
			return true
		}
	}

	return false
}

// isWithinTimeWindow checks if the given time is within a single time window.
func isWithinTimeWindow(window TimeWindow, now time.Time) bool {
	switch window.Type {
	case TimeWindowRecurring:
		return isWithinRecurringWindow(window, now)
	case TimeWindowAbsolute:
		return isWithinAbsoluteWindow(window, now)
	default:
		return false
	}
}

// isWithinRecurringWindow checks if now falls within a recurring time window.
func isWithinRecurringWindow(window TimeWindow, now time.Time) bool {
	// Load timezone (default to UTC)
	loc := time.UTC
	if window.Timezone != "" {
		var err error
		loc, err = time.LoadLocation(window.Timezone)
		if err != nil {
			// Invalid timezone, skip this window
			return false
		}
	}

	// Convert now to the window's timezone
	nowInZone := now.In(loc)

	// Check day of week if specified
	if len(window.DaysOfWeek) > 0 {
		matched := false
		for _, day := range window.DaysOfWeek {
			if weekday, ok := normalizeWeekday(day); ok && weekday == nowInZone.Weekday() {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Parse start and end times
	startParts := strings.Split(window.Start, ":")
	endParts := strings.Split(window.End, ":")
	if len(startParts) != 2 || len(endParts) != 2 {
		return false
	}

	// Create time.Time values for start and end in the same day as now
	startHour, _ := parseTimeComponents(startParts[0], startParts[1])
	endHour, _ := parseTimeComponents(endParts[0], endParts[1])

	start := time.Date(nowInZone.Year(), nowInZone.Month(), nowInZone.Day(), startHour, parseMinute(startParts[1]), 0, 0, loc)
	end := time.Date(nowInZone.Year(), nowInZone.Month(), nowInZone.Day(), endHour, parseMinute(endParts[1]), 0, 0, loc)

	// Handle wrap-around (e.g., 22:00-02:00 spans midnight)
	if end.Before(start) || end.Equal(start) {
		// Window spans midnight: check if we're after start OR before end (same day end time)
		return (nowInZone.After(start) || nowInZone.Equal(start)) || nowInZone.Before(end)
	}

	// Normal window: check if we're between start and end
	return (nowInZone.After(start) || nowInZone.Equal(start)) && nowInZone.Before(end)
}

// isWithinAbsoluteWindow checks if now falls within an absolute time window.
func isWithinAbsoluteWindow(window TimeWindow, now time.Time) bool {
	start, err := time.Parse(time.RFC3339, window.Start)
	if err != nil {
		return false
	}
	end, err := time.Parse(time.RFC3339, window.End)
	if err != nil {
		return false
	}

	return (now.After(start) || now.Equal(start)) && now.Before(end)
}

// NextTimeWindowBoundary calculates when the next time window opens or closes.
// Returns the next boundary time and whether a window will be open at that time.
func NextTimeWindowBoundary(windows []TimeWindow, now time.Time) (nextBoundary time.Time, willBeOpen bool) {
	// No windows means always open
	if len(windows) == 0 {
		return time.Time{}, true
	}

	var earliestBoundary time.Time
	var boundaryIsOpening bool

	for _, window := range windows {
		boundary, isOpening := nextBoundaryForWindow(window, now)
		if !boundary.IsZero() && (earliestBoundary.IsZero() || boundary.Before(earliestBoundary)) {
			earliestBoundary = boundary
			boundaryIsOpening = isOpening
		}
	}

	return earliestBoundary, boundaryIsOpening
}

// nextBoundaryForWindow calculates the next boundary (opening or closing) for a single window.
func nextBoundaryForWindow(window TimeWindow, now time.Time) (boundary time.Time, isOpening bool) {
	switch window.Type {
	case TimeWindowRecurring:
		return nextRecurringBoundary(window, now)
	case TimeWindowAbsolute:
		return nextAbsoluteBoundary(window, now)
	default:
		return time.Time{}, false
	}
}

// nextRecurringBoundary finds the next start or end time for a recurring window.
func nextRecurringBoundary(window TimeWindow, now time.Time) (boundary time.Time, isOpening bool) {
	loc := time.UTC
	if window.Timezone != "" {
		var err error
		loc, err = time.LoadLocation(window.Timezone)
		if err != nil {
			return time.Time{}, false
		}
	}

	nowInZone := now.In(loc)
	startParts := strings.Split(window.Start, ":")
	endParts := strings.Split(window.End, ":")
	if len(startParts) != 2 || len(endParts) != 2 {
		return time.Time{}, false
	}

	startHour, _ := parseTimeComponents(startParts[0], startParts[1])
	endHour, _ := parseTimeComponents(endParts[0], endParts[1])

	// Check boundaries for the next 7 days
	for daysAhead := 0; daysAhead < 8; daysAhead++ {
		checkDate := nowInZone.AddDate(0, 0, daysAhead)

		// Skip if day of week doesn't match
		if len(window.DaysOfWeek) > 0 {
			matched := false
			for _, day := range window.DaysOfWeek {
				if weekday, ok := normalizeWeekday(day); ok && weekday == checkDate.Weekday() {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		start := time.Date(checkDate.Year(), checkDate.Month(), checkDate.Day(), startHour, parseMinute(startParts[1]), 0, 0, loc)
		end := time.Date(checkDate.Year(), checkDate.Month(), checkDate.Day(), endHour, parseMinute(endParts[1]), 0, 0, loc)

		// Handle wrap-around
		if end.Before(start) || end.Equal(start) {
			end = end.Add(24 * time.Hour)
		}

		// Check if start is in the future
		if start.After(nowInZone) {
			return start, true
		}

		// Check if end is in the future
		if end.After(nowInZone) {
			return end, false
		}
	}

	return time.Time{}, false
}

// nextAbsoluteBoundary finds the next start or end time for an absolute window.
func nextAbsoluteBoundary(window TimeWindow, now time.Time) (boundary time.Time, isOpening bool) {
	start, err := time.Parse(time.RFC3339, window.Start)
	if err != nil {
		return time.Time{}, false
	}
	end, err := time.Parse(time.RFC3339, window.End)
	if err != nil {
		return time.Time{}, false
	}

	// If before start, next boundary is start (opening)
	if now.Before(start) {
		return start, true
	}

	// If between start and end, next boundary is end (closing)
	if now.Before(end) {
		return end, false
	}

	// After window has closed, no future boundary
	return time.Time{}, false
}

// Helper functions
func parseTimeComponents(hourStr, minStr string) (hour int, min int) {
	fmt.Sscanf(hourStr, "%d", &hour)
	fmt.Sscanf(minStr, "%d", &min)
	return hour, min
}

func parseMinute(minStr string) int {
	var min int
	fmt.Sscanf(minStr, "%d", &min)
	return min
}
