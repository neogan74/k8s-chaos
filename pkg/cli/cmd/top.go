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
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"

	chaosv1alpha1 "github.com/neogan74/k8s-chaos/api/v1alpha1"
)

var topCmd = &cobra.Command{
	Use:   "top",
	Short: "Show top chaos experiments by various metrics",
	Long: `Display the top chaos experiments ranked by different metrics
such as retry count, age, or resource count.

Examples:
  # Show top experiments by retry count
  k8s-chaos top

  # Show top 5 experiments
  k8s-chaos top --limit 5

  # Show top experiments in a specific namespace
  k8s-chaos top -n chaos-testing`,
	RunE: runTop,
}

var topLimit int

func init() {
	topCmd.Flags().IntVarP(&topLimit, "limit", "l", 10, "limit the number of experiments to show")
	rootCmd.AddCommand(topCmd)
}

type experimentMetrics struct {
	Name       string
	Namespace  string
	Action     string
	RetryCount int
	Phase      string
	Age        time.Duration
	TargetNS   string
}

func runTop(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	k8sClient, err := getKubeClient()
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	expList := &chaosv1alpha1.ChaosExperimentList{}
	listOpts := []client.ListOption{}
	if namespace != "" {
		listOpts = append(listOpts, client.InNamespace(namespace))
	}

	if err := k8sClient.List(ctx, expList, listOpts...); err != nil {
		return fmt.Errorf("failed to list chaos experiments: %w", err)
	}

	if len(expList.Items) == 0 {
		fmt.Println("No chaos experiments found")
		return nil
	}

	// Collect metrics
	metrics := make([]experimentMetrics, 0, len(expList.Items))
	for _, exp := range expList.Items {
		metrics = append(metrics, experimentMetrics{
			Name:       exp.Name,
			Namespace:  exp.Namespace,
			Action:     exp.Spec.Action,
			RetryCount: exp.Status.RetryCount,
			Phase:      exp.Status.Phase,
			Age:        time.Since(exp.CreationTimestamp.Time),
			TargetNS:   exp.Spec.Namespace,
		})
	}

	// Print top experiments by retry count
	fmt.Println("=== Top Experiments by Retry Count ===")
	printTopByRetries(metrics, topLimit)

	fmt.Println()
	fmt.Println("=== Top Experiments by Age ===")
	printTopByAge(metrics, topLimit)

	fmt.Println()
	fmt.Println("=== Failed Experiments ===")
	printFailed(metrics)

	return nil
}

func printTopByRetries(metrics []experimentMetrics, limit int) {
	// Sort by retry count descending
	sorted := make([]experimentMetrics, len(metrics))
	copy(sorted, metrics)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].RetryCount > sorted[j].RetryCount
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAMESPACE\tNAME\tACTION\tRETRIES\tPHASE\tAGE")

	count := 0
	for _, m := range sorted {
		if count >= limit {
			break
		}
		if m.RetryCount > 0 { // Only show experiments with retries
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
				m.Namespace,
				m.Name,
				m.Action,
				m.RetryCount,
				m.Phase,
				formatAge(time.Now().Add(-m.Age)),
			)
			count++
		}
	}

	if count == 0 {
		fmt.Fprintln(w, "No experiments with retries found")
	}

	w.Flush()
}

func printTopByAge(metrics []experimentMetrics, limit int) {
	// Sort by age descending (oldest first)
	sorted := make([]experimentMetrics, len(metrics))
	copy(sorted, metrics)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Age > sorted[j].Age
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAMESPACE\tNAME\tACTION\tPHASE\tAGE")

	for i := 0; i < limit && i < len(sorted); i++ {
		m := sorted[i]
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			m.Namespace,
			m.Name,
			m.Action,
			m.Phase,
			formatAge(time.Now().Add(-m.Age)),
		)
	}

	w.Flush()
}

func printFailed(metrics []experimentMetrics) {
	// Filter failed experiments
	var failed []experimentMetrics
	for _, m := range metrics {
		if m.Phase == "Failed" {
			failed = append(failed, m)
		}
	}

	if len(failed) == 0 {
		fmt.Println("No failed experiments")
		return
	}

	// Sort by age descending (most recent first)
	sort.Slice(failed, func(i, j int) bool {
		return failed[i].Age < failed[j].Age
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAMESPACE\tNAME\tACTION\tRETRIES\tAGE")

	for _, m := range failed {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
			m.Namespace,
			m.Name,
			m.Action,
			m.RetryCount,
			formatAge(time.Now().Add(-m.Age)),
		)
	}

	w.Flush()
}
