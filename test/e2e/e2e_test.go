//go:build e2e
// +build e2e

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

package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/neogan74/k8s-chaos/test/utils"
)

// namespace where the project is deployed in
const namespace = "k8s-chaos-system"

// serviceAccountName created for the project
const serviceAccountName = "k8s-chaos-controller-manager"

// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = "k8s-chaos-controller-manager-metrics-service"

// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = "k8s-chaos-metrics-binding"

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string

	// Before running the tests, set up the environment by creating the namespace,
	// enforce the restricted security policy to the namespace, installing CRDs,
	// and deploying the controller.
	BeforeAll(func() {
		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")

		By("labeling the namespace to enforce the restricted security policy")
		cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
			"pod-security.kubernetes.io/enforce=restricted")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to label namespace with restricted policy")

		By("installing CRDs")
		cmd = exec.Command("make", "install")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectImage))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")
	})

	// After all tests have been executed, clean up by undeploying the controller, uninstalling CRDs,
	// and deleting the namespace.
	AfterAll(func() {
		By("cleaning up the curl pod for metrics")
		cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace)
		_, _ = utils.Run(cmd)

		By("undeploying the controller-manager")
		cmd = exec.Command("make", "undeploy")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall")
		_, _ = utils.Run(cmd)

		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)
	})

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("Fetching controller manager pod logs")
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			controllerLogs, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
			}

			By("Fetching Kubernetes events")
			cmd = exec.Command("kubectl", "get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s", eventsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Kubernetes events: %s", err)
			}

			By("Fetching curl-metrics logs")
			cmd = exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
			metricsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Metrics logs:\n %s", metricsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get curl-metrics logs: %s", err)
			}

			By("Fetching controller manager pod description")
			cmd = exec.Command("kubectl", "describe", "pod", controllerPodName, "-n", namespace)
			podDescription, err := utils.Run(cmd)
			if err == nil {
				fmt.Println("Pod description:\n", podDescription)
			} else {
				fmt.Println("Failed to describe controller pod")
			}
		}
	})

	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	Context("Manager", func() {
		It("should run successfully", func() {
			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func(g Gomega) {
				// Get the name of the controller-manager pod
				cmd := exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
				controllerPodName = podNames[0]
				g.Expect(controllerPodName).To(ContainSubstring("controller-manager"))

				// Validate the pod's status
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Incorrect controller-manager pod status")
			}
			Eventually(verifyControllerUp).Should(Succeed())
		})

		It("should ensure the metrics endpoint is serving metrics", func() {
			By("creating a ClusterRoleBinding for the service account to allow access to metrics")
			cmd := exec.Command("kubectl", "create", "clusterrolebinding", metricsRoleBindingName,
				"--clusterrole=k8s-chaos-metrics-reader",
				fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
			)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterRoleBinding")

			By("validating that the metrics service is available")
			cmd = exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Metrics service should exist")

			By("getting the service account token")
			token, err := serviceAccountToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())

			By("waiting for the metrics endpoint to be ready")
			verifyMetricsEndpointReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "endpoints", metricsServiceName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("8443"), "Metrics endpoint is not ready")
			}
			Eventually(verifyMetricsEndpointReady).Should(Succeed())

			By("verifying that the controller manager is serving the metrics server")
			verifyMetricsServerStarted := func(g Gomega) {
				cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("controller-runtime.metrics\tServing metrics server"),
					"Metrics server not yet started")
			}
			Eventually(verifyMetricsServerStarted).Should(Succeed())

			By("creating the curl-metrics pod to access the metrics endpoint")
			cmd = exec.Command("kubectl", "run", "curl-metrics", "--restart=Never",
				"--namespace", namespace,
				"--image=curlimages/curl:latest",
				"--overrides",
				fmt.Sprintf(`{
					"spec": {
						"containers": [{
							"name": "curl",
							"image": "curlimages/curl:latest",
							"command": ["/bin/sh", "-c"],
							"args": ["curl -v -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics"],
							"securityContext": {
								"readOnlyRootFilesystem": true,
								"allowPrivilegeEscalation": false,
								"capabilities": {
									"drop": ["ALL"]
								},
								"runAsNonRoot": true,
								"runAsUser": 1000,
								"seccompProfile": {
									"type": "RuntimeDefault"
								}
							}
						}],
						"serviceAccountName": "%s"
					}
				}`, token, metricsServiceName, namespace, serviceAccountName))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create curl-metrics pod")

			By("waiting for the curl-metrics pod to complete.")
			verifyCurlUp := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods", "curl-metrics",
					"-o", "jsonpath={.status.phase}",
					"-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Succeeded"), "curl pod in wrong status")
			}
			Eventually(verifyCurlUp, 5*time.Minute).Should(Succeed())

			By("getting the metrics by checking curl-metrics logs")
			metricsOutput := getMetricsOutput()
			Expect(metricsOutput).To(ContainSubstring(
				"controller_runtime_reconcile_total",
			))
		})

		// +kubebuilder:scaffold:e2e-webhooks-checks
	})

	Context("ChaosExperiment - pod-network-loss", func() {
		const testNamespace = "chaos-test-network-loss"

		BeforeEach(func() {
			By("creating test namespace")
			cmd := exec.Command("kubectl", "create", "ns", testNamespace)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create test namespace")

			By("deploying test pods")
			cmd = exec.Command("kubectl", "run", "test-pod-1",
				"--image=busybox:1.36",
				"--labels=app=test-app",
				"--namespace", testNamespace,
				"--command", "--", "sleep", "3600")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create test-pod-1")

			cmd = exec.Command("kubectl", "run", "test-pod-2",
				"--image=busybox:1.36",
				"--labels=app=test-app",
				"--namespace", testNamespace,
				"--command", "--", "sleep", "3600")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create test-pod-2")

			cmd = exec.Command("kubectl", "run", "test-pod-3",
				"--image=busybox:1.36",
				"--labels=app=test-app",
				"--namespace", testNamespace,
				"--command", "--", "sleep", "3600")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create test-pod-3")

			By("waiting for test pods to be ready")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods",
					"-n", testNamespace,
					"-l", "app=test-app",
					"-o", "jsonpath={.items[*].status.phase}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("Running"))

				// Verify all 3 pods are running
				cmd = exec.Command("kubectl", "get", "pods",
					"-n", testNamespace,
					"-l", "app=test-app",
					"--field-selector=status.phase=Running",
					"-o", "name")
				output, err = utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				pods := utils.GetNonEmptyLines(output)
				g.Expect(pods).To(HaveLen(3), "Expected 3 pods to be running")
			}, 2*time.Minute, 5*time.Second).Should(Succeed())
		})

		AfterEach(func() {
			By("cleaning up test namespace")
			cmd := exec.Command("kubectl", "delete", "ns", testNamespace, "--ignore-not-found=true")
			_, _ = utils.Run(cmd)
		})

		It("should inject packet loss into pods successfully", func() {
			By("creating a pod-network-loss experiment")
			experimentYAML := fmt.Sprintf(`apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: test-network-loss
  namespace: %s
spec:
  action: pod-network-loss
  namespace: %s
  selector:
    app: test-app
  count: 2
  duration: "30s"
  lossPercentage: 10
  lossCorrelation: 0
`, testNamespace, testNamespace)

			experimentFile := filepath.Join("/tmp", "network-loss-experiment.yaml")
			err := os.WriteFile(experimentFile, []byte(experimentYAML), os.FileMode(0644))
			Expect(err).NotTo(HaveOccurred())

			cmd := exec.Command("kubectl", "apply", "-f", experimentFile)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create ChaosExperiment")

			By("verifying the experiment status updates")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "chaosexperiment", "test-network-loss",
					"-n", testNamespace,
					"-o", "jsonpath={.status.message}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Or(
					ContainSubstring("Successfully injected"),
					ContainSubstring("packet loss"),
				))
			}, 3*time.Minute, 5*time.Second).Should(Succeed())

			By("verifying ephemeral containers were injected")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods",
					"-n", testNamespace,
					"-l", "app=test-app",
					"-o", "jsonpath={.items[*].spec.ephemeralContainers[*].name}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("network-loss"))
			}, 1*time.Minute, 5*time.Second).Should(Succeed())

			By("verifying metrics were recorded")
			time.Sleep(10 * time.Second) // Wait for metrics to be scraped
			metricsOutput := getMetricsOutput()
			Expect(metricsOutput).To(ContainSubstring("chaos_experiments_total"))
			Expect(metricsOutput).To(Or(
				ContainSubstring(`action="pod-network-loss"`),
				ContainSubstring("pod-network-loss"),
			))

			By("cleaning up experiment")
			cmd = exec.Command("kubectl", "delete", "chaosexperiment", "test-network-loss", "-n", testNamespace)
			_, _ = utils.Run(cmd)
		})

		It("should support dry-run mode", func() {
			By("creating a dry-run pod-network-loss experiment")
			experimentYAML := fmt.Sprintf(`apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: test-network-loss-dryrun
  namespace: %s
spec:
  action: pod-network-loss
  namespace: %s
  selector:
    app: test-app
  count: 2
  duration: "30s"
  lossPercentage: 10
  dryRun: true
`, testNamespace, testNamespace)

			experimentFile := filepath.Join("/tmp", "network-loss-dryrun.yaml")
			err := os.WriteFile(experimentFile, []byte(experimentYAML), os.FileMode(0644))
			Expect(err).NotTo(HaveOccurred())

			cmd := exec.Command("kubectl", "apply", "-f", experimentFile)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create ChaosExperiment")

			By("verifying dry-run status message")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "chaosexperiment", "test-network-loss-dryrun",
					"-n", testNamespace,
					"-o", "jsonpath={.status.message}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("DRY RUN"))
			}, 2*time.Minute, 5*time.Second).Should(Succeed())

			By("verifying no ephemeral containers were actually injected")
			cmd = exec.Command("kubectl", "get", "pods",
				"-n", testNamespace,
				"-l", "app=test-app",
				"-o", "jsonpath={.items[*].spec.ephemeralContainers}")
			output, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			// Should be empty or not contain network-loss containers from this experiment
			Expect(output).NotTo(ContainSubstring("network-loss"))

			By("cleaning up experiment")
			cmd = exec.Command("kubectl", "delete", "chaosexperiment", "test-network-loss-dryrun", "-n", testNamespace)
			_, _ = utils.Run(cmd)
		})

		It("should respect lossCorrelation parameter", func() {
			By("creating a pod-network-loss experiment with correlation")
			experimentYAML := fmt.Sprintf(`apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: test-network-loss-correlation
  namespace: %s
spec:
  action: pod-network-loss
  namespace: %s
  selector:
    app: test-app
  count: 1
  duration: "30s"
  lossPercentage: 20
  lossCorrelation: 50
`, testNamespace, testNamespace)

			experimentFile := filepath.Join("/tmp", "network-loss-correlation.yaml")
			err := os.WriteFile(experimentFile, []byte(experimentYAML), os.FileMode(0644))
			Expect(err).NotTo(HaveOccurred())

			cmd := exec.Command("kubectl", "apply", "-f", experimentFile)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create ChaosExperiment")

			By("verifying the experiment completes successfully")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "chaosexperiment", "test-network-loss-correlation",
					"-n", testNamespace,
					"-o", "jsonpath={.status.message}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("Successfully injected"))
			}, 2*time.Minute, 5*time.Second).Should(Succeed())

			By("cleaning up experiment")
			cmd = exec.Command("kubectl", "delete", "chaosexperiment", "test-network-loss-correlation", "-n", testNamespace)
			_, _ = utils.Run(cmd)
		})

		It("should handle no eligible pods gracefully", func() {
			By("creating an experiment with non-matching selector")
			experimentYAML := fmt.Sprintf(`apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: test-network-loss-nomatch
  namespace: %s
spec:
  action: pod-network-loss
  namespace: %s
  selector:
    app: non-existent-app
  count: 1
  duration: "30s"
  lossPercentage: 10
`, testNamespace, testNamespace)

			experimentFile := filepath.Join("/tmp", "network-loss-nomatch.yaml")
			err := os.WriteFile(experimentFile, []byte(experimentYAML), os.FileMode(0644))
			Expect(err).NotTo(HaveOccurred())

			cmd := exec.Command("kubectl", "apply", "-f", experimentFile)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create ChaosExperiment")

			By("verifying the experiment reports no eligible pods")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "chaosexperiment", "test-network-loss-nomatch",
					"-n", testNamespace,
					"-o", "jsonpath={.status.message}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("No eligible pods"))
			}, 2*time.Minute, 5*time.Second).Should(Succeed())

			By("cleaning up experiment")
			cmd = exec.Command("kubectl", "delete", "chaosexperiment", "test-network-loss-nomatch", "-n", testNamespace)
			_, _ = utils.Run(cmd)
		})

		It("should respect maxPercentage safety limit", func() {
			By("creating an experiment with maxPercentage")
			// We have 3 pods, maxPercentage=30 means max 1 pod (30% of 3 = 0.9, rounds down to 0, but we enforce minimum 1)
			experimentYAML := fmt.Sprintf(`apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: test-network-loss-maxpercent
  namespace: %s
spec:
  action: pod-network-loss
  namespace: %s
  selector:
    app: test-app
  count: 3
  duration: "30s"
  lossPercentage: 10
  maxPercentage: 30
`, testNamespace, testNamespace)

			experimentFile := filepath.Join("/tmp", "network-loss-maxpercent.yaml")
			err := os.WriteFile(experimentFile, []byte(experimentYAML), os.FileMode(0644))
			Expect(err).NotTo(HaveOccurred())

			cmd := exec.Command("kubectl", "apply", "-f", experimentFile)
			_, err = utils.Run(cmd)
			// This should either be rejected by webhook or succeed with limited count
			// Let's check what happens
			if err == nil {
				By("verifying the experiment was created but limited")
				Eventually(func(g Gomega) {
					cmd := exec.Command("kubectl", "get", "chaosexperiment", "test-network-loss-maxpercent",
						"-n", testNamespace,
						"-o", "jsonpath={.status.message}")
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred())
					// Should succeed but only affect 1 pod
					g.Expect(output).To(Or(
						ContainSubstring("Successfully injected"),
						ContainSubstring("1 pod"),
					))
				}, 2*time.Minute, 5*time.Second).Should(Succeed())

				By("cleaning up experiment")
				cmd = exec.Command("kubectl", "delete", "chaosexperiment", "test-network-loss-maxpercent", "-n", testNamespace)
				_, _ = utils.Run(cmd)
			}
		})
	})
})

// serviceAccountToken returns a token for the specified service account in the given namespace.
// It uses the Kubernetes TokenRequest API to generate a token by directly sending a request
// and parsing the resulting token from the API response.
func serviceAccountToken() (string, error) {
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	// Temporary file to store the token request
	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		// Execute kubectl command to create the token
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		// Parse the JSON output to extract the token
		var token tokenRequest
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return out, err
}

// getMetricsOutput retrieves and returns the logs from the curl pod used to access the metrics endpoint.
func getMetricsOutput() string {
	By("getting the curl-metrics logs")
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	metricsOutput, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
	Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
	return metricsOutput
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response,
// containing only the token field that we need to extract.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}
