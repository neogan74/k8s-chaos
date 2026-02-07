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
	networkPartitionNamespace      = "network-partition-test"
	networkPartitionDeploymentName = "network-partition-app"
	networkPartitionExperiment     = "network-partition-experiment"
)

var _ = Describe("Network Partition Chaos Experiments", Ordered, func() {
	BeforeAll(func() {
		By("creating test namespace")
		cmd := exec.Command("kubectl", "create", "namespace", networkPartitionNamespace)
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
    app: network-partition-app
spec:
  replicas: 2
  selector:
    matchLabels:
      app: network-partition-app
  template:
    metadata:
      labels:
        app: network-partition-app
    spec:
      containers:
      - name: nginx
        image: nginx:alpine
        ports:
        - containerPort: 80
`, networkPartitionDeploymentName, networkPartitionNamespace)

		cmd = exec.Command("kubectl", "apply", "-f", "-")
		cmd.Stdin = strings.NewReader(deploymentYAML)
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy test application")

		By("waiting for test pods to be ready")
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "pods",
				"-n", networkPartitionNamespace,
				"-l", "app=network-partition-app",
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
		cmd := exec.Command("kubectl", "delete", "namespace", networkPartitionNamespace, "--timeout=60s")
		_, _ = utils.Run(cmd)
	})

	AfterEach(func() {
		By("cleaning up chaos experiments")
		cmd := exec.Command("kubectl", "delete", "chaosexperiment", "--all", "-n", networkPartitionNamespace)
		_, _ = utils.Run(cmd)

		time.Sleep(2 * time.Second)
	})

	Context("Basic Network Partition Tests", func() {
		It("should successfully inject network partition into pods", func() {
			By("creating a network partition experiment")
			experimentYAML := fmt.Sprintf(`
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: %s
  namespace: %s
spec:
  action: network-partition
  namespace: %s
  selector:
    app: network-partition-app
  count: 1
  duration: "10s"
  direction: "both"
`, networkPartitionExperiment, networkPartitionNamespace, networkPartitionNamespace)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(experimentYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create chaos experiment")

			By("verifying the experiment status becomes Running")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "chaosexperiment",
					networkPartitionExperiment,
					"-n", networkPartitionNamespace,
					"-o", "jsonpath={.status.phase}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Experiment should be in Running phase")
			}, 1*time.Minute, 2*time.Second).Should(Succeed())

			By("verifying ephemeral containers are injected")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods",
					"-n", networkPartitionNamespace,
					"-l", "app=network-partition-app",
					"-o", "jsonpath={.items[*].spec.ephemeralContainers[*].name}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("network-partition"), "Ephemeral containers should be injected")
			}, 1*time.Minute, 2*time.Second).Should(Succeed())

			By("verifying the experiment completes successfully")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "chaosexperiment",
					networkPartitionExperiment,
					"-n", networkPartitionNamespace,
					"-o", "jsonpath={.status.message}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("Successfully injected network partition"))
			}, 2*time.Minute, 5*time.Second).Should(Succeed())
		})
	})

	Context("Validation Tests", func() {
		It("should reject network partition experiment without duration", func() {
			requireWebhookEnabled()

			By("attempting to create experiment without duration")
			experimentYAML := fmt.Sprintf(`
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: network-partition-no-duration
  namespace: %s
spec:
  action: network-partition
  namespace: %s
  selector:
    app: network-partition-app
  count: 1
  direction: "both"
`, networkPartitionNamespace, networkPartitionNamespace)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(experimentYAML)
			output, err := utils.Run(cmd)

			Expect(err).To(HaveOccurred(), "Should reject experiment without duration")
			Expect(output).To(ContainSubstring("duration"), "Error should mention duration requirement")
		})
	})
})
