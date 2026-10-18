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
	"context"
	"errors"
	"strings"
	"testing"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"

	chaosv1alpha1 "github.com/neogan74/k8s-chaos/api/v1alpha1"
)

const (
	testControllerNS = "chaos-system"
	testDeployName   = "k8s-chaos-controller-manager"
	readyOneOfOne    = "1/1 ready"
	nameLabelKey     = "app.kubernetes.io/name"
	projectName      = "k8s-chaos"
)

func int32Ptr(i int32) *int32 { return &i }

func operatorDeployment(name string, labels map[string]string, replicas, ready int32, image string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testControllerNS, Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(replicas),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "manager", Image: image}}},
			},
		},
		Status: appsv1.DeploymentStatus{ReadyReplicas: ready},
	}
}

func chaosAPIResources(names ...string) []*metav1.APIResourceList {
	list := &metav1.APIResourceList{GroupVersion: chaosv1alpha1.GroupVersion.String()}
	for _, n := range names {
		list.APIResources = append(list.APIResources, metav1.APIResource{Name: n})
	}
	return []*metav1.APIResourceList{list}
}

func chaosWebhook() *admissionv1.ValidatingWebhookConfiguration {
	return &admissionv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: "validating-webhook-configuration"},
		Webhooks: []admissionv1.ValidatingWebhook{{
			Name: "vchaosexperiment.kb.io",
			Rules: []admissionv1.RuleWithOperations{{
				Rule: admissionv1.Rule{APIGroups: []string{"chaos.gushchin.dev"}, Resources: []string{resourceExperiments}},
			}},
		}},
	}
}

// newFakeChecker builds a checker on a fake clientset. denied lists "verb/resource"
// pairs that SelfSubjectAccessReview should refuse.
func newFakeChecker(t *testing.T, ns string, resources []*metav1.APIResourceList, denied map[string]bool,
	objs ...runtime.Object) *clusterChecker {
	t.Helper()
	cs := fake.NewSimpleClientset(objs...)

	disc, ok := cs.Discovery().(*fakediscovery.FakeDiscovery)
	if !ok {
		t.Fatal("expected fake discovery")
	}
	disc.FakedServerVersion = &version.Info{GitVersion: "v1.30.0"}
	disc.Resources = resources

	cs.PrependReactor("create", "selfsubjectaccessreviews",
		func(action clienttesting.Action) (bool, runtime.Object, error) {
			review, _ := action.(clienttesting.CreateAction).GetObject().(*authorizationv1.SelfSubjectAccessReview)
			attrs := review.Spec.ResourceAttributes
			review.Status.Allowed = !denied[attrs.Verb+"/"+attrs.Resource]
			return true, review, nil
		})
	return &clusterChecker{cs: cs, namespace: ns}
}

func findResult(t *testing.T, results []checkResult, prefix string) checkResult {
	t.Helper()
	for _, r := range results {
		if strings.HasPrefix(r.Name, prefix) {
			return r
		}
	}
	t.Fatalf("no result with name prefix %q in %+v", prefix, results)
	return checkResult{}
}

func TestCheck_AllHealthy(t *testing.T) {
	c := newFakeChecker(t, "", chaosAPIResources(resourceExperiments, resourceHistories), nil,
		operatorDeployment(testDeployName, map[string]string{nameLabelKey: projectName}, 1, 1, "img"),
		chaosWebhook(),
	)
	results := c.run(context.Background())

	if len(results) != 6 {
		t.Fatalf("expected 6 results, got %d: %+v", len(results), results)
	}
	for _, r := range results {
		if r.Status != statusPass {
			t.Errorf("%s: expected pass, got status %d (%s)", r.Name, r.Status, r.Detail)
		}
	}
	if got := findResult(t, results, "API server").Detail; !strings.Contains(got, "v1.30.0") {
		t.Errorf("connectivity detail should include the server version, got %q", got)
	}
}

func TestCheck_ConnectivityFailureSkipsRest(t *testing.T) {
	c := newFakeChecker(t, "", nil, nil)
	c.cs.(*fake.Clientset).PrependReactor("get", "version", func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("connection refused")
	})

	results := c.run(context.Background())
	if len(results) != 1 || results[0].Status != statusFail || !strings.Contains(results[0].Detail, "connection refused") {
		t.Fatalf("expected a single failed result mentioning the cause, got %+v", results)
	}
}

func TestCheck_CRDs(t *testing.T) {
	tests := []struct {
		name      string
		resources []*metav1.APIResourceList
		want      checkStatus
		contains  string
	}{
		{"both present", chaosAPIResources(resourceExperiments, resourceHistories), statusPass, "available"},
		{"history missing", chaosAPIResources(resourceExperiments), statusWarn, "history will not be recorded"},
		{"experiments missing", chaosAPIResources(resourceHistories), statusFail, "missing"},
		{"group not served", nil, statusFail, "not served"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := newFakeChecker(t, "", tc.resources, nil)
			got := c.checkCRDs()
			if got.Status != tc.want || !strings.Contains(got.Detail, tc.contains) {
				t.Fatalf("got status=%d detail=%q, want status=%d containing %q", got.Status, got.Detail, tc.want, tc.contains)
			}
			if tc.want != statusPass && got.Hint == "" {
				t.Error("expected a hint for a non-passing check")
			}
		})
	}
}

