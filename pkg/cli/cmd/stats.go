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
	"text/tabwriter"

	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"

	chaosv1alpha1 "github.com/neogan74/k8s-chaos/api/v1alpha1"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show chaos experiment statistics",
	Long: `Display statistics about chaos experiments in the cluster,
including total experiments, success/failure rates, and experiment phases.

Examples:
  # Show stats for all experiments
  k8s-chaos stats

  # Show stats for a specific namespace
  k8s-chaos stats -n chaos-testing`,
	RunE: runStats,
}

func init() {
	rootCmd.AddCommand(statsCmd)
}

type stats struct {
	Total       int
	Running     int
	Completed   int
	Failed      int
	Pending     int
	ByAction    map[string]int
	WithRetry   int
	TimeLimited int
}

func runStats(cmd *cobra.Command, args []string) error {
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

	stats := calculateStats(expList.Items)
	printStats(stats, namespace)

	return nil
}

func calculateStats(experiments []chaosv1alpha1.ChaosExperiment) stats {
	s := stats{
		ByAction: make(map[string]int),
	}

	for _, exp := range experiments {
		s.Total++

		// Count by phase
		switch exp.Status.Phase {
		case "Running":
			s.Running++
		case "Completed":
			s.Completed++
		case "Failed":
			s.Failed++
		case "Pending":
			s.Pending++
		}

		// Count by action
		s.ByAction[exp.Spec.Action]++

		// Count experiments with retry configuration
		if exp.Spec.MaxRetries > 0 {
			s.WithRetry++
		}

		// Count time-limited experiments
		if exp.Spec.ExperimentDuration != "" {
			s.TimeLimited++
		}
	}

	return s
}

func printStats(s stats, ns string) {
	fmt.Println("=== Chaos Experiment Statistics ===")
	if ns != "" {
		fmt.Printf("Namespace: %s\n", ns)
	} else {
		fmt.Println("Namespace: All namespaces")
	}
	fmt.Println()

	// Overall stats
	fmt.Println("Overall:")
	fmt.Printf("  Total Experiments:   %d\n", s.Total)
	fmt.Printf("  Running:             %d\n", s.Running)
	fmt.Printf("  Completed:           %d\n", s.Completed)
	fmt.Printf("  Failed:              %d\n", s.Failed)
	fmt.Printf("  Pending:             %d\n", s.Pending)
	fmt.Println()

	// Success rate
	if s.Total > 0 {
		successRate := float64(s.Completed) / float64(s.Total) * 100
		failureRate := float64(s.Failed) / float64(s.Total) * 100
		fmt.Println("Success Rate:")
		fmt.Printf("  Successful:          %.1f%%\n", successRate)
		fmt.Printf("  Failed:              %.1f%%\n", failureRate)
		fmt.Println()
	}

	// Experiments by action
	if len(s.ByAction) > 0 {
		fmt.Println("By Action:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "  ACTION\tCOUNT\tPERCENTAGE")
		for action, count := range s.ByAction {
			percentage := float64(count) / float64(s.Total) * 100
			fmt.Fprintf(w, "  %s\t%d\t%.1f%%\n", action, count, percentage)
		}
		w.Flush()
		fmt.Println()
	}

	// Configuration stats
	fmt.Println("Configuration:")
	fmt.Printf("  With Retry Logic:    %d (%.1f%%)\n", s.WithRetry, float64(s.WithRetry)/float64(s.Total)*100)
	fmt.Printf("  Time-Limited:        %d (%.1f%%)\n", s.TimeLimited, float64(s.TimeLimited)/float64(s.Total)*100)
	fmt.Printf("  Indefinite:          %d (%.1f%%)\n", s.Total-s.TimeLimited, float64(s.Total-s.TimeLimited)/float64(s.Total)*100)
}
