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
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	chaosv1alpha1 "github.com/neogan74/k8s-chaos/api/v1alpha1"
)

const (
	operatorNameLabel   = "app.kubernetes.io/name=k8s-chaos"
	controlPlaneLabel   = "control-plane=controller-manager"
	operatorNameMarker  = "k8s-chaos"
	resourceExperiments = "chaosexperiments"
	resourceHistories   = "chaosexperimenthistories"

	verbList   = "list"
	verbGet    = "get"
	verbCreate = "create"
	verbDelete = "delete"
)

var checkTimeout time.Duration

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check that the cluster is ready to run chaos experiments",
	Long: `Run a set of read-only health checks against the current cluster:

  - API server connectivity
  - ChaosExperiment CRDs are installed
  - The k8s-chaos controller is deployed and ready
  - The validating admission webhook is registered
  - The current user has the permissions the CLI needs (RBAC)

Each check reports ✓ (ok), ! (warning) or ✗ (failure). The command exits with
a non-zero status if any check fails; warnings do not affect the exit code.

Use -n to check permissions and look for the controller in a single namespace.

Examples:
  # Check the current cluster
  k8s-chaos check

  # Check permissions for one namespace
  k8s-chaos check -n chaos-testing`,
	Args: cobra.NoArgs,
	RunE: runCheck,
}

func init() {
	checkCmd.Flags().DurationVar(&checkTimeout, "timeout", 30*time.Second, "overall timeout for all checks")
	rootCmd.AddCommand(checkCmd)
}

type checkStatus int

const (
	statusPass checkStatus = iota
	statusWarn
	statusFail
)

func (s checkStatus) symbol() string {
	switch s {
	case statusPass:
		return "✓"
	case statusWarn:
		return "!"
	default:
		return "✗"
	}
}

// checkResult is the outcome of one readiness check.
type checkResult struct {
	Name   string
	Status checkStatus
	Detail string
	Hint   string // how to fix a warning or failure
}

// clusterChecker runs the readiness checks against a cluster.
type clusterChecker struct {
	cs        kubernetes.Interface
	namespace string // empty means all namespaces
}

func runCheck(cmd *cobra.Command, _ []string) error {
	// Results are printed below and Execute() prints the returned error once,
	// so keep cobra from printing usage or the error itself.
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	config, err := getRestConfig()
	if err != nil {
		return err
	}
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), checkTimeout)
	defer cancel()

	checker := &clusterChecker{cs: cs, namespace: namespace}
	results := checker.run(ctx)
	if failed := printCheckResults(cmd.OutOrStdout(), results); failed > 0 {
		return fmt.Errorf("%d check(s) failed", failed)
	}
	return nil
}

// run executes every check in order. If the API server is unreachable the
// remaining checks cannot say anything useful, so they are skipped.
func (c *clusterChecker) run(ctx context.Context) []checkResult {
	conn := c.checkConnectivity()
	results := []checkResult{conn}
	if conn.Status == statusFail {
		return results
	}
	results = append(results, c.checkCRDs())
	results = append(results, c.checkController(ctx))
	results = append(results, c.checkWebhook(ctx))
	results = append(results, c.checkRBAC(ctx)...)
	return results
}

func (c *clusterChecker) checkConnectivity() checkResult {
	res := checkResult{Name: "API server connectivity"}
	version, err := c.cs.Discovery().ServerVersion()
	if err != nil {
		res.Status = statusFail
		res.Detail = fmt.Sprintf("cannot reach the API server: %v", err)
		res.Hint = "check your kubeconfig and context (kubectl config current-context)"
		return res
	}
	res.Detail = fmt.Sprintf("connected, Kubernetes %s", version.GitVersion)
	return res
}

