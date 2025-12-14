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
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"

	chaosv1alpha1 "github.com/neogan74/k8s-chaos/api/v1alpha1"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all chaos experiments",
	Long: `List all chaos experiments in the cluster or a specific namespace.

Examples:
  # List all experiments across all namespaces
  k8s-chaos list

  # List experiments in a specific namespace
  k8s-chaos list -n chaos-testing

  # List with wide output showing more details
  k8s-chaos list --wide`,
	Aliases: []string{"ls"},
	RunE:    runList,
}

var wideOutput bool

func init() {
	listCmd.Flags().BoolVarP(&wideOutput, "wide", "w", false, "show more details in output")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
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

	// Print table header and experiments
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

	if wideOutput {
		_, _ = fmt.Fprintln(w, "NAMESPACE\tNAME\tACTION\tTARGET-NS\tSELECTOR\tCOUNT\tPHASE\tRETRIES\tDURATION\tAGE")
	} else {
		_, _ = fmt.Fprintln(w, "NAMESPACE\tNAME\tACTION\tTARGET-NS\tPHASE\tAGE")
	}

	for _, exp := range expList.Items {
		age := formatAge(exp.CreationTimestamp.Time)
		selector := formatSelector(exp.Spec.Selector)

		if wideOutput {
			duration := exp.Spec.ExperimentDuration
			if duration == "" {
				duration = "∞"
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%s\t%d\t%s\t%s\n",
				exp.Namespace,
				exp.Name,
				exp.Spec.Action,
				exp.Spec.Namespace,
				selector,
				exp.Spec.Count,
				exp.Status.Phase,
				exp.Status.RetryCount,
				duration,
				age,
			)
		} else {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				exp.Namespace,
				exp.Name,
				exp.Spec.Action,
				exp.Spec.Namespace,
				exp.Status.Phase,
				age,
			)
		}
	}

	_ = w.Flush()
	return nil
}

// formatAge formats a time.Time to a human-readable age string
func formatAge(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	}
	return fmt.Sprintf("%dd", int(duration.Hours()/24))
}

// formatSelector formats a label selector map to a string
func formatSelector(selector map[string]string) string {
	if len(selector) == 0 {
		return "<none>"
	}

	pairs := make([]string, 0, len(selector))
	for k, v := range selector {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}

	result := strings.Join(pairs, ",")
	if len(result) > 30 {
		return result[:27] + "..."
	}
	return result
}
