package cmd

import (
	"path/filepath"
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