func (c *clusterChecker) checkCRDs() checkResult {
	res := checkResult{Name: "CRDs installed"}
	groupVersion := chaosv1alpha1.GroupVersion.String()

	list, err := c.cs.Discovery().ServerResourcesForGroupVersion(groupVersion)
	if err != nil {
		if apierrors.IsNotFound(err) {
			res.Status = statusFail
			res.Detail = fmt.Sprintf("API group %s is not served", groupVersion)
			res.Hint = "install the CRDs: make install (or install the Helm chart / dist/install.yaml)"
			return res
		}
		res.Status = statusFail
		res.Detail = fmt.Sprintf("cannot discover %s: %v", groupVersion, err)
		return res
	}

	found := map[string]bool{}
	for _, r := range list.APIResources {
		found[r.Name] = true
	}
	switch {
	case !found[resourceExperiments]:
		res.Status = statusFail
		res.Detail = fmt.Sprintf("%s.%s is missing", resourceExperiments, chaosv1alpha1.GroupVersion.Group)
		res.Hint = "install the CRDs: make install"
	case !found[resourceHistories]:
		res.Status = statusWarn
		res.Detail = fmt.Sprintf("%s is installed, but %s is missing (history will not be recorded)",
			resourceExperiments, resourceHistories)
		res.Hint = "re-apply the CRDs: make install"
	default:
		res.Detail = fmt.Sprintf("%s and %s are available", resourceExperiments, resourceHistories)
	}
	return res
}

// findOperatorDeployments returns the k8s-chaos controller deployments. It
// first looks for the standard name label and falls back to the kubebuilder
// control-plane label, filtered so other operators are not mistaken for ours.
func (c *clusterChecker) findOperatorDeployments(ctx context.Context) ([]appsv1.Deployment, error) {
	deployments := c.cs.AppsV1().Deployments(c.namespace)

	list, err := deployments.List(ctx, metav1.ListOptions{LabelSelector: operatorNameLabel})
	if err != nil {
		return nil, err
	}
	if len(list.Items) > 0 {
		return list.Items, nil
	}

	list, err = deployments.List(ctx, metav1.ListOptions{LabelSelector: controlPlaneLabel})
	if err != nil {
		return nil, err
	}
	var found []appsv1.Deployment
	for _, d := range list.Items {
		if isOperatorDeployment(d) {
			found = append(found, d)
		}
	}
	return found, nil
}

func isOperatorDeployment(d appsv1.Deployment) bool {
	if strings.Contains(d.Name, operatorNameMarker) {
		return true
	}
	for _, container := range d.Spec.Template.Spec.Containers {
		if strings.Contains(container.Image, operatorNameMarker) {
			return true
		}
	}
	return false
}

func (c *clusterChecker) checkController(ctx context.Context) checkResult {
	res := checkResult{Name: "Controller deployed and ready"}

	deployments, err := c.findOperatorDeployments(ctx)
	if err != nil {
		res.Status = statusWarn
		res.Detail = fmt.Sprintf("cannot list deployments: %v", err)
		res.Hint = "this needs permission to list deployments; the controller may still be running"
		return res
	}
	if len(deployments) == 0 {
		res.Status = statusWarn
		res.Detail = "no k8s-chaos controller deployment found"
		res.Hint = "deploy it with make deploy or the Helm chart (ignore this if you run it locally with make run)"
		return res
	}

	var problems []string
	var ok []string
	for _, d := range deployments {
		want := int32(1)
		if d.Spec.Replicas != nil {
			want = *d.Spec.Replicas
		}
		id := fmt.Sprintf("%s/%s", d.Namespace, d.Name)
		switch {
		case want == 0:
			res.Status = statusFail
			problems = append(problems, fmt.Sprintf("%s is scaled to 0 replicas", id))
		case d.Status.ReadyReplicas == 0:
			res.Status = statusFail
			problems = append(problems, fmt.Sprintf("%s has 0/%d ready replicas", id, want))
		case d.Status.ReadyReplicas < want:
			if res.Status < statusWarn {
				res.Status = statusWarn
			}
			problems = append(problems, fmt.Sprintf("%s has %d/%d ready replicas", id, d.Status.ReadyReplicas, want))
		default:
			ok = append(ok, fmt.Sprintf("%s (%d/%d ready)", id, d.Status.ReadyReplicas, want))
		}
	}

	if len(problems) > 0 {
		res.Detail = strings.Join(append(problems, ok...), "; ")
		res.Hint = "inspect it with: kubectl -n <namespace> describe deployment <name> and kubectl logs"
		return res
	}
	res.Detail = strings.Join(ok, "; ")
	return res
}

