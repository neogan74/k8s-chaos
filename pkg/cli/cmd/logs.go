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
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	chaosv1alpha1 "github.com/neogan74/k8s-chaos/api/v1alpha1"
)

// Labels set by the controller on every ChaosExperimentHistory record.
const (
	historyLabelExperiment = "chaos.gushchin.dev/experiment"
	historyLabelAction     = "chaos.gushchin.dev/action"
	historyLabelStatus     = "chaos.gushchin.dev/status"

	statusSuccess = "success"
	statusFailure = "failure"

	formatTable = "table"
	formatWide  = "wide"
	formatJSON  = "json"
	formatYAML  = "yaml"
	formatCSV   = "csv"

	defaultLogsLimit = 20
	timeLayoutShort  = "2006-01-02 15:04:05"
)

var (
	logsStatus     string
	logsAction     string
	logsSince      string
	logsUntil      string
	logsLimit      int
	logsOutput     string
	logsOutputFile string
)

var logsCmd = &cobra.Command{
	Use:   "logs [EXPERIMENT_NAME]",
	Short: "Show experiment execution history",
	Long: `Show past executions of chaos experiments, newest first.

History is read from ChaosExperimentHistory records, which the controller
creates after each execution. Use -n to select the namespace the history
records live in (default: all namespaces).

Time filters (--since, --until) accept a relative duration such as 90m, 24h,
7d or 2w, an absolute date such as 2026-10-01, or an RFC3339 timestamp.

Output formats (-o): table (default), wide, json, yaml, csv.

Examples:
  # The 20 most recent executions
  k8s-chaos logs

  # History of one experiment
  k8s-chaos logs nginx-chaos-demo -n chaos-testing

  # Failures in the last 7 days
  k8s-chaos logs --status failure --since 7d

  # Everything in October, exported to CSV
  k8s-chaos logs --since 2026-10-01 --until 2026-11-01 --limit 0 -o csv --file history.csv`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLogs,
}

func init() {
	flags := logsCmd.Flags()
	flags.StringVar(&logsStatus, "status", "", "only show executions with this status: success, failure")
	flags.StringVar(&logsAction, "action", "", "only show executions of this chaos action (e.g. pod-kill)")
	flags.StringVar(&logsSince, "since", "",
		"only show executions started after this time (e.g. 24h, 7d, 2026-10-01)")
	flags.StringVar(&logsUntil, "until", "", "only show executions started before this time")
	flags.IntVar(&logsLimit, "limit", defaultLogsLimit, "maximum number of executions to show (0 for all)")
	flags.StringVarP(&logsOutput, "output", "o", formatTable, "output format: table, wide, json, yaml, csv")
	flags.StringVar(&logsOutputFile, "file", "", "write the output to this file instead of stdout")
	rootCmd.AddCommand(logsCmd)
}

// historyQuery describes which history records to show.
type historyQuery struct {
	Experiment string
	Action     string
	Status     string
	Since      time.Time // zero means no lower bound
	Until      time.Time // zero means no upper bound
	Limit      int       // 0 means no limit
}

func runLogs(cmd *cobra.Command, args []string) error {
	// Execute() prints the returned error once; keep cobra from printing usage or it again.
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	query := historyQuery{Action: logsAction, Limit: logsLimit}
	if len(args) == 1 {
		query.Experiment = args[0]
	}

	if err := validateOutputFormat(logsOutput); err != nil {
		return err
	}
	if logsLimit < 0 {
		return fmt.Errorf("--limit must not be negative, got %d", logsLimit)
	}

	var err error
	if query.Status, err = normalizeStatus(logsStatus); err != nil {
		return err
	}
	now := time.Now()
	if query.Since, err = parseTimeSpec(logsSince, now); err != nil {
		return fmt.Errorf("invalid --since: %w", err)
	}
	if query.Until, err = parseTimeSpec(logsUntil, now); err != nil {
		return fmt.Errorf("invalid --until: %w", err)
	}
	if !query.Since.IsZero() && !query.Until.IsZero() && query.Until.Before(query.Since) {
		return fmt.Errorf("--until (%s) is before --since (%s)",
			query.Until.Format(time.RFC3339), query.Since.Format(time.RFC3339))
	}

	k8sClient, err := getKubeClient()
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	items, err := fetchHistories(context.Background(), k8sClient, namespace, query)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if logsOutputFile != "" {
		f, err := os.Create(logsOutputFile)
		if err != nil {
			return fmt.Errorf("cannot create %q: %w", logsOutputFile, err)
		}
		defer func() { _ = f.Close() }()
		out = f
	}

	if len(items) == 0 && (logsOutput == formatTable || logsOutput == formatWide) {
		_, _ = fmt.Fprintln(out, "No execution history found")
		return nil
	}
	if err := writeHistories(out, items, logsOutput); err != nil {
		return err
	}
	if logsOutputFile != "" {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Wrote %d record(s) to %s\n", len(items), logsOutputFile)
	}
	return nil
}

