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
	"fmt"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/neogan74/k8s-chaos/test/utils"
)

const (
	testNamespace          = "memory-stress-test"
	testDeploymentName     = "test-app"
	memoryStressExperiment = "memory-stress-experiment"
)

var _ = Describe("Memory Stress Chaos Experiments", Ordered, func() {
	BeforeAll(func() {
		By("creating test namespace")
		cmd := exec.Command("kubectl", "create", "namespace", testNamespace)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create test namespace")

		By("deploying test application")
		deploymentYAML := fmt.Sprintf(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    app: test-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: test-app
  template:
    metadata:
      labels:
        app: test-app
    spec:
      containers:
      - name: nginx
        image: nginx:alpine
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "2Gi"
            cpu: "500m"
        ports:
        - containerPort: 80
`, testDeploymentName, testNamespace)

		cmd = exec.Command("kubectl", "apply", "-f", "-")
		cmd.Stdin = strings.NewReader(deploymentYAML)
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy test application")

		By("waiting for test pods to be ready")
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "pods",
				"-n", testNamespace,
				"-l", "app=test-app",
				"-o", "jsonpath={.items[*].status.phase}")
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).To(ContainSubstring("Running"))

			// Ensure we have 3 pods running
			cmd = exec.Command("kubectl", "get", "pods",
				"-n", testNamespace,
				"-l", "app=test-app",
				"--field-selector=status.phase=Running",
				"--no-headers")
			output, err = utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			lines := utils.GetNonEmptyLines(output)
			g.Expect(lines).To(HaveLen(3), "Expected 3 running pods")
		}, 2*time.Minute, 2*time.Second).Should(Succeed())
	})

	AfterAll(func() {
		By("deleting test namespace")
		cmd := exec.Command("kubectl", "delete", "namespace", testNamespace, "--timeout=60s")
		_, _ = utils.Run(cmd)
	})

	AfterEach(func() {
		By("cleaning up chaos experiments")
		cmd := exec.Command("kubectl", "delete", "chaosexperiment", "--all", "-n", testNamespace)
		_, _ = utils.Run(cmd)

		// Wait a moment for cleanup
		time.Sleep(2 * time.Second)
	})

	Context("Basic Memory Stress Tests", func() {
		It("should successfully inject memory stress into pods", func() {
			By("creating a memory stress experiment")
			experimentYAML := fmt.Sprintf(`
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: %s
  namespace: %s
spec:
  action: pod-memory-stress
  namespace: %s
  selector:
    app: test-app
  count: 2
  duration: "30s"
  memorySize: "256M"
  memoryWorkers: 1
`, memoryStressExperiment, testNamespace, testNamespace)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(experimentYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create chaos experiment")

			By("verifying the experiment status becomes Running")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "chaosexperiment",
					memoryStressExperiment,
					"-n", testNamespace,
					"-o", "jsonpath={.status.phase}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Experiment should be in Running phase")
			}, 1*time.Minute, 2*time.Second).Should(Succeed())

			By("verifying ephemeral containers are injected")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods",
					"-n", testNamespace,
					"-l", "app=test-app",
					"-o", "jsonpath={.items[*].spec.ephemeralContainers[*].name}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("memory-stress"), "Ephemeral containers should be injected")
			}, 1*time.Minute, 2*time.Second).Should(Succeed())

			By("verifying the experiment completes successfully")
			// Wait for duration + processing time
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "chaosexperiment",
					memoryStressExperiment,
					"-n", testNamespace,
					"-o", "jsonpath={.status.message}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("Successfully"), "Experiment should complete successfully")
			}, 2*time.Minute, 5*time.Second).Should(Succeed())
		})

		It("should handle multiple workers correctly", func() {
			By("creating a memory stress experiment with multiple workers")
			experimentYAML := fmt.Sprintf(`
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: memory-stress-multi-worker
  namespace: %s
spec:
  action: pod-memory-stress
  namespace: %s
  selector:
    app: test-app
  count: 1
  duration: "30s"
  memorySize: "128M"
  memoryWorkers: 4
`, testNamespace, testNamespace)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(experimentYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create chaos experiment")

			By("verifying the experiment runs successfully")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "chaosexperiment",
					"memory-stress-multi-worker",
					"-n", testNamespace,
					"-o", "jsonpath={.status.phase}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Or(Equal("Running"), Equal("Completed")))
			}, 2*time.Minute, 2*time.Second).Should(Succeed())
		})
	})

	Context("Dry-Run Mode Tests", func() {
		It("should preview affected pods without injecting stress in dry-run mode", func() {
			By("creating a dry-run memory stress experiment")
			experimentYAML := fmt.Sprintf(`
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: memory-stress-dryrun
  namespace: %s
spec:
  action: pod-memory-stress
  namespace: %s
  selector:
    app: test-app
  count: 2
  duration: "30s"
  memorySize: "256M"
  memoryWorkers: 1
  dryRun: true
`, testNamespace, testNamespace)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(experimentYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create dry-run experiment")

			By("verifying the experiment status shows dry-run mode")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "chaosexperiment",
					"memory-stress-dryrun",
					"-n", testNamespace,
					"-o", "jsonpath={.status.message}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("DRY-RUN"), "Status should indicate dry-run mode")
			}, 1*time.Minute, 2*time.Second).Should(Succeed())

			By("verifying no ephemeral containers are actually injected")
			Consistently(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods",
					"-n", testNamespace,
					"-l", "app=test-app",
					"-o", "jsonpath={.items[*].spec.ephemeralContainers}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				// Output should be empty or not contain memory-stress containers
				if output != "" {
					g.Expect(output).NotTo(ContainSubstring("memory-stress"))
				}
			}, 10*time.Second, 2*time.Second).Should(Succeed())
		})
	})

	Context("Safety Features Tests", func() {
		It("should respect maxPercentage limits", func() {
			By("creating an experiment with maxPercentage that should succeed")
			experimentYAML := fmt.Sprintf(`
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: memory-stress-maxpercent-pass
  namespace: %s
spec:
  action: pod-memory-stress
  namespace: %s
  selector:
    app: test-app
  count: 1
  duration: "30s"
  memorySize: "256M"
  memoryWorkers: 1
  maxPercentage: 50
`, testNamespace, testNamespace)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(experimentYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create experiment")

			By("verifying the experiment runs successfully")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "chaosexperiment",
					"memory-stress-maxpercent-pass",
					"-n", testNamespace,
					"-o", "jsonpath={.status.phase}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"))
			}, 1*time.Minute, 2*time.Second).Should(Succeed())
		})

		It("should exclude pods with exclusion label", func() {
			By("labeling one pod with exclusion label")
			cmd := exec.Command("kubectl", "get", "pods",
				"-n", testNamespace,
				"-l", "app=test-app",
				"-o", "jsonpath={.items[0].metadata.name}")
			podName, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(podName).NotTo(BeEmpty())

			cmd = exec.Command("kubectl", "label", "pod", podName,
				"-n", testNamespace,
				"chaos.gushchin.dev/exclude=true")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("creating an experiment targeting all pods")
			experimentYAML := fmt.Sprintf(`
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: memory-stress-exclusion
  namespace: %s
spec:
  action: pod-memory-stress
  namespace: %s
  selector:
    app: test-app
  count: 3
  duration: "30s"
  memorySize: "256M"
  memoryWorkers: 1
`, testNamespace, testNamespace)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(experimentYAML)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying only non-excluded pods get stress containers")
			Eventually(func(g Gomega) {
				// Check that the excluded pod does NOT have ephemeral containers
				cmd := exec.Command("kubectl", "get", "pod", podName,
					"-n", testNamespace,
					"-o", "jsonpath={.spec.ephemeralContainers}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				if output != "" {
					g.Expect(output).NotTo(ContainSubstring("memory-stress"))
				}
			}, 1*time.Minute, 2*time.Second).Should(Succeed())

			By("cleaning up exclusion label")
			cmd = exec.Command("kubectl", "label", "pod", podName,
				"-n", testNamespace,
				"chaos.gushchin.dev/exclude-")
			_, _ = utils.Run(cmd)
		})
	})

	Context("Metrics Validation Tests", func() {
		It("should expose memory stress experiment metrics", func() {
			By("creating a memory stress experiment")
			experimentYAML := fmt.Sprintf(`
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: memory-stress-metrics
  namespace: %s
spec:
  action: pod-memory-stress
  namespace: %s
  selector:
    app: test-app
  count: 1
  duration: "20s"
  memorySize: "256M"
  memoryWorkers: 1
`, testNamespace, testNamespace)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(experimentYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for experiment to run")
			time.Sleep(10 * time.Second)

			By("verifying metrics endpoint includes memory stress metrics")
			Eventually(func(g Gomega) {
				metricsOutput := getMetricsOutput()
				g.Expect(metricsOutput).To(ContainSubstring("chaos_experiments_total"),
					"Metrics should include experiment counter")
				g.Expect(metricsOutput).To(ContainSubstring("pod-memory-stress"),
					"Metrics should include pod-memory-stress action")
			}, 2*time.Minute, 5*time.Second).Should(Succeed())
		})
	})

	Context("Validation Tests", func() {
		It("should reject memory stress experiment without duration", func() {
			By("attempting to create experiment without duration")
			experimentYAML := fmt.Sprintf(`
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: memory-stress-no-duration
  namespace: %s
spec:
  action: pod-memory-stress
  namespace: %s
  selector:
    app: test-app
  count: 1
  memorySize: "256M"
  memoryWorkers: 1
`, testNamespace, testNamespace)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(experimentYAML)
			output, err := utils.Run(cmd)

			// Should be rejected by webhook
			Expect(err).To(HaveOccurred(), "Should reject experiment without duration")
			Expect(output).To(ContainSubstring("duration"), "Error should mention duration requirement")
		})

		It("should reject memory stress experiment without memorySize", func() {
			By("attempting to create experiment without memorySize")
			experimentYAML := fmt.Sprintf(`
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: memory-stress-no-size
  namespace: %s
spec:
  action: pod-memory-stress
  namespace: %s
  selector:
    app: test-app
  count: 1
  duration: "30s"
  memoryWorkers: 1
`, testNamespace, testNamespace)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(experimentYAML)
			output, err := utils.Run(cmd)

			// Should be rejected by webhook
			Expect(err).To(HaveOccurred(), "Should reject experiment without memorySize")
			Expect(output).To(ContainSubstring("memorySize"), "Error should mention memorySize requirement")
		})

		It("should reject invalid memorySize format", func() {
			By("attempting to create experiment with invalid memorySize")
			experimentYAML := fmt.Sprintf(`
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: memory-stress-invalid-size
  namespace: %s
spec:
  action: pod-memory-stress
  namespace: %s
  selector:
    app: test-app
  count: 1
  duration: "30s"
  memorySize: "256"
  memoryWorkers: 1
`, testNamespace, testNamespace)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(experimentYAML)
			output, err := utils.Run(cmd)

			// Should be rejected by OpenAPI validation
			Expect(err).To(HaveOccurred(), "Should reject invalid memorySize format")
			Expect(output).To(Or(
				ContainSubstring("memorySize"),
				ContainSubstring("pattern"),
			), "Error should mention memorySize validation")
		})
	})
})
