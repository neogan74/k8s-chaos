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

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"

	chaosv1alpha1 "github.com/neogan74/k8s-chaos/api/v1alpha1"
)

var describeCmd = &cobra.Command{
	Use:   "describe EXPERIMENT_NAME",
	Short: "Show detailed information about a chaos experiment",
	Long: `Display detailed information about a specific chaos experiment,
including its configuration, status, and execution history.

Examples:
  # Describe an experiment in the current namespace
  k8s-chaos describe nginx-chaos-demo

  # Describe an experiment in a specific namespace
  k8s-chaos describe nginx-chaos-demo -n chaos-testing`,
	Args: cobra.ExactArgs(1),
	RunE: runDescribe,
}

func init() {
	rootCmd.AddCommand(describeCmd)
}

func runDescribe(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	experimentName := args[0]

	if namespace == "" {
		return fmt.Errorf("namespace is required, use -n flag to specify")
	}

	k8sClient, err := getKubeClient()
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	exp := &chaosv1alpha1.ChaosExperiment{}
	if err := k8sClient.Get(ctx, types.NamespacedName{
		Name:      experimentName,
		Namespace: namespace,
	}, exp); err != nil {
		return fmt.Errorf("failed to get experiment: %w", err)
	}

	printExperimentDetails(exp)
	return nil
}

func printExperimentDetails(exp *chaosv1alpha1.ChaosExperiment) {
	fmt.Printf("Name:         %s\n", exp.Name)
	fmt.Printf("Namespace:    %s\n", exp.Namespace)
	fmt.Printf("Created:      %s (Age: %s)\n", exp.CreationTimestamp.Format("2006-01-02 15:04:05"), formatAge(exp.CreationTimestamp.Time))
	fmt.Println()

	fmt.Println("Spec:")
	fmt.Printf("  Action:              %s\n", exp.Spec.Action)
	fmt.Printf("  Target Namespace:    %s\n", exp.Spec.Namespace)
	fmt.Printf("  Selector:            %s\n", formatSelectorMultiline(exp.Spec.Selector))
	fmt.Printf("  Count:               %d\n", exp.Spec.Count)

	if exp.Spec.Duration != "" {
		fmt.Printf("  Duration:            %s\n", exp.Spec.Duration)
	}

	if exp.Spec.ExperimentDuration != "" {
		fmt.Printf("  Experiment Duration: %s\n", exp.Spec.ExperimentDuration)
	} else {
		fmt.Printf("  Experiment Duration: ∞ (runs indefinitely)\n")
	}

	fmt.Println()
	fmt.Println("Retry Configuration:")
	fmt.Printf("  Max Retries:         %d\n", exp.Spec.MaxRetries)
	fmt.Printf("  Retry Backoff:       %s\n", exp.Spec.RetryBackoff)
	fmt.Printf("  Retry Delay:         %s\n", exp.Spec.RetryDelay)

	fmt.Println()
	fmt.Println("Status:")
	fmt.Printf("  Phase:               %s\n", exp.Status.Phase)
	fmt.Printf("  Message:             %s\n", exp.Status.Message)

	if exp.Status.StartTime != nil {
		fmt.Printf("  Start Time:          %s\n", exp.Status.StartTime.Format("2006-01-02 15:04:05"))
	}

	if exp.Status.LastRunTime != nil {
		fmt.Printf("  Last Run Time:       %s\n", exp.Status.LastRunTime.Format("2006-01-02 15:04:05"))
	}

	if exp.Status.CompletedAt != nil {
		fmt.Printf("  Completed At:        %s\n", exp.Status.CompletedAt.Format("2006-01-02 15:04:05"))
	}

	if exp.Status.RetryCount > 0 {
		fmt.Printf("  Retry Count:         %d\n", exp.Status.RetryCount)
		if exp.Status.LastError != "" {
			fmt.Printf("  Last Error:          %s\n", exp.Status.LastError)
		}
		if exp.Status.NextRetryTime != nil {
			fmt.Printf("  Next Retry Time:     %s\n", exp.Status.NextRetryTime.Format("2006-01-02 15:04:05"))
		}
	}
}

func formatSelectorMultiline(selector map[string]string) string {
	if len(selector) == 0 {
		return "<none>"
	}

	result := ""
	first := true
	for k, v := range selector {
		if !first {
			result += "\n                       "
		}
		result += fmt.Sprintf("%s=%s", k, v)
		first = false
	}
	return result
}
