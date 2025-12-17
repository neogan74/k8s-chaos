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

package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestCLI_Build(t *testing.T) {
	// Verify CLI binary can be built
	cmd := exec.Command("go", "build", "-o", "/dev/null", ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build CLI: %v", err)
	}
}

func TestCLI_HelpFlag(t *testing.T) {
	// Run the CLI with --help and verify output
	cmd := exec.Command("go", "run", ".", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run CLI with --help: %v", err)
	}

	outputStr := string(output)

	expectedStrings := []string{
		"k8s-chaos",
		"chaos engineering",
		"list",
		"describe",
		"delete",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(strings.ToLower(outputStr), strings.ToLower(expected)) {
			t.Errorf("expected help output to contain '%s'", expected)
		}
	}
}

func TestCLI_VersionFlag(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run CLI with --version: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "k8s-chaos") {
		t.Errorf("expected version output to contain 'k8s-chaos', got: %s", outputStr)
	}
}

func TestCLI_ListHelp(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "list", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run CLI list --help: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "--wide") {
		t.Errorf("expected list help to contain '--wide' flag")
	}
}

func TestCLI_DescribeHelp(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "describe", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run CLI describe --help: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "EXPERIMENT_NAME") {
		t.Errorf("expected describe help to show EXPERIMENT_NAME usage")
	}
}

func TestCLI_DeleteHelp(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "delete", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run CLI delete --help: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "--force") {
		t.Errorf("expected delete help to contain '--force' flag")
	}
}

func TestCLI_StatsHelp(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "stats", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run CLI stats --help: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(strings.ToLower(outputStr), "statistics") {
		t.Errorf("expected stats help to mention 'statistics'")
	}
}

func TestCLI_TopHelp(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "top", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run CLI top --help: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "--limit") {
		t.Errorf("expected top help to contain '--limit' flag")
	}
}