func validateOutputFormat(format string) error {
	switch format {
	case formatTable, formatWide, formatJSON, formatYAML, formatCSV:
		return nil
	}
	return fmt.Errorf("unsupported output format %q, must be one of: table, wide, json, yaml, csv", format)
}

// normalizeStatus maps user input to the status values stored on history records.
func normalizeStatus(s string) (string, error) {
	switch strings.ToLower(s) {
	case "":
		return "", nil
	case statusSuccess, "succeeded", "ok":
		return statusSuccess, nil
	case statusFailure, "failed", "error":
		return statusFailure, nil
	}
	return "", fmt.Errorf("invalid --status %q, must be one of: success, failure", s)
}

// parseTimeSpec parses a relative duration (30m, 24h, 7d, 2w), a date
// (2026-10-01), a date-time or an RFC3339 timestamp. Relative values are
// interpreted as "that long before now". An empty string returns the zero time.
func parseTimeSpec(spec string, now time.Time) (time.Time, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return time.Time{}, nil
	}

	if d, ok := parseRelativeDuration(spec); ok {
		return now.Add(-d), nil
	}

	layouts := []string{time.RFC3339, "2006-01-02T15:04:05", timeLayoutShort, "2006-01-02 15:04", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, spec, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf(
		"cannot parse %q: use a duration like 24h or 7d, a date like 2026-10-01, or RFC3339", spec)
}

// parseRelativeDuration extends time.ParseDuration with d (day) and w (week) suffixes.
func parseRelativeDuration(s string) (time.Duration, bool) {
	if n := len(s); n > 1 && (s[n-1] == 'd' || s[n-1] == 'w') {
		count, err := strconv.Atoi(s[:n-1])
		if err != nil || count < 0 {
			return 0, false
		}
		unit := 24 * time.Hour
		if s[n-1] == 'w' {
			unit = 7 * 24 * time.Hour
		}
		return time.Duration(count) * unit, true
	}
	d, err := time.ParseDuration(s)
	if err != nil || d < 0 {
		return 0, false
	}
	return d, true
}

// fetchHistories lists history records matching the query, newest first.
// Experiment, action and status are filtered by the API server through labels;
// time bounds and the limit are applied here.
func fetchHistories(
	ctx context.Context, c client.Client, ns string, q historyQuery,
) ([]chaosv1alpha1.ChaosExperimentHistory, error) {
	labels := client.MatchingLabels{}
	if q.Experiment != "" {
		labels[historyLabelExperiment] = q.Experiment
	}
	if q.Action != "" {
		labels[historyLabelAction] = q.Action
	}
	if q.Status != "" {
		labels[historyLabelStatus] = q.Status
	}

	opts := []client.ListOption{labels}
	if ns != "" {
		opts = append(opts, client.InNamespace(ns))
	}

	list := &chaosv1alpha1.ChaosExperimentHistoryList{}
	if err := c.List(ctx, list, opts...); err != nil {
		return nil, fmt.Errorf("failed to list execution history: %w", err)
	}
	return selectHistories(list.Items, q), nil
}

// selectHistories applies the time bounds, sorts newest first and applies the limit.
func selectHistories(
	items []chaosv1alpha1.ChaosExperimentHistory, q historyQuery,
) []chaosv1alpha1.ChaosExperimentHistory {
	selected := make([]chaosv1alpha1.ChaosExperimentHistory, 0, len(items))
	for _, h := range items {
		started := h.Spec.Execution.StartTime.Time
		if !q.Since.IsZero() && started.Before(q.Since) {
			continue
		}
		if !q.Until.IsZero() && started.After(q.Until) {
			continue
		}
		selected = append(selected, h)
	}

	sort.SliceStable(selected, func(i, j int) bool {
		return selected[i].Spec.Execution.StartTime.After(selected[j].Spec.Execution.StartTime.Time)
	})

	if q.Limit > 0 && len(selected) > q.Limit {
		selected = selected[:q.Limit]
	}
	return selected
}

