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
	podNetworkLossNamespace  = "pod-network-loss-test"
	podNetworkLossLabelValue = "pod-network-loss-app"
)

var _ = Describe("Pod Network Loss Chaos Experiments", Ordered, func() {
	BeforeEach(func() {
		createNamespace(podNetworkLossNamespace)
		deployBusyboxPods(podNetworkLossNamespace, podNetworkLossLabelValue, "network-loss-app", 3)
		waitForRunningPods(podNetworkLossNamespace, "app="+podNetworkLossLabelValue, 3)
	})

	AfterEach(func() {
		deleteAllExperiments(podNetworkLossNamespace)
		deleteNamespace(podNetworkLossNamespace)
	})

	Context("Basic Network Loss Tests", func() {
		It("should inject packet loss into selected pods", func() {
			applyExperimentOrFail(fmt.Sprintf(`
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: pod-network-loss-basic
  namespace: %s
spec:
  action: pod-network-loss
  namespace: %s
  selector:
    app: %s
  count: 2
  duration: "30s"
  lossPercentage: 10
`, podNetworkLossNamespace, podNetworkLossNamespace, podNetworkLossLabelValue))

			Eventually(func(g Gomega) {
				message := getExperimentStatusField(podNetworkLossNamespace, "pod-network-loss-basic", "{.status.message}")
				g.Expect(message).To(ContainSubstring("Successfully injected 10% packet loss"))
				g.Expect(message).To(ContainSubstring("2 pod(s)"))
			}, 3*time.Minute, 5*time.Second).Should(Succeed())

			Eventually(func(g Gomega) {
				names := getPodsEphemeralContainerNames(podNetworkLossNamespace, "app="+podNetworkLossLabelValue)
				g.Expect(names).To(ContainSubstring("network-loss"))
			}, 1*time.Minute, 5*time.Second).Should(Succeed())
		})

		It("should support dry-run mode without injecting ephemeral containers", func() {
			applyExperimentOrFail(fmt.Sprintf(`
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: pod-network-loss-dryrun
  namespace: %s
spec:
  action: pod-network-loss
  namespace: %s
  selector:
    app: %s
  count: 2
  duration: "30s"
  lossPercentage: 10
  dryRun: true
`, podNetworkLossNamespace, podNetworkLossNamespace, podNetworkLossLabelValue))

			Eventually(func(g Gomega) {
				message := getExperimentStatusField(podNetworkLossNamespace, "pod-network-loss-dryrun", "{.status.message}")
				g.Expect(message).To(ContainSubstring("DRY RUN"))
				g.Expect(message).To(ContainSubstring("pod-network-loss"))
			}, 2*time.Minute, 5*time.Second).Should(Succeed())

			names := getPodsEphemeralContainerNames(podNetworkLossNamespace, "app="+podNetworkLossLabelValue)
			Expect(names).NotTo(ContainSubstring("network-loss"))
		})

		It("should honor lossCorrelation in status flow", func() {
			applyExperimentOrFail(fmt.Sprintf(`
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: pod-network-loss-correlation
  namespace: %s
spec:
  action: pod-network-loss
  namespace: %s
  selector:
    app: %s
  count: 1
  duration: "30s"
  lossPercentage: 20
  lossCorrelation: 50
`, podNetworkLossNamespace, podNetworkLossNamespace, podNetworkLossLabelValue))

			Eventually(func(g Gomega) {
				message := getExperimentStatusField(podNetworkLossNamespace, "pod-network-loss-correlation", "{.status.message}")
				g.Expect(message).To(ContainSubstring("Successfully injected 20% packet loss"))
				g.Expect(message).To(ContainSubstring("1 pod(s)"))
			}, 2*time.Minute, 5*time.Second).Should(Succeed())
		})
	})

	Context("Safety Features Tests", func() {
		It("should report when no eligible pods match the selector", func() {
			applyExperimentOrFail(fmt.Sprintf(`
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: pod-network-loss-no-match
  namespace: %s
spec:
  action: pod-network-loss
  namespace: %s
  selector:
    app: does-not-exist
  count: 1
  duration: "30s"
  lossPercentage: 10
`, podNetworkLossNamespace, podNetworkLossNamespace))

			Eventually(func(g Gomega) {
				message := getExperimentStatusField(podNetworkLossNamespace, "pod-network-loss-no-match", "{.status.message}")
				g.Expect(message).To(ContainSubstring("No eligible pods"))
			}, 2*time.Minute, 5*time.Second).Should(Succeed())
		})

		It("should reject experiments that exceed maxPercentage when webhook is enabled", func() {
			requireWebhookEnabled()

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(fmt.Sprintf(`
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: pod-network-loss-max-percentage
  namespace: %s
spec:
  action: pod-network-loss
  namespace: %s
  selector:
    app: %s
  count: 3
  duration: "30s"
  lossPercentage: 10
  maxPercentage: 30
`, podNetworkLossNamespace, podNetworkLossNamespace, podNetworkLossLabelValue))

			output, err := utils.Run(cmd)
			Expect(err).To(HaveOccurred())
			Expect(output).To(ContainSubstring("maxPercentage"))
		})
	})

	Context("Validation Tests", func() {
		It("should reject network loss experiment without duration", func() {
			requireWebhookEnabled()

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(fmt.Sprintf(`
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: pod-network-loss-no-duration
  namespace: %s
spec:
  action: pod-network-loss
  namespace: %s
  selector:
    app: %s
  count: 1
  lossPercentage: 50
`, podNetworkLossNamespace, podNetworkLossNamespace, podNetworkLossLabelValue))

			output, err := utils.Run(cmd)
			Expect(err).To(HaveOccurred())
			Expect(output).To(ContainSubstring("duration"))
		})

		It("should reject network loss experiment without lossPercentage", func() {
			requireWebhookEnabled()

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(fmt.Sprintf(`
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: pod-network-loss-no-percentage
  namespace: %s
spec:
  action: pod-network-loss
  namespace: %s
  selector:
    app: %s
  count: 1
  duration: "10s"
`, podNetworkLossNamespace, podNetworkLossNamespace, podNetworkLossLabelValue))

			output, err := utils.Run(cmd)
			Expect(err).To(HaveOccurred())
			Expect(output).To(ContainSubstring("lossPercentage"))
		})
	})

	Context("Advanced Scenarios", func() {
		const advancedNamespace = "pod-network-loss-advanced"
		const controlNamespace = "pod-network-loss-control"

		BeforeEach(func() {
			createNamespace(advancedNamespace)
			createNamespace(controlNamespace)

			createBusyboxPod(advancedNamespace, "target-pod-1", "app=target-app,tier=backend")
			createBusyboxPod(advancedNamespace, "target-pod-2", "app=target-app,tier=backend")
			createBusyboxPod(advancedNamespace, "ignored-pod", "app=target-app,tier=frontend")
			createBusyboxPod(controlNamespace, "control-pod", "app=target-app,tier=backend")

			waitForRunningPods(advancedNamespace, "app=target-app", 3)
			waitForRunningPods(controlNamespace, "app=target-app", 1)
		})

		AfterEach(func() {
			deleteAllExperiments(advancedNamespace)
			deleteNamespace(advancedNamespace)
			deleteNamespace(controlNamespace)
		})

		It("should strictly respect labels and namespaces", func() {
			applyExperimentOrFail(fmt.Sprintf(`
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: pod-network-loss-selectors
  namespace: %s
spec:
  action: pod-network-loss
  namespace: %s
  selector:
    app: target-app
    tier: backend
  count: 2
  duration: "30s"
  lossPercentage: 10
`, advancedNamespace, advancedNamespace))

			Eventually(func(g Gomega) {
				for _, podName := range []string{"target-pod-1", "target-pod-2"} {
					names := getPodEphemeralContainerNames(advancedNamespace, podName)
					g.Expect(names).To(ContainSubstring("network-loss"), podName+" should be targeted")
				}
			}, 2*time.Minute, 5*time.Second).Should(Succeed())

			Expect(getPodEphemeralContainerNames(advancedNamespace, "ignored-pod")).NotTo(ContainSubstring("network-loss"))
			Expect(getPodEphemeralContainerNames(controlNamespace, "control-pod")).NotTo(ContainSubstring("network-loss"))
		})

		It("should handle concurrent experiments targeting different selectors", func() {
			applyExperimentOrFail(fmt.Sprintf(`
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: pod-network-loss-concurrent-backend
  namespace: %s
spec:
  action: pod-network-loss
  namespace: %s
  selector:
    app: target-app
    tier: backend
  count: 1
  duration: "30s"
  lossPercentage: 10
`, advancedNamespace, advancedNamespace))

			applyExperimentOrFail(fmt.Sprintf(`
apiVersion: chaos.gushchin.dev/v1alpha1
kind: ChaosExperiment
metadata:
  name: pod-network-loss-concurrent-frontend
  namespace: %s
spec:
  action: pod-network-loss
  namespace: %s
  selector:
    app: target-app
    tier: frontend
  count: 1
  duration: "30s"
  lossPercentage: 20
`, advancedNamespace, advancedNamespace))

			Eventually(func(g Gomega) {
				backendNames := getPodsEphemeralContainerNames(advancedNamespace, "tier=backend")
				g.Expect(backendNames).To(ContainSubstring("network-loss"))

				frontendNames := getPodsEphemeralContainerNames(advancedNamespace, "tier=frontend")
				g.Expect(frontendNames).To(ContainSubstring("network-loss"))
			}, 2*time.Minute, 5*time.Second).Should(Succeed())
		})
	})
})

