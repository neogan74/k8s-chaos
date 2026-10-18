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
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"

	chaosv1alpha1 "github.com/neogan74/k8s-chaos/api/v1alpha1"
)

const (
	stdinPath    = "-"
	expectedKind = "ChaosExperiment"
)

var validateFiles []string

var validateCmd = &cobra.Command{
	Use:   "validate -f FILE|DIR|-",
	Short: "Validate ChaosExperiment manifests offline",
	Long: `Validate ChaosExperiment YAML files without contacting a cluster.

Checks performed:
  - YAML syntax and unknown/misspelled fields
  - apiVersion and kind
  - CRD schema rules (action, ranges, duration and size formats, enums)
  - Cross-field rules shared with the admission webhook (e.g. duration is
    required for pod-delay, valid cron schedule, time windows)

Not checked (needs a cluster): namespace existence, selector matches, and
production-namespace protection.

The command exits with a non-zero status when any document is invalid.

Examples:
  # Validate a single file
  k8s-chaos validate -f experiment.yaml

  # Validate every .yaml/.yml/.json file in a directory (recursive)
  k8s-chaos validate -f config/samples/

  # Validate from stdin
  cat experiment.yaml | k8s-chaos validate -f -`,
	Args: cobra.NoArgs,
	RunE: runValidate,
}

func init() {
	validateCmd.Flags().StringArrayVarP(&validateFiles, "filename", "f", nil,
		"file, directory or '-' for stdin containing ChaosExperiment manifests (repeatable)")
	_ = validateCmd.MarkFlagRequired("filename")
	rootCmd.AddCommand(validateCmd)
}

// validationResult is the outcome for one YAML document.
type validationResult struct {
	Source string // file name, or "<stdin>"
	Index  int    // 1-based document index within the source
	Name   string // metadata.name, when known
	Kind   string // kind of the document, when known
	Skip   bool   // not a ChaosExperiment; ignored
	Empty  bool   // empty or comment-only document; not reported
	Err    error  // nil when valid
}

func (r validationResult) label() string {
	label := r.Source
	if r.Index > 1 || r.Name == "" {
		label = fmt.Sprintf("%s[%d]", r.Source, r.Index)
	}
	if r.Name != "" {
		label = fmt.Sprintf("%s (%s)", label, r.Name)
	}
	return label
}

func runValidate(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()

	var results []validationResult
	for _, target := range validateFiles {
		res, err := validateTarget(cmd.InOrStdin(), target)
		if err != nil {
			return err
		}
		results = append(results, res...)
	}

	// Errors are reported in the summary below; do not print usage or the error twice.
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	checked, invalid := printValidationResults(out, results)
	if checked == 0 {
		return errors.New("no ChaosExperiment documents found")
	}
	if invalid > 0 {
		return fmt.Errorf("%d of %d document(s) failed validation", invalid, checked)
	}
	return nil
}

// printValidationResults prints one line per document plus a summary and
// returns how many ChaosExperiment documents were checked and how many failed.
func printValidationResults(out io.Writer, results []validationResult) (checked, invalid int) {
	skipped := 0
	for _, r := range results {
		switch {
		case r.Empty:
			continue
		case r.Skip:
			skipped++
			_, _ = fmt.Fprintf(out, "- %s: skipped (kind %s)\n", r.label(), r.Kind)
		case r.Err == nil:
			checked++
			_, _ = fmt.Fprintf(out, "✓ %s: valid\n", r.label())
		default:
			checked++
			invalid++
			_, _ = fmt.Fprintf(out, "✗ %s: invalid\n", r.label())
			for _, line := range errorLines(r.Err) {
				_, _ = fmt.Fprintf(out, "    - %s\n", line)
			}
		}
	}
	_, _ = fmt.Fprintf(out, "\n%d ChaosExperiment(s) checked, %d valid, %d invalid, %d other document(s) skipped\n",
		checked, checked-invalid, invalid, skipped)
	return checked, invalid
}