func TestCheck_Controller(t *testing.T) {
	nameLabel := map[string]string{nameLabelKey: projectName}
	cpLabel := map[string]string{"control-plane": "controller-manager"}

	tests := []struct {
		name     string
		objs     []runtime.Object
		want     checkStatus
		contains string
	}{
		{"ready", []runtime.Object{operatorDeployment(testDeployName, nameLabel, 1, 1, "x")},
			statusPass, readyOneOfOne},
		{"partially ready", []runtime.Object{operatorDeployment(testDeployName, nameLabel, 3, 1, "x")},
			statusWarn, "1/3"},
		{"none ready", []runtime.Object{operatorDeployment(testDeployName, nameLabel, 2, 0, "x")},
			statusFail, "0/2"},
		{"scaled to zero", []runtime.Object{operatorDeployment(testDeployName, nameLabel, 0, 0, "x")},
			statusFail, "scaled to 0"},
		{"not deployed", nil, statusWarn, "no k8s-chaos controller"},
		{"fallback label matched by name", []runtime.Object{operatorDeployment(testDeployName, cpLabel, 1, 1, "x")},
			statusPass, readyOneOfOne},
		{
			"fallback label matched by image",
			[]runtime.Object{operatorDeployment("manager", cpLabel, 1, 1, "ghcr.io/neogan74/k8s-chaos:v1")},
			statusPass, readyOneOfOne,
		},
		{"other operator with same control-plane label is ignored",
			[]runtime.Object{operatorDeployment("cert-manager-controller", cpLabel, 1, 0, "quay.io/jetstack/cert-manager")},
			statusWarn, "no k8s-chaos controller"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := newFakeChecker(t, "", chaosAPIResources(resourceExperiments), nil, tc.objs...)
			got := c.checkController(context.Background())
			if got.Status != tc.want || !strings.Contains(got.Detail, tc.contains) {
				t.Fatalf("got status=%d detail=%q, want status=%d containing %q", got.Status, got.Detail, tc.want, tc.contains)
			}
		})
	}
}

func TestCheck_ControllerScopedToNamespace(t *testing.T) {
	nameLabel := map[string]string{nameLabelKey: projectName}
	c := newFakeChecker(t, "other-ns", nil, nil, operatorDeployment("k8s-chaos-cm", nameLabel, 1, 1, "x"))
	if got := c.checkController(context.Background()); got.Status != statusWarn {
		t.Fatalf("deployment in another namespace must not be found, got %+v", got)
	}
}

func TestCheck_Webhook(t *testing.T) {
	c := newFakeChecker(t, "", nil, nil, chaosWebhook())
	if got := c.checkWebhook(context.Background()); got.Status != statusPass {
		t.Errorf("expected pass, got %+v", got)
	}

	c = newFakeChecker(t, "", nil, nil)
	if got := c.checkWebhook(context.Background()); got.Status != statusWarn {
		t.Errorf("expected warn when missing, got %+v", got)
	}

	other := chaosWebhook()
	other.Webhooks[0].Rules[0].Rule.APIGroups = []string{"apps"}
	c = newFakeChecker(t, "", nil, nil, other)
	if got := c.checkWebhook(context.Background()); got.Status != statusWarn {
		t.Errorf("a webhook for another group must not count, got %+v", got)
	}
}

func TestCheck_RBAC(t *testing.T) {
	tests := []struct {
		name     string
		denied   map[string]bool
		wantExp  checkStatus
		wantHist checkStatus
	}{
		{"everything allowed", nil, statusPass, statusPass},
		{"read-only user warns only", map[string]bool{"create/chaosexperiments": true, "delete/chaosexperiments": true},
			statusWarn, statusPass},
		{"cannot list experiments fails", map[string]bool{"list/chaosexperiments": true}, statusFail, statusPass},
		{"cannot read history only warns", map[string]bool{"list/chaosexperimenthistories": true}, statusPass, statusWarn},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := newFakeChecker(t, "chaos-testing", nil, tc.denied)
			results := c.checkRBAC(context.Background())
			exp := findResult(t, results, "RBAC: chaosexperiments")
			hist := findResult(t, results, "RBAC: chaosexperimenthistories")
			if exp.Status != tc.wantExp || hist.Status != tc.wantHist {
				t.Fatalf("got experiments=%d history=%d, want %d/%d (%+v)",
					exp.Status, hist.Status, tc.wantExp, tc.wantHist, results)
			}
			if !strings.Contains(exp.Name, "namespace chaos-testing") {
				t.Errorf("name should mention the namespace scope, got %q", exp.Name)
			}
		})
	}
}

func TestCheck_RBACEvaluationError(t *testing.T) {
	c := newFakeChecker(t, "", nil, nil)
	cs := c.cs.(*fake.Clientset)
	cs.PrependReactor("create", "selfsubjectaccessreviews", func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("boom")
	})
	got := c.checkRBAC(context.Background())
	if r := findResult(t, got, "RBAC: chaosexperiments"); r.Status != statusWarn || !strings.Contains(r.Detail, "boom") {
		t.Fatalf("an API error should warn, not fail: %+v", r)
	}
}

func TestPrintCheckResults(t *testing.T) {
	var out bytes.Buffer
	failed := printCheckResults(&out, []checkResult{
		{Name: "a", Status: statusPass, Detail: "fine", Hint: "ignored for passes"},
		{Name: "b", Status: statusWarn, Detail: "meh", Hint: "do x"},
		{Name: "c", Status: statusFail, Detail: "bad", Hint: "do y"},
	})
	if failed != 1 {
		t.Errorf("failed = %d, want 1", failed)
	}
	for _, want := range []string{"✓ a: fine", "! b: meh", "✗ c: bad", "hint: do x", "hint: do y",
		"3 check(s): 1 passed, 1 warning(s), 1 failed"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output missing %q:\n%s", want, out.String())
		}
	}
	if strings.Contains(out.String(), "ignored for passes") {
		t.Error("hints must not be printed for passing checks")
	}
}
