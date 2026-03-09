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
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/neogan74/k8s-chaos/test/utils"
)

const (
	podRestartNamespace      = "pod-restart-test"
	podRestartDeploymentName = "restart-test-app"
	podRestartExperiment     = "pod-restart-experiment"
)

var _ = Describe("Pod Restart Chaos Experiments", Ordered, func() {
	BeforeAll(func() {
		By("creating test namespace")
		cmd := exec.Command("kubectl", "create", "namespace", podRestartNamespace)
		output, err := utils.Run(cmd)
		if err != nil && !strings.Contains(output, "already exists") {
			Fail(fmt.Sprintf("Failed to create test namespace: %s", output))
		}

		By("deploying test application")
		deploymentYAML := fmt.Sprintf(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    app: restart-test-app
spec:
  replicas: 2
  selector:
    matchLabels:
      app: restart-test-app
  template:
    metadata:
      labels:
        app: restart-test-app
    spec:
      containers:
      - name: nginx
        image: nginx:alpine
        ports:
        - containerPort: 80
`, podRestartDeploymentName, podRestartNamespace)

		cmd = exec.Command("kubectl", "apply", "-f", "-")
		cmd.Stdin = strings.NewReader(deploymentYAML)
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy test application")

		By("waiting for test pods to be ready")
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "pods",
				"-n", podRestartNamespace,
				"-l", "app=restart-test-app",
				"--field-selector=status.phase=Running",
				"--no-headers")
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			lines := utils.GetNonEmptyLines(output)
			g.Expect(lines).To(HaveLen(2), "Expected 2 running pods")
		}, 2*time.Minute, 2*time.Second).Should(Succeed())
	})

	AfterAll(func() {
		By("deleting test namespace")
		cmd := exec.Command("kubectl", "delete", "namespace", podRestartNamespace, "--timeout=60s")
		_, _ = utils.Run(cmd)
	})

	AfterEach(func() {
		By("cleaning up chaos experiments")
		cmd := exec.Command("kubectl", "delete", "chaosexperiment", "--all", "-n", podRestartNamespace)
		_, _ = utils.Run(cmd)

		time.Sleep(2 * time.Second)
	})

	Context("Basic Pod Restart Tests", func() {
		It("should increase restart count of target pods", func() {
			By("getting initial restart counts")
			// Get restart counts for all pods
			cmd := exec.Command("kubectl", "get", "pods",
				"-n", podRestartNamespace,
				"-l", "app=restart-test-app",
				"--sort-by=.metadata.name", 
				"-o", "jsonpath={range .items[*]}{.status.containerStatuses[0].restartCount}{' '}{end}")
			initialRestartsStr, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			
			// Parse initial restart counts
			initialRestarts := strings.Fields(initialRestartsStr)
			Expect(initialRestarts).To(HaveLen(2))
			
			By("creating a pod restart experiment")
			experimentYAML := fmt.Sprintf(`
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: %s
  namespace: %s
spec:
  action: pod-restart
  namespace: %s
  selector:
    app: restart-test-app
  count: 1
`, podRestartExperiment, podRestartNamespace, podRestartNamespace)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(experimentYAML)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create chaos experiment")

			By("verifying the experiment completes successfully")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "chaosexperiment",
					podRestartExperiment,
					"-n", podRestartNamespace,
					"-o", "jsonpath={.status.message}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("Successfully restarted"), "Experiment should complete successfully")
			}, 2*time.Minute, 5*time.Second).Should(Succeed())

			By("verifying restart count increased for at least one pod")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods",
					"-n", podRestartNamespace,
					"-l", "app=restart-test-app",
					"--sort-by=.metadata.name",
					"-o", "jsonpath={range .items[*]}{.status.containerStatuses[0].restartCount}{' '}{end}")
				currentRestartsStr, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				
				currentRestarts := strings.Fields(currentRestartsStr)
				g.Expect(currentRestarts).To(HaveLen(2))
				
				// Check if any pod has increased restart count
				increased := false
				for i, countStr := range currentRestarts {
					initial, _ := strconv.Atoi(initialRestarts[i])
					current, _ := strconv.Atoi(countStr)
					if current > initial {
						increased = true
						break
					}
				}
				g.Expect(increased).To(BeTrue(), "At least one pod should have increased restart count")
			}, 1*time.Minute, 2*time.Second).Should(Succeed())
		})
	})
})
