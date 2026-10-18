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
	"errors"
	"fmt"
	"strings"
)

// SupportedActions lists every value accepted by spec.action. It must stay in
// sync with the kubebuilder Enum marker on ChaosExperimentSpec.Action (a unit
// test enforces this).
var SupportedActions = []string{
	"pod-kill", "pod-delay", "node-drain", "node-taint", "node-cpu-stress",
	"node-disk-fill", "pod-cpu-stress", "pod-memory-stress", "pod-failure",
	"pod-network-loss", "pod-network-corruption", "pod-disk-fill", "pod-restart",
	"network-partition",
}

type intRange struct {
	field    string
	value    int
	min, max int
}

// ValidateStatic runs every check that needs no cluster access: the CRD
// schema constraints (enums, ranges, patterns) followed by the webhook's
// cross-field and action-specific rules. It does not check that the namespace
// exists, that the selector matches pods, or the safety/production rules.
//
// All schema problems are reported together (joined). Cross-field rules are
// only evaluated once the schema is valid, because they assume sane values.
func (r *ChaosExperiment) ValidateStatic() error {
	if errs := r.validateSchema(); len(errs) > 0 {
		return errors.Join(errs...)
	}
	return (&ChaosExperimentWebhook{}).validateCrossFieldConstraints(r.Name, &r.Spec)
}

func (r *ChaosExperiment) validateSchema() []error {
	s := &r.Spec
	var errs []error
	addf := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }

	if s.Action == "" {
		addf("spec.action: required")
	} else if !containsString(SupportedActions, s.Action) {
		addf("spec.action: unsupported value %q, must be one of: %s", s.Action, strings.Join(SupportedActions, ", "))
	}
	if s.Namespace == "" {
		addf("spec.namespace: required")
	}
	if len(s.Selector) == 0 {
		addf("spec.selector: required, must have at least one label")
	}

	// Zero means "unset" for omitempty ints, so only non-zero values are range-checked
	// except where the CRD allows 0 explicitly.
	for _, c := range []intRange{
		{"count", s.Count, 1, 100},
		{"maxRetries", s.MaxRetries, 0, 10},
		{"cpuLoad", s.CPULoad, 1, 100},
		{"cpuWorkers", s.CPUWorkers, 1, 32},
		{"memoryWorkers", s.MemoryWorkers, 1, 8},
		{"lossPercentage", s.LossPercentage, 1, 40},
		{"lossCorrelation", s.LossCorrelation, 0, 100},
		{"corruptionPercentage", s.CorruptionPercentage, 1, 100},
		{"corruptionCorrelation", s.CorruptionCorrelation, 0, 100},
		{"fillPercentage", s.FillPercentage, 50, 95},
		{"maxPercentage", s.MaxPercentage, 1, 100},
	} {
		if c.value != 0 && (c.value < c.min || c.value > c.max) {
			addf("spec.%s: %d is out of range, must be between %d and %d", c.field, c.value, c.min, c.max)
		}
		if c.value < 0 {
			addf("spec.%s: %d must not be negative", c.field, c.value)
		}
	}

	for field, value := range map[string]string{
		"duration":           s.Duration,
		"experimentDuration": s.ExperimentDuration,
		"retryDelay":         s.RetryDelay,
		"restartInterval":    s.RestartInterval,
	} {
		if value != "" && !durationPattern.MatchString(value) {
			addf("spec.%s: %q is invalid, expected a duration such as 30s, 5m or 1h30m", field, value)
		}
	}
	if s.MemorySize != "" && !memorySizePattern.MatchString(s.MemorySize) {
		addf("spec.memorySize: %q is invalid, expected a number followed by M or G (e.g. 256M, 1G)", s.MemorySize)
	}

	checkEnum := func(field, value string, allowed ...string) {
		if value != "" && !containsString(allowed, value) {
			addf("spec.%s: unsupported value %q, must be one of: %s", field, value, strings.Join(allowed, ", "))
		}
	}
	checkEnum("retryBackoff", s.RetryBackoff, "exponential", "fixed")
	checkEnum("direction", s.Direction, "both", "ingress", "egress")
	checkEnum("taintEffect", s.TaintEffect, "NoSchedule", "PreferNoSchedule", "NoExecute")
	for i, p := range s.TargetProtocols {
		checkEnum(fmt.Sprintf("targetProtocols[%d]", i), p, "tcp", "udp", "icmp")
	}

	return errs
}

func containsString(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
