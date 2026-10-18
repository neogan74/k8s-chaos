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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testActionKill = "pod-kill"
	wantSkip       = "SKIP"
)

const validDoc = `apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: ok
spec:
  action: pod-kill
  namespace: default
  selector:
    app: nginx
`

func TestValidateReader(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCount int
		wantErr   []string // per-document substring; "" means valid; wantSkip means skipped
	}{
		{"valid", validDoc, 1, []string{""}},
		{
			"unknown field is caught",
			strings.Replace(validDoc, "action: pod-kill", "action: pod-kill\n  coutn: 2", 1),
			1, []string{"unknown field"},
		},
		{
			"wrong kind field values",
			strings.Replace(validDoc, "action: pod-kill", "action: pod-explode", 1),
			1, []string{"unsupported value"},
		},
		{
			"wrong apiVersion",
			strings.Replace(validDoc, "chaos.gushchin.dev/v1alpha1", "chaos.gushchin.dev/v1", 1),
			1, []string{"apiVersion"},
		},
		{
			"missing name",
			strings.Replace(validDoc, "  name: ok\n", "  labels: {}\n", 1),
			1, []string{"metadata.name"},
		},
		{"spec without kind is an error", "spec:\n  action: pod-kill\n", 1, []string{"kind"}},
		{"other kind is skipped", "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: x\n", 1, []string{wantSkip}},
		{"no kind and no spec is skipped", "resources:\n- a.yaml\n", 1, []string{wantSkip}},
		{
			"multi document",
			validDoc + "---\n" + strings.Replace(validDoc, testActionKill, "nope", 1),
			2, []string{"", "unsupported value"},
		},
		{"leading separator and comment-only doc", "---\n# nothing\n---\n" + validDoc, 1, []string{""}},
		{"invalid yaml", "spec: [unclosed\n", 1, []string{"cannot parse"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			results, err := validateReader("test.yaml", strings.NewReader(tc.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var got []validationResult
			for _, r := range results {
				if !r.Empty {
					got = append(got, r)
				}
			}
			if len(got) != tc.wantCount {
				t.Fatalf("got %d results, want %d: %+v", len(got), tc.wantCount, got)
			}
			for i, want := range tc.wantErr {
				r := got[i]
				switch want {
				case wantSkip:
					if !r.Skip {
						t.Errorf("doc %d: expected skip, got %+v", i, r)
					}
				case "":
					if r.Err != nil || r.Skip {
						t.Errorf("doc %d: expected valid, got %+v", i, r)
					}
				default:
					if r.Err == nil || !strings.Contains(r.Err.Error(), want) {
						t.Errorf("doc %d: expected error containing %q, got %v", i, want, r.Err)
					}
				}
			}
		})
	}
}

func TestValidateReader_EmptyInput(t *testing.T) {
	results, err := validateReader("empty.yaml", strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Err == nil {
		t.Fatalf("expected a single 'no documents' error, got %+v", results)
	}
}

func TestValidateTarget_DirectoryAndFile(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("a.yaml", validDoc)
	write("sub/b.yml", strings.Replace(validDoc, testActionKill, "nope", 1))
	write("notes.txt", "ignored")

	results, err := validateTarget(nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results (txt ignored), got %d", len(results))
	}
	if results[0].Err != nil || results[1].Err == nil {
		t.Fatalf("unexpected results: %+v", results)
	}

	if _, err := validateTarget(nil, filepath.Join(dir, "missing.yaml")); err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestValidateTarget_Stdin(t *testing.T) {
	results, err := validateTarget(strings.NewReader(validDoc), "-")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Err != nil || results[0].Source != "<stdin>" {
		t.Fatalf("unexpected results: %+v", results)
	}
}

// runValidateCmd runs `validate -f -` with the given stdin and returns its output.
func runValidateCmd(t *testing.T, stdin string) (string, error) {
	t.Helper()
	orig := validateFiles
	t.Cleanup(func() { validateFiles = orig })
	validateFiles = []string{stdinPath}

	var out bytes.Buffer
	validateCmd.SetOut(&out)
	validateCmd.SetIn(strings.NewReader(stdin))
	t.Cleanup(func() { validateCmd.SetOut(nil); validateCmd.SetIn(nil) })

	err := runValidate(validateCmd, nil)
	return out.String(), err
}

func TestRunValidate_Output(t *testing.T) {
	out, err := runValidateCmd(t, validDoc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"✓ <stdin> (ok): valid", "1 ChaosExperiment(s) checked, 1 valid, 0 invalid"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}

	bad := strings.Replace(validDoc, testActionKill, "pod-delay", 1) // pod-delay requires duration
	out, err = runValidateCmd(t, bad)
	if err == nil {
		t.Fatal("expected non-zero result for invalid document")
	}
	for _, want := range []string{"✗ <stdin> (ok): invalid", "duration is required", "0 valid, 1 invalid"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRunValidate_NoExperimentsIsAnError(t *testing.T) {
	_, err := runValidateCmd(t, "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: x\n")
	if err == nil || !strings.Contains(err.Error(), "no ChaosExperiment documents") {
		t.Fatalf("expected 'no ChaosExperiment documents' error, got %v", err)
	}
}

// TestSamplesAreValid makes sure every shipped sample passes offline validation,
// so broken examples are caught in CI.
func TestSamplesAreValid(t *testing.T) {
	results, err := validateTarget(nil, filepath.Join("..", "..", "..", "config", "samples"))
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, r := range results {
		if r.Skip || r.Empty {
			continue
		}
		checked++
		if r.Err != nil {
			t.Errorf("%s is invalid: %v", r.label(), r.Err)
		}
	}
	if checked == 0 {
		t.Fatal("no sample experiments were checked")
	}
}
