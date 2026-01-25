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
	diskFillNamespace      = "disk-fill-test"
	diskFillDeploymentName = "disk-fill-app"
	diskFillExperiment     = "disk-fill-experiment"
)

var _ = Describe("Disk Fill Chaos Experiments", Ordered, func() {
	BeforeAll(func() {
		By("creating test namespace")
		cmd := exec.Command("kubectl", "create", "namespace", diskFillNamespace)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create test namespace")

		By("deploying test application with writable volume")
		deploymentYAML := fmt.Sprintf(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    app: disk-fill-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: disk-fill-app
  template:
    metadata:
      labels:
        app: disk-fill-app
    spec:
      containers:
      - name: nginx
        image: nginx:alpine
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        volumeMounts:
        - name: data
          mountPath: /data
      volumes:
      - name: data
        emptyDir: {}
`, diskFillDeploymentName, diskFillNamespace)

		cmd = exec.Command("kubectl", "apply", "-f", "-")
		cmd.Stdin = strings.NewReader(deploymentYAML)
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy test application")

		By("waiting for test pods to be ready")
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "pods",
				"-n", diskFillNamespace,
				"-l", "app=disk-fill-app",
				"--field-selector=status.phase=Running",
				"--no-headers")
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			lines := utils.GetNonEmptyLines(output)
			g.Expect(lines).To(HaveLen(3), "Expected 3 running pods")
		}, 2*time.Minute, 2*time.Second).Should(Succeed())
	})

	AfterAll(func() {
		By("deleting test namespace")
		cmd := exec.Command("kubectl", "delete", "namespace", diskFillNamespace, "--timeout=60s")
		_, _ = utils.Run(cmd)
	})

	AfterEach(func() {
		By("cleaning up chaos experiments")
		cmd := exec.Command("kubectl", "delete", "chaosexperiment", "--all", "-n", diskFillNamespace)
		_, _ = utils.Run(cmd)

		time.Sleep(2 * time.Second)
	})

	Context("Basic Disk Fill Tests", func() {
		It("should successfully inject disk fill into pods", func() {
			By("creating a disk fill experiment")
			experimentYAML := fmt.Sprintf(`
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: %s
  namespace: %s
spec:
  action: pod-disk-fill
  namespace: %s
  selector:
    app: disk-fill-app
  count: 1
  duration: "20s"
  fillPercentage: 80
  targetPath: "/data"
`, diskFillExperiment, diskFillNamespace, diskFillNamespace)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(experimentYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create chaos experiment")

			By("verifying the experiment status becomes Running")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "chaosexperiment",
					diskFillExperiment,
					"-n", diskFillNamespace,
					"-o", "jsonpath={.status.phase}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Experiment should be in Running phase")
			}, 1*time.Minute, 2*time.Second).Should(Succeed())

			By("verifying ephemeral containers are injected")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods",
					"-n", diskFillNamespace,
					"-l", "app=disk-fill-app",
					"-o", "jsonpath={.items[*].spec.ephemeralContainers[*].name}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("disk-fill"), "Ephemeral containers should be injected")
			}, 1*time.Minute, 2*time.Second).Should(Succeed())

			By("verifying the experiment completes successfully")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "chaosexperiment",
					diskFillExperiment,
					"-n", diskFillNamespace,
					"-o", "jsonpath={.status.message}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("Successfully filled disk"))
			}, 2*time.Minute, 5*time.Second).Should(Succeed())
		})
	})

	Context("Validation Tests", func() {
		It("should reject disk fill experiment without duration", func() {
			requireWebhookEnabled()

			By("attempting to create experiment without duration")
			experimentYAML := fmt.Sprintf(`
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: disk-fill-no-duration
  namespace: %s
spec:
  action: pod-disk-fill
  namespace: %s
  selector:
    app: disk-fill-app
  count: 1
  fillPercentage: 80
  targetPath: "/data"
`, diskFillNamespace, diskFillNamespace)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(experimentYAML)
			output, err := utils.Run(cmd)

			Expect(err).To(HaveOccurred(), "Should reject experiment without duration")
			Expect(output).To(ContainSubstring("duration"), "Error should mention duration requirement")
		})
	})
})