func createNamespace(name string) {
	By("creating namespace " + name)
	cmd := exec.Command("kubectl", "create", "namespace", name)
	output, err := utils.Run(cmd)
	if err != nil && !strings.Contains(output, "already exists") {
		Fail(fmt.Sprintf("Failed to create namespace %s: %s", name, output))
	}
}

func deleteNamespace(name string) {
	By("deleting namespace " + name)
	cmd := exec.Command("kubectl", "delete", "namespace", name, "--ignore-not-found=true", "--timeout=60s")
	_, _ = utils.Run(cmd)
}

func deployBusyboxPods(namespace, appLabel, baseName string, replicas int) {
	By("deploying busybox test pods")
	manifest := fmt.Sprintf(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
spec:
  replicas: %d
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
    spec:
      containers:
      - name: busybox
        image: busybox:1.36
        command: ["sleep", "3600"]
`, baseName, namespace, appLabel, replicas, appLabel, appLabel)

	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(manifest)
	_, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to deploy busybox test pods")
}

func createBusyboxPod(namespace, name, labels string) {
	cmd := exec.Command("kubectl", "run", name,
		"--image=busybox:1.36",
		"--labels="+labels,
		"--namespace", namespace,
		"--command", "--", "sleep", "3600")
	output, err := utils.Run(cmd)
	if err != nil && !strings.Contains(output, "already exists") {
		Fail(fmt.Sprintf("Failed to create pod %s: %s", name, output))
	}
}

func waitForRunningPods(namespace, selector string, expectedCount int) {
	By("waiting for running pods in namespace " + namespace)
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "pods",
			"-n", namespace,
			"-l", selector,
			"--field-selector=status.phase=Running",
			"--no-headers")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(utils.GetNonEmptyLines(output)).To(HaveLen(expectedCount))
	}, 2*time.Minute, 2*time.Second).Should(Succeed())
}

func applyExperimentOrFail(manifest string) {
	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(manifest)
	_, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to apply experiment manifest")
}

func deleteAllExperiments(namespace string) {
	cmd := exec.Command("kubectl", "delete", "chaosexperiment", "--all", "-n", namespace, "--ignore-not-found=true")
	_, _ = utils.Run(cmd)
}

func getExperimentStatusField(namespace, name, jsonPath string) string {
	cmd := exec.Command("kubectl", "get", "chaosexperiment", name, "-n", namespace, "-o", "jsonpath="+jsonPath)
	output, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred())
	return output
}

func getPodsEphemeralContainerNames(namespace, selector string) string {
	cmd := exec.Command("kubectl", "get", "pods",
		"-n", namespace,
		"-l", selector,
		"-o", "jsonpath={.items[*].spec.ephemeralContainers[*].name}")
	output, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred())
	return output
}

func getPodEphemeralContainerNames(namespace, podName string) string {
	cmd := exec.Command("kubectl", "get", "pod", podName,
		"-n", namespace,
		"-o", "jsonpath={.spec.ephemeralContainers[*].name}")
	output, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred())
	return output
}
