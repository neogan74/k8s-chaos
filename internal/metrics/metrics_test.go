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
	"sort"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

// metricCase describes the externally visible contract of a metric: the name
// and label names that dashboards and alerts depend on.
type metricCase struct {
	name   string
	labels []string
	// record creates one sample so the metric shows up when gathering.
	record func(labels ...string)
}

func counterVecRecorder(v *prometheus.CounterVec) func(...string) {
	return func(l ...string) { v.WithLabelValues(l...).Inc() }
}

func gaugeVecRecorder(v *prometheus.GaugeVec) func(...string) {
	return func(l ...string) { v.WithLabelValues(l...).Set(1) }
}

func metricCases() []metricCase {
	return []metricCase{
		{"chaosexperiment_executions_total", []string{"action", "namespace", "status"}, counterVecRecorder(ExperimentsTotal)},
		{"chaosexperiment_duration_seconds", []string{"action", "namespace"},
			func(l ...string) { ExperimentDuration.WithLabelValues(l...).Observe(1.5) }},
		{"chaosexperiment_resources_affected", []string{"action", "namespace", "experiment"}, gaugeVecRecorder(ResourcesAffected)},
		{"chaosexperiment_errors_total", []string{"action", "namespace", "error_type"}, counterVecRecorder(ExperimentErrors)},
		{"chaosexperiment_active", []string{"action"}, gaugeVecRecorder(ActiveExperiments)},
		{"chaosexperiment_history_records_total", []string{"action", "status"}, counterVecRecorder(HistoryRecordsTotal)},
		{"chaosexperiment_history_cleanup_total", []string{"reason"}, counterVecRecorder(HistoryCleanupTotal)},
		{"chaosexperiment_history_records_count", []string{"experiment", "namespace"}, gaugeVecRecorder(HistoryRecordsCount)},
		{"chaosexperiment_safety_dryrun_total", []string{"action", "namespace"}, counterVecRecorder(SafetyDryRunExecutions)},
		{"chaosexperiment_safety_production_blocks_total", []string{"action", "namespace"}, counterVecRecorder(SafetyProductionBlocks)},
		{"chaosexperiment_safety_percentage_violations_total", []string{"action", "namespace"}, counterVecRecorder(SafetyPercentageViolations)},
		{"chaosexperiment_safety_excluded_resources_total", []string{"action", "namespace", "resource_type"}, counterVecRecorder(SafetyExcludedResources)},
	}
}

func gatherByName(t *testing.T) map[string]*dto.MetricFamily {
	t.Helper()
	families, err := ctrlmetrics.Registry.Gather()
	if err != nil {
		t.Fatalf("gathering metrics: %v", err)
	}
	byName := make(map[string]*dto.MetricFamily, len(families))
	for _, f := range families {
		byName[f.GetName()] = f
	}
	return byName
}

// TestMetricsRegisteredWithExpectedNamesAndLabels checks that every metric is
// registered in controller-runtime's registry under its documented name and
// exposes exactly the documented label names.
func TestMetricsRegisteredWithExpectedNamesAndLabels(t *testing.T) {
	cases := metricCases()
	for _, tc := range cases {
		values := make([]string, len(tc.labels))
		for i := range values {
			values[i] = "test-" + tc.labels[i]
		}
		tc.record(values...)
	}

	gathered := gatherByName(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			family, ok := gathered[tc.name]
			if !ok {
				t.Fatalf("metric %q is not registered in the controller-runtime registry", tc.name)
			}
			if family.GetHelp() == "" {
				t.Errorf("metric %q has no help text", tc.name)
			}
			if len(family.GetMetric()) == 0 {
				t.Fatalf("metric %q has no samples", tc.name)
			}
			var got []string
			for _, lp := range family.GetMetric()[0].GetLabel() {
				got = append(got, lp.GetName())
			}
			want := append([]string(nil), tc.labels...)
			sort.Strings(got)
			sort.Strings(want)
			if len(got) != len(want) {
				t.Fatalf("labels = %v, want %v", got, want)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("labels = %v, want %v", got, want)
				}
			}
		})
	}
}

func TestRegisteredMetricCountMatchesDeclared(t *testing.T) {
	// Guards against adding a metric variable without registering it (or
	// without adding it to metricCases): every chaosexperiment_* family in the
	// registry must be covered by the table above.
	known := make(map[string]bool)
	for _, tc := range metricCases() {
		known[tc.name] = true
	}
	for _, tc := range metricCases() {
		values := make([]string, len(tc.labels))
		for i := range values {
			values[i] = "x"
		}
		tc.record(values...)
	}
	for name := range gatherByName(t) {
		if len(name) > len("chaosexperiment_") && name[:len("chaosexperiment_")] == "chaosexperiment_" && !known[name] {
			t.Errorf("metric %q is registered but missing from the test table", name)
		}
	}
}

func TestCounterIncrementsPerLabelSet(t *testing.T) {
	ExperimentsTotal.Reset()

	ExperimentsTotal.WithLabelValues("pod-kill", "default", "success").Inc()
	ExperimentsTotal.WithLabelValues("pod-kill", "default", "success").Inc()
	ExperimentsTotal.WithLabelValues("pod-kill", "default", "failure").Inc()

	if got := testutil.ToFloat64(ExperimentsTotal.WithLabelValues("pod-kill", "default", "success")); got != 2 {
		t.Errorf("success counter = %v, want 2", got)
	}
	if got := testutil.ToFloat64(ExperimentsTotal.WithLabelValues("pod-kill", "default", "failure")); got != 1 {
		t.Errorf("failure counter = %v, want 1", got)
	}
	if got := testutil.CollectAndCount(ExperimentsTotal); got != 2 {
		t.Errorf("series count = %d, want 2", got)
	}
}

func TestActiveExperimentsGaugeIncDec(t *testing.T) {
	ActiveExperiments.Reset()

	g := ActiveExperiments.WithLabelValues("node-drain")
	g.Inc()
	g.Inc()
	g.Dec()

	if got := testutil.ToFloat64(g); got != 1 {
		t.Errorf("active gauge = %v, want 1", got)
	}
}

func TestExperimentDurationHistogramObserves(t *testing.T) {
	ExperimentDuration.Reset()

	ExperimentDuration.WithLabelValues("pod-delay", "default").Observe(0.2)
	ExperimentDuration.WithLabelValues("pod-delay", "default").Observe(3)

	if got := testutil.CollectAndCount(ExperimentDuration); got != 1 {
		t.Fatalf("series count = %d, want 1", got)
	}
	families := gatherByName(t)
	h := families["chaosexperiment_duration_seconds"].GetMetric()[0].GetHistogram()
	if h.GetSampleCount() != 2 {
		t.Errorf("sample count = %d, want 2", h.GetSampleCount())
	}
	if h.GetSampleSum() != 3.2 {
		t.Errorf("sample sum = %v, want 3.2", h.GetSampleSum())
	}
}

func TestWrongLabelCountPanics(t *testing.T) {
	// Call sites pass label values positionally; a mismatch must be loud.
	defer func() {
		if recover() == nil {
			t.Error("expected panic when label count does not match")
		}
	}()
	ExperimentsTotal.WithLabelValues("only-one")
}
