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
)

func init() {
	// Register custom metrics with controller-runtime's registry
	metrics.Registry.MustRegister(
		ExperimentsTotal,
		ExperimentDuration,
		ResourcesAffected,
		ExperimentErrors,
		ActiveExperiments,
	)
}
