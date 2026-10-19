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

package cmd

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	chaosv1alpha1 "github.com/neogan74/k8s-chaos/api/v1alpha1"
)

const (
	testHistNS   = "chaos-testing"
	testActKill  = "pod-kill"
	testActDelay = "pod-delay"
	idNew        = "a-new"
	idOld        = "a-old"
	idMid        = "b-mid"
	expAlpha     = "alpha"
)

var baseTime = time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)

// newHistory builds a history record that started `ago` before baseTime.
func newHistory(name, experiment, action, status string, ago time.Duration) *chaosv1alpha1.ChaosExperimentHistory {
	started := metav1.NewTime(baseTime.Add(-ago))
	ended := metav1.NewTime(started.Add(30 * time.Second))
	return &chaosv1alpha1.ChaosExperimentHistory{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testHistNS,
			Labels: map[string]string{
				historyLabelExperiment: experiment,
				historyLabelAction:     action,
				historyLabelStatus:     status,
			},
		},
		Spec: chaosv1alpha1.ChaosExperimentHistorySpec{
			ExperimentRef:  chaosv1alpha1.ObjectReference{Name: experiment, Namespace: testHistNS},
			ExperimentSpec: chaosv1alpha1.ChaosExperimentSpec{Action: action, Namespace: "demo"},
			Execution: chaosv1alpha1.ExecutionDetails{
				StartTime: started,
				EndTime:   &ended,
				Duration:  "30s",
				Status:    status,
				Message:   "done",
			},
			AffectedResources: []chaosv1alpha1.ResourceReference{{Kind: "Pod", Name: "p1", Action: "deleted"}},
			Audit:             chaosv1alpha1.AuditMetadata{InitiatedBy: "alice", RetryCount: 1},
		},
	}
}

func fakeHistoryClient(t *testing.T, objs ...*chaosv1alpha1.ChaosExperimentHistory) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := chaosv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	builder := fake.NewClientBuilder().WithScheme(scheme)
	for _, o := range objs {
		builder = builder.WithObjects(o)
	}
	return builder.Build()
}

func names(items []chaosv1alpha1.ChaosExperimentHistory) []string {
	out := make([]string, len(items))
	for i, h := range items {
		out[i] = h.Name
	}
	return out
}

func sameStrings(a, b []string) bool {
	return strings.Join(a, ",") == strings.Join(b, ",")
}

func TestParseTimeSpec(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)

	relative := map[string]time.Duration{
		"90m": 90 * time.Minute,
		"24h": 24 * time.Hour,
		"7d":  7 * 24 * time.Hour,
		"2w":  14 * 24 * time.Hour,
	}
	for spec, d := range relative {
		got, err := parseTimeSpec(spec, now)
		if err != nil {
			t.Errorf("%s: unexpected error %v", spec, err)
			continue
		}
		if !got.Equal(now.Add(-d)) {
			t.Errorf("%s: got %v, want %v", spec, got, now.Add(-d))
		}
	}

	if got, err := parseTimeSpec("", now); err != nil || !got.IsZero() {
		t.Errorf("empty spec should be the zero time, got %v, %v", got, err)
	}

	got, err := parseTimeSpec("2026-10-01T08:30:00Z", now)
	if err != nil || !got.Equal(time.Date(2026, 10, 1, 8, 30, 0, 0, time.UTC)) {
		t.Errorf("RFC3339: got %v, %v", got, err)
	}
	got, err = parseTimeSpec("2026-10-01", now)
	if err != nil || got.Year() != 2026 || got.Month() != 10 || got.Day() != 1 {
		t.Errorf("date only: got %v, %v", got, err)
	}

	for _, bad := range []string{"yesterday", "-5h", "10x", "2026-13-45", "d"} {
		if _, err := parseTimeSpec(bad, now); err == nil {
			t.Errorf("%q: expected an error", bad)
		}
	}
}