func writeHistories(w io.Writer, items []chaosv1alpha1.ChaosExperimentHistory, format string) error {
	switch format {
	case formatJSON:
		data, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
		_, err = fmt.Fprintln(w, string(data))
		return err
	case formatYAML:
		data, err := yaml.Marshal(items)
		if err != nil {
			return fmt.Errorf("failed to encode YAML: %w", err)
		}
		_, err = w.Write(data)
		return err
	case formatCSV:
		return writeHistoriesCSV(w, items)
	default:
		return writeHistoriesTable(w, items, format == formatWide)
	}
}

func writeHistoriesTable(w io.Writer, items []chaosv1alpha1.ChaosExperimentHistory, wide bool) error {
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	if wide {
		_, _ = fmt.Fprintln(tw, "STARTED\tEXPERIMENT\tNAMESPACE\tACTION\tSTATUS\tDURATION\tAFFECTED"+
			"\tRETRIES\tDRY-RUN\tINITIATED-BY\tMESSAGE")
	} else {
		_, _ = fmt.Fprintln(tw, "STARTED\tEXPERIMENT\tNAMESPACE\tACTION\tSTATUS\tDURATION\tAFFECTED")
	}

	for _, h := range items {
		e := h.Spec.Execution
		base := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%d",
			e.StartTime.Local().Format(timeLayoutShort),
			h.Spec.ExperimentRef.Name,
			h.Spec.ExperimentRef.Namespace,
			h.Spec.ExperimentSpec.Action,
			e.Status,
			displayOrDash(e.Duration),
			len(h.Spec.AffectedResources),
		)
		if !wide {
			_, _ = fmt.Fprintln(tw, base)
			continue
		}
		_, _ = fmt.Fprintf(tw, "%s\t%d\t%t\t%s\t%s\n",
			base,
			h.Spec.Audit.RetryCount,
			h.Spec.Audit.DryRun,
			displayOrDash(h.Spec.Audit.InitiatedBy),
			displayOrDash(singleLine(historyMessage(h))),
		)
	}
	return tw.Flush()
}

func writeHistoriesCSV(w io.Writer, items []chaosv1alpha1.ChaosExperimentHistory) error {
	cw := csv.NewWriter(w)
	header := []string{
		"started", "ended", "history", "experiment", "namespace", "action", "target_namespace",
		"status", "duration", "affected", "retries", "dry_run", "scheduled", "initiated_by", "message",
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, h := range items {
		e := h.Spec.Execution
		ended := ""
		if e.EndTime != nil {
			ended = e.EndTime.UTC().Format(time.RFC3339)
		}
		record := []string{
			e.StartTime.UTC().Format(time.RFC3339),
			ended,
			h.Name,
			h.Spec.ExperimentRef.Name,
			h.Spec.ExperimentRef.Namespace,
			h.Spec.ExperimentSpec.Action,
			h.Spec.ExperimentSpec.Namespace,
			e.Status,
			e.Duration,
			strconv.Itoa(len(h.Spec.AffectedResources)),
			strconv.Itoa(h.Spec.Audit.RetryCount),
			strconv.FormatBool(h.Spec.Audit.DryRun),
			strconv.FormatBool(h.Spec.Audit.ScheduledExecution),
			h.Spec.Audit.InitiatedBy,
			historyMessage(h),
		}
		if err := cw.Write(record); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// historyMessage returns the most useful one-line explanation for a record:
// the error message for failures, otherwise the execution message.
func historyMessage(h chaosv1alpha1.ChaosExperimentHistory) string {
	if h.Spec.Error != nil {
		if h.Spec.Error.Message != "" {
			return h.Spec.Error.Message
		}
		if h.Spec.Error.LastError != "" {
			return h.Spec.Error.LastError
		}
	}
	return h.Spec.Execution.Message
}

// singleLine collapses whitespace and newlines so a message cannot break table rows.
func singleLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func displayOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