func (c *clusterChecker) checkWebhook(ctx context.Context) checkResult {
	res := checkResult{Name: "Admission webhook registered"}

	configs, err := c.cs.AdmissionregistrationV1().ValidatingWebhookConfigurations().List(ctx, metav1.ListOptions{})
	if err != nil {
		res.Status = statusWarn
		res.Detail = fmt.Sprintf("cannot list ValidatingWebhookConfigurations: %v", err)
		return res
	}

	group := chaosv1alpha1.GroupVersion.Group
	for _, cfg := range configs.Items {
		for _, wh := range cfg.Webhooks {
			for _, rule := range wh.Rules {
				if containsAny(rule.APIGroups, group) && containsAny(rule.Resources, resourceExperiments) {
					res.Detail = fmt.Sprintf("%s (%s)", cfg.Name, wh.Name)
					return res
				}
			}
		}
	}

	res.Status = statusWarn
	res.Detail = "no validating webhook for chaosexperiments found"
	res.Hint = "without it namespace, selector and production-protection checks only happen at reconcile time"
	return res
}

// rbacRequirement is a set of verbs the CLI needs on one resource.
type rbacRequirement struct {
	resource string
	verbs    []string
	// required verbs make the check fail when denied; others only warn
	required map[string]bool
}

func (c *clusterChecker) checkRBAC(ctx context.Context) []checkResult {
	requirements := []rbacRequirement{
		{
			resourceExperiments,
			[]string{verbList, verbGet, verbCreate, verbDelete},
			map[string]bool{verbList: true, verbGet: true},
		},
		{resourceHistories, []string{verbList, verbGet}, map[string]bool{}},
	}

	results := make([]checkResult, 0, len(requirements))
	for _, req := range requirements {
		results = append(results, c.checkResourcePermissions(ctx, req))
	}
	return results
}

func (c *clusterChecker) checkResourcePermissions(ctx context.Context, req rbacRequirement) checkResult {
	scope := "all namespaces"
	if c.namespace != "" {
		scope = "namespace " + c.namespace
	}
	res := checkResult{Name: fmt.Sprintf("RBAC: %s (%s)", req.resource, scope)}

	denied := make([]string, 0, len(req.verbs))
	var deniedRequired []string
	for _, verb := range req.verbs {
		allowed, err := c.canI(ctx, verb, req.resource)
		if err != nil {
			res.Status = statusWarn
			res.Detail = fmt.Sprintf("cannot evaluate permissions: %v", err)
			return res
		}
		if allowed {
			continue
		}
		denied = append(denied, verb)
		if req.required[verb] {
			deniedRequired = append(deniedRequired, verb)
		}
	}

	switch {
	case len(denied) == 0:
		res.Detail = "allowed: " + strings.Join(req.verbs, ", ")
	case len(deniedRequired) > 0:
		res.Status = statusFail
		res.Detail = "denied: " + strings.Join(denied, ", ")
		res.Hint = fmt.Sprintf("grant these verbs on %s.%s to your user (a Role/ClusterRole and binding)",
			req.resource, chaosv1alpha1.GroupVersion.Group)
	default:
		res.Status = statusWarn
		res.Detail = "denied: " + strings.Join(denied, ", ") + " (read-only access is enough for most commands)"
	}
	return res
}

func (c *clusterChecker) canI(ctx context.Context, verb, resource string) (bool, error) {
	review := &authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: c.namespace,
				Verb:      verb,
				Group:     chaosv1alpha1.GroupVersion.Group,
				Resource:  resource,
			},
		},
	}
	resp, err := c.cs.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, review, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}
	if resp.Status.Allowed {
		return true, nil
	}
	if resp.Status.EvaluationError != "" && !resp.Status.Denied {
		return false, errors.New(resp.Status.EvaluationError)
	}
	return false, nil
}

func containsAny(list []string, wanted string) bool {
	for _, item := range list {
		if item == wanted || item == "*" {
			return true
		}
	}
	return false
}

// printCheckResults prints one block per check plus a summary and returns the
// number of failed checks.
func printCheckResults(out io.Writer, results []checkResult) int {
	var pass, warn, fail int
	for _, r := range results {
		switch r.Status {
		case statusPass:
			pass++
		case statusWarn:
			warn++
		default:
			fail++
		}
		_, _ = fmt.Fprintf(out, "%s %s: %s\n", r.Status.symbol(), r.Name, r.Detail)
		if r.Hint != "" && r.Status != statusPass {
			_, _ = fmt.Fprintf(out, "    hint: %s\n", r.Hint)
		}
	}
	_, _ = fmt.Fprintf(out, "\n%d check(s): %d passed, %d warning(s), %d failed\n", len(results), pass, warn, fail)
	return fail
}