func TestNormalizeStatus(t *testing.T) {
	cases := map[string]string{
		"": "", "success": statusSuccess, "SUCCESS": statusSuccess, "ok": statusSuccess,
		"failure": statusFailure, "failed": statusFailure, "Error": statusFailure,
	}
	for in, want := range cases {
		got, err := normalizeStatus(in)
		if err != nil || got != want {
			t.Errorf("normalizeStatus(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := normalizeStatus("pending"); err == nil {
		t.Error("expected an error for an unknown status")
	}
}

func TestFetchHistories_FiltersAndOrder(t *testing.T) {
	c := fakeHistoryClient(t,
		newHistory(idOld, expAlpha, testActKill, statusSuccess, 72*time.Hour),
		newHistory(idNew, expAlpha, testActKill, statusFailure, 1*time.Hour),
		newHistory(idMid, "beta", testActDelay, statusSuccess, 24*time.Hour),
	)
	ctx := context.Background()

	tests := []struct {
		name  string
		query historyQuery
		want  []string
	}{
		{"all, newest first", historyQuery{}, []string{idNew, idMid, idOld}},
		{"by experiment", historyQuery{Experiment: expAlpha}, []string{idNew, idOld}},
		{"by action", historyQuery{Action: testActDelay}, []string{idMid}},
		{"by status", historyQuery{Status: statusFailure}, []string{idNew}},
		{"combined filters", historyQuery{Experiment: expAlpha, Status: statusSuccess}, []string{idOld}},
		{"since", historyQuery{Since: baseTime.Add(-30 * time.Hour)}, []string{idNew, idMid}},
		{"until", historyQuery{Until: baseTime.Add(-30 * time.Hour)}, []string{idOld}},
		{"since and until", historyQuery{Since: baseTime.Add(-48 * time.Hour), Until: baseTime.Add(-12 * time.Hour)},
			[]string{idMid}},
		{"limit keeps the newest", historyQuery{Limit: 2}, []string{idNew, idMid}},
		{"no match", historyQuery{Experiment: "missing"}, []string{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := fetchHistories(ctx, c, "", tc.query)
			if err != nil {
				t.Fatal(err)
			}
			if !sameStrings(names(got), tc.want) {
				t.Fatalf("got %v, want %v", names(got), tc.want)
			}
		})
	}
}

func TestFetchHistories_NamespaceScope(t *testing.T) {
	other := newHistory("other-ns", expAlpha, testActKill, statusSuccess, time.Hour)
	other.Namespace = "elsewhere"
	c := fakeHistoryClient(t, newHistory("here", expAlpha, testActKill, statusSuccess, time.Hour), other)

	got, err := fetchHistories(context.Background(), c, testHistNS, historyQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if !sameStrings(names(got), []string{"here"}) {
		t.Fatalf("namespace filter: got %v", names(got))
	}

	all, _ := fetchHistories(context.Background(), c, "", historyQuery{})
	if len(all) != 2 {
		t.Fatalf("empty namespace should list all namespaces, got %v", names(all))
	}
}

func sampleItems(t *testing.T) []chaosv1alpha1.ChaosExperimentHistory {
	t.Helper()
	failed := newHistory("failed-run", expAlpha, testActKill, statusFailure, time.Hour)
	failed.Spec.Error = &chaosv1alpha1.ErrorDetails{Message: "pod not found,\nretry later"}
	ok := newHistory("ok-run", "beta", testActDelay, statusSuccess, 2*time.Hour)
	got, err := fetchHistories(context.Background(), fakeHistoryClient(t, failed, ok), "", historyQuery{})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestWriteHistories_Table(t *testing.T) {
	var out bytes.Buffer
	if err := writeHistories(&out, sampleItems(t), formatTable); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected header + 2 rows, got:\n%s", out.String())
	}
	for _, want := range []string{"STARTED", "EXPERIMENT", "STATUS", "AFFECTED"} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("header missing %q: %s", want, lines[0])
		}
	}
	if !strings.Contains(lines[1], expAlpha) || !strings.Contains(lines[1], statusFailure) {
		t.Errorf("newest row should be the failed alpha run: %s", lines[1])
	}
	if strings.Contains(out.String(), "MESSAGE") {
		t.Error("the plain table must not include the wide-only MESSAGE column")
	}
}

func TestWriteHistories_WideCollapsesMessage(t *testing.T) {
	var out bytes.Buffer
	if err := writeHistories(&out, sampleItems(t), formatWide); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "MESSAGE") || !strings.Contains(out.String(), "alice") {
		t.Errorf("wide output should include message and initiator:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "pod not found, retry later") {
		t.Errorf("multi-line messages must be collapsed onto one line:\n%s", out.String())
	}
	if got := len(strings.Split(strings.TrimSpace(out.String()), "\n")); got != 3 {
		t.Errorf("expected 3 lines, got %d:\n%s", got, out.String())
	}
}

func TestWriteHistories_JSONAndYAML(t *testing.T) {
	items := sampleItems(t)

	var jsonOut bytes.Buffer
	if err := writeHistories(&jsonOut, items, formatJSON); err != nil {
		t.Fatal(err)
	}
	var decoded []chaosv1alpha1.ChaosExperimentHistory
	if err := json.Unmarshal(jsonOut.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, jsonOut.String())
	}
	if len(decoded) != 2 || decoded[0].Name != "failed-run" {
		t.Errorf("unexpected decoded JSON: %v", names(decoded))
	}

	var yamlOut bytes.Buffer
	if err := writeHistories(&yamlOut, items, formatYAML); err != nil {
		t.Fatal(err)
	}
	var fromYAML []chaosv1alpha1.ChaosExperimentHistory
	if err := yaml.Unmarshal(yamlOut.Bytes(), &fromYAML); err != nil {
		t.Fatalf("output is not valid YAML: %v", err)
	}
	if len(fromYAML) != 2 {
		t.Errorf("expected 2 records from YAML, got %d", len(fromYAML))
	}
}