// errorLines splits joined errors into one line each.
func errorLines(err error) []string {
	return strings.Split(err.Error(), "\n")
}

// validateTarget validates a file, a directory (recursively) or stdin.
func validateTarget(stdin io.Reader, target string) ([]validationResult, error) {
	if target == stdinPath {
		return validateReader("<stdin>", stdin)
	}

	info, err := os.Stat(target)
	if err != nil {
		return nil, fmt.Errorf("cannot access %q: %w", target, err)
	}
	if !info.IsDir() {
		return validateFile(target)
	}

	var files []string
	walkErr := filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && isManifestFile(path) {
			files = append(files, path)
		}
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("cannot read directory %q: %w", target, walkErr)
	}
	sort.Strings(files)

	var results []validationResult
	for _, f := range files {
		res, err := validateFile(f)
		if err != nil {
			return nil, err
		}
		results = append(results, res...)
	}
	return results, nil
}

func isManifestFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml", ".json":
		return true
	}
	return false
}

func validateFile(path string) ([]validationResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return validateReader(path, f)
}

// validateReader validates every document in a (multi-document) YAML stream.
// Documents of a kind other than ChaosExperiment are reported as invalid so a
// wrong file is not silently accepted.
func validateReader(source string, r io.Reader) ([]validationResult, error) {
	reader := utilyaml.NewYAMLReader(bufio.NewReader(r))

	var results []validationResult
	for index := 1; ; {
		doc, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", source, err)
		}
		if len(strings.TrimSpace(string(doc))) == 0 {
			continue
		}
		results = append(results, validateDocument(source, index, doc))
		index++
	}

	if len(results) == 0 {
		results = append(results, validationResult{Source: source, Index: 1, Err: errors.New("no YAML documents found")})
	}
	return results, nil
}

func validateDocument(source string, index int, doc []byte) validationResult {
	res := validationResult{Source: source, Index: index}

	// Look at the type first (non-strict) so unrelated documents such as a
	// Deployment in the same file are skipped instead of reported as errors.
	var generic map[string]any
	if err := yaml.Unmarshal(doc, &generic); err != nil {
		res.Err = fmt.Errorf("cannot parse manifest: %w", err)
		return res
	}
	if len(generic) == 0 {
		res.Empty = true // comment-only or empty document
		return res
	}
	var meta metav1.TypeMeta
	_ = yaml.Unmarshal(doc, &meta)
	res.Kind = meta.Kind
	_, hasSpec := generic["spec"]
	switch {
	case meta.Kind != "" && meta.Kind != expectedKind:
		res.Skip = true
		return res
	case meta.Kind == "" && !hasSpec:
		// Not an experiment (e.g. a kustomization.yaml); a manifest with a
		// spec but no kind is still validated so the typo is reported.
		res.Skip = true
		res.Kind = "unknown"
		return res
	}

	var exp chaosv1alpha1.ChaosExperiment
	// UnmarshalStrict rejects unknown fields, which catches misspelled keys.
	if err := yaml.UnmarshalStrict(doc, &exp); err != nil {
		res.Err = fmt.Errorf("cannot parse manifest: %w", err)
		return res
	}
	res.Name = exp.Name

	var errs []error
	if exp.APIVersion != chaosv1alpha1.GroupVersion.String() {
		errs = append(errs, fmt.Errorf("apiVersion: got %q, want %q", exp.APIVersion, chaosv1alpha1.GroupVersion.String()))
	}
	if exp.Kind != expectedKind {
		errs = append(errs, fmt.Errorf("kind: got %q, want %q", exp.Kind, expectedKind))
	}
	if exp.Name == "" {
		errs = append(errs, errors.New("metadata.name: required"))
	}
	if len(errs) > 0 {
		res.Err = errors.Join(errs...)
		return res
	}

	res.Err = exp.ValidateStatic()
	return res
}
