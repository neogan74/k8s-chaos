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

var (
	force bool
)

var deleteCmd = &cobra.Command{
	Use:   "delete EXPERIMENT_NAME",
	Short: "Delete a chaos experiment",
	Long: `Delete a chaos experiment from the cluster.

Examples:
  # Delete an experiment (will prompt for confirmation)
  k8s-chaos delete nginx-chaos-demo -n chaos-testing

  # Delete without confirmation
  k8s-chaos delete nginx-chaos-demo -n chaos-testing --force`,
	Args: cobra.ExactArgs(1),
	RunE: runDelete,
}

func init() {
	deleteCmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation prompt")
	rootCmd.AddCommand(deleteCmd)
}

func runDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	experimentName := args[0]

	if namespace == "" {
		return fmt.Errorf("namespace is required, use -n flag to specify")
	}

	k8sClient, err := getKubeClient()
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	// Get experiment first to verify it exists
	exp := &chaosv1alpha1.ChaosExperiment{}
	if err := k8sClient.Get(ctx, types.NamespacedName{
		Name:      experimentName,
		Namespace: namespace,
	}, exp); err != nil {
		return fmt.Errorf("failed to get experiment: %w", err)
	}

	// Prompt for confirmation unless --force is used
	if !force {
		fmt.Printf("Are you sure you want to delete experiment '%s' in namespace '%s'? (y/N): ", experimentName, namespace)
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			// Ignore scan errors (e.g., empty input), treat as "no"
			fmt.Println("Deletion cancelled")
			return nil
		}
		if response != "y" && response != "Y" && response != "yes" {
			fmt.Println("Deletion cancelled")
			return nil
		}
	}

	// Delete the experiment
	if err := k8sClient.Delete(ctx, exp); err != nil {
		return fmt.Errorf("failed to delete experiment: %w", err)
	}

	fmt.Printf("Experiment '%s' deleted successfully\n", experimentName)
	return nil
}
