package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetKubeconfigPath_PrefersFlag(t *testing.T) {
	orig := kubeconfig
	t.Cleanup(func() { kubeconfig = orig })

	t.Setenv("KUBECONFIG", "/tmp/envconfig")
	kubeconfig = "/tmp/flagconfig"

	got := getKubeconfigPath()
	if got != "/tmp/flagconfig" {
		t.Fatalf("expected flag kubeconfig, got %s", got)
	}
}

func TestGetKubeconfigPath_UsesEnv(t *testing.T) {
	orig := kubeconfig
	t.Cleanup(func() { kubeconfig = orig })

	kubeconfig = ""
	t.Setenv("KUBECONFIG", "/tmp/from-env")

	got := getKubeconfigPath()
	if got != "/tmp/from-env" {
		t.Fatalf("expected env kubeconfig, got %s", got)
	}
}

func TestGetKubeconfigPath_DefaultHome(t *testing.T) {
	orig := kubeconfig
	t.Cleanup(func() { kubeconfig = orig })

	kubeconfig = ""
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("KUBECONFIG", "")

	got := getKubeconfigPath()
	want := filepath.Join(home, ".kube", "config")
	if got != want {
		t.Fatalf("expected default kubeconfig %s, got %s", want, got)
	}
}

func TestRootCmd_Name(t *testing.T) {
	if rootCmd.Use != "k8s-chaos" {
		t.Fatalf("expected root command use to be 'k8s-chaos', got %s", rootCmd.Use)
	}
}

func TestRootCmd_Version(t *testing.T) {
	if rootCmd.Version == "" {
		t.Fatal("expected root command to have a version set")
	}
}

func TestRootCmd_HasSubcommands(t *testing.T) {
	expectedCommands := []string{"list", "describe", "delete", "stats", "top"}

	commands := rootCmd.Commands()
	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name()] = true
	}

	for _, expected := range expectedCommands {
		if !commandNames[expected] {
			t.Errorf("expected subcommand '%s' not found", expected)
		}
	}
}

func TestRootCmd_PersistentFlags(t *testing.T) {
	// Check kubeconfig flag exists
	kubeconfigFlag := rootCmd.PersistentFlags().Lookup("kubeconfig")
	if kubeconfigFlag == nil {
		t.Fatal("expected --kubeconfig persistent flag")
	}

	// Check namespace flag exists with shorthand
	namespaceFlag := rootCmd.PersistentFlags().Lookup("namespace")
	if namespaceFlag == nil {
		t.Fatal("expected --namespace persistent flag")
	}
	if namespaceFlag.Shorthand != "n" {
		t.Fatalf("expected namespace shorthand '-n', got '-%s'", namespaceFlag.Shorthand)
	}
}

func TestRootCmd_HelpOutput(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Check for expected content in help
	expectedStrings := []string{
		"k8s-chaos",
		"chaos engineering",
		"list",
		"describe",
		"delete",
		"stats",
		"top",
		"--kubeconfig",
		"--namespace",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(strings.ToLower(output), strings.ToLower(expected)) {
			t.Errorf("expected help output to contain '%s'", expected)
		}
	}
}

func TestListCmd_HasAlias(t *testing.T) {
	if len(listCmd.Aliases) == 0 {
		t.Fatal("expected list command to have aliases")
	}

	hasLsAlias := false
	for _, alias := range listCmd.Aliases {
		if alias == "ls" {
			hasLsAlias = true
			break
		}
	}

	if !hasLsAlias {
		t.Fatal("expected list command to have 'ls' alias")
	}
}

func TestListCmd_WideFlag(t *testing.T) {
	wideFlag := listCmd.Flags().Lookup("wide")
	if wideFlag == nil {
		t.Fatal("expected --wide flag on list command")
	}
	if wideFlag.Shorthand != "w" {
		t.Fatalf("expected wide shorthand '-w', got '-%s'", wideFlag.Shorthand)
	}
}

func TestDeleteCmd_ForceFlag(t *testing.T) {
	forceFlag := deleteCmd.Flags().Lookup("force")
	if forceFlag == nil {
		t.Fatal("expected --force flag on delete command")
	}
	if forceFlag.Shorthand != "f" {
		t.Fatalf("expected force shorthand '-f', got '-%s'", forceFlag.Shorthand)
	}
}

func TestTopCmd_LimitFlag(t *testing.T) {
	limitFlag := topCmd.Flags().Lookup("limit")
	if limitFlag == nil {
		t.Fatal("expected --limit flag on top command")
	}
	if limitFlag.Shorthand != "l" {
		t.Fatalf("expected limit shorthand '-l', got '-%s'", limitFlag.Shorthand)
	}
}

func TestDescribeCmd_RequiresArg(t *testing.T) {
	if describeCmd.Args == nil {
		t.Fatal("expected describe command to have args validation")
	}
}