func TestWriteHistories_CSV(t *testing.T) {
	var out bytes.Buffer
	if err := writeHistories(&out, sampleItems(t), formatCSV); err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(&out).ReadAll()
	if err != nil {
		t.Fatalf("output is not valid CSV: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected header + 2 rows, got %d", len(rows))
	}
	col := map[string]int{}
	for i, name := range rows[0] {
		col[name] = i
	}
	first := rows[1]
	if first[col["experiment"]] != expAlpha || first[col["status"]] != statusFailure || first[col["affected"]] != "1" {
		t.Errorf("unexpected first row: %v", first)
	}
	if first[col["message"]] != "pod not found,\nretry later" {
		t.Errorf("CSV must keep the raw message (quoted), got %q", first[col["message"]])
	}
	if first[col["started"]] != baseTime.Add(-time.Hour).Format(time.RFC3339) {
		t.Errorf("started should be UTC RFC3339, got %q", first[col["started"]])
	}
}

func TestValidateOutputFormat(t *testing.T) {
	for _, ok := range []string{"table", "wide", "json", "yaml", "csv"} {
		if err := validateOutputFormat(ok); err != nil {
			t.Errorf("%s: %v", ok, err)
		}
	}
	if err := validateOutputFormat("xml"); err == nil {
		t.Error("expected an error for xml")
	}
}

func TestHistoryMessagePrefersError(t *testing.T) {
	h := newHistory("x", "e", testActKill, statusFailure, time.Hour)
	if got := historyMessage(*h); got != "done" {
		t.Errorf("without an error the execution message is used, got %q", got)
	}
	h.Spec.Error = &chaosv1alpha1.ErrorDetails{LastError: "last"}
	if got := historyMessage(*h); got != "last" {
		t.Errorf("LastError is the fallback, got %q", got)
	}
	h.Spec.Error.Message = "boom"
	if got := historyMessage(*h); got != "boom" {
		t.Errorf("error message wins, got %q", got)
	}
}
