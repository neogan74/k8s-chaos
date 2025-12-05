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

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// ExperimentsTotal counts the total number of chaos experiments executed
	ExperimentsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chaosexperiment_executions_total",
			Help: "Total number of chaos experiments executed",
		},
		[]string{"action", "namespace", "status"},
	)

	// ExperimentDuration tracks the duration of chaos experiment executions
	ExperimentDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "chaosexperiment_duration_seconds",
			Help:    "Duration of chaos experiment execution in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"action", "namespace"},
	)

	// ResourcesAffected tracks the number of resources affected by experiments
	ResourcesAffected = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "chaosexperiment_resources_affected",
			Help: "Number of resources (pods/nodes) affected by chaos experiments",
		},
		[]string{"action", "namespace", "experiment"},
	)

	// ExperimentErrors counts the number of errors during chaos experiments
	ExperimentErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chaosexperiment_errors_total",
			Help: "Total number of errors during chaos experiments",
		},
		[]string{"action", "namespace", "error_type"},
	)

	// ActiveExperiments tracks the number of currently active experiments
	ActiveExperiments = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "chaosexperiment_active",
			Help: "Number of currently active chaos experiments",
		},
		[]string{"action"},
	)

	// HistoryRecordsTotal counts the total number of history records created
	HistoryRecordsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chaosexperiment_history_records_total",
			Help: "Total number of history records created",
		},
		[]string{"action", "status"},
	)

	// HistoryCleanupTotal counts the number of history records cleaned up
	HistoryCleanupTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chaosexperiment_history_cleanup_total",
			Help: "Total number of history records deleted by retention policy",
		},
		[]string{"reason"},
	)

	// HistoryRecordsCount tracks the current number of history records per experiment
	HistoryRecordsCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "chaosexperiment_history_records_count",
			Help: "Current number of history records per experiment",
		},
		[]string{"experiment", "namespace"},
	)

	// SafetyDryRunExecutions counts experiments executed in dry-run mode
	SafetyDryRunExecutions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chaosexperiment_safety_dryrun_total",
			Help: "Total number of experiments executed in dry-run mode",
		},
		[]string{"action", "namespace"},
	)

	// SafetyProductionBlocks counts experiments blocked due to production protection
	SafetyProductionBlocks = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chaosexperiment_safety_production_blocks_total",
			Help: "Total number of experiments blocked due to production namespace protection",
		},
		[]string{"action", "namespace"},
	)

	// SafetyPercentageViolations counts experiments blocked due to maxPercentage violations
	SafetyPercentageViolations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chaosexperiment_safety_percentage_violations_total",
			Help: "Total number of experiments blocked due to maxPercentage limit violations",
		},
		[]string{"action", "namespace"},
	)

	// SafetyExcludedResources tracks resources excluded from experiments via exclusion labels
	SafetyExcludedResources = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chaosexperiment_safety_excluded_resources_total",
			Help: "Total number of resources excluded from experiments via exclusion labels",
		},
		[]string{"action", "namespace", "resource_type"},
	)
)

func init() {
	// Register custom metrics with controller-runtime's registry
	metrics.Registry.MustRegister(
		ExperimentsTotal,
		ExperimentDuration,
		ResourcesAffected,
		ExperimentErrors,
		ActiveExperiments,
		HistoryRecordsTotal,
		HistoryCleanupTotal,
		HistoryRecordsCount,
		SafetyDryRunExecutions,
		SafetyProductionBlocks,
		SafetyPercentageViolations,
		SafetyExcludedResources,
	)
}
