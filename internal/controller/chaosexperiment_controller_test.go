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

package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	chaosv1alpha1 "github.com/neogan74/k8s-chaos/api/v1alpha1"
)

var _ = Describe("ChaosExperiment Controller", func() {
	Context("When reconciling a ChaosExperiment", func() {
		const (
			experimentName      = "test-chaos"
			experimentNamespace = "default"
			targetNamespace     = "test-ns"
			timeout             = time.Second * 10
			duration            = time.Second * 2
			interval            = time.Millisecond * 250
		)

		ctx := context.Background()
		typeNamespacedName := types.NamespacedName{
			Name:      experimentName,
			Namespace: experimentNamespace,
		}

		BeforeEach(func() {
			// Create target namespace
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: targetNamespace,
				},
			}
			_ = k8sClient.Create(ctx, ns)
		})

		AfterEach(func() {
			// Cleanup
			experiment := &chaosv1alpha1.ChaosExperiment{}
			_ = k8sClient.Get(ctx, typeNamespacedName, experiment)
			_ = k8sClient.Delete(ctx, experiment)

			// Clean up test pods
			podList := &corev1.PodList{}
			_ = k8sClient.List(ctx, podList, client.InNamespace(targetNamespace))
			for _, pod := range podList.Items {
				_ = k8sClient.Delete(ctx, &pod)
			}

			// Delete namespace
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: targetNamespace,
				},
			}
			_ = k8sClient.Delete(ctx, ns)
		})

		It("Should handle pod-kill action successfully", func() {
			By("Creating test pods")
			for i := 0; i < 3; i++ {
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod-" + string(rune('0'+i)),
						Namespace: targetNamespace,
						Labels: map[string]string{
							"app": "test",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "test",
								Image: "busybox",
								Command: []string{
									"sh", "-c", "sleep 3600",
								},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, pod)).Should(Succeed())
			}

			By("Creating a ChaosExperiment with pod-kill action")
			experiment := &chaosv1alpha1.ChaosExperiment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      experimentName,
					Namespace: experimentNamespace,
				},
				Spec: chaosv1alpha1.ChaosExperimentSpec{
					Action:    "pod-kill",
					Namespace: targetNamespace,
					Selector: map[string]string{
						"app": "test",
					},
					Count: 2,
				},
			}
			Expect(k8sClient.Create(ctx, experiment)).Should(Succeed())

			By("Checking if the controller reconciles and kills pods")
			reconciler := &ChaosExperimentReconciler{
				Client:        k8sClient,
				Scheme:        k8sClient.Scheme(),
				Recorder:      record.NewFakeRecorder(100),
				HistoryConfig: DefaultHistoryConfig(),
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying status is updated")
			Eventually(func() bool {
				exp := &chaosv1alpha1.ChaosExperiment{}
				if err := k8sClient.Get(ctx, typeNamespacedName, exp); err != nil {
					return false
				}
				return exp.Status.LastRunTime != nil && exp.Status.Message != ""
			}, timeout, interval).Should(BeTrue())

			By("Verifying pods were deleted")
			Eventually(func() int {
				podList := &corev1.PodList{}
				_ = k8sClient.List(ctx, podList, client.InNamespace(targetNamespace))
				// Account for pods being recreated or still terminating
				return len(podList.Items)
			}, timeout, interval).Should(BeNumerically("<=", 1))
		})

		It("Should handle no matching pods gracefully", func() {
			By("Creating a ChaosExperiment with selector that matches no pods")
			experiment := &chaosv1alpha1.ChaosExperiment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      experimentName,
					Namespace: experimentNamespace,
				},
				Spec: chaosv1alpha1.ChaosExperimentSpec{
					Action:    "pod-kill",
					Namespace: targetNamespace,
					Selector: map[string]string{
						"app": "nonexistent",
					},
					Count: 1,
				},
			}
			Expect(k8sClient.Create(ctx, experiment)).Should(Succeed())

			By("Reconciling the experiment")
			reconciler := &ChaosExperimentReconciler{
				Client:        k8sClient,
				Scheme:        k8sClient.Scheme(),
				Recorder:      record.NewFakeRecorder(100),
				HistoryConfig: DefaultHistoryConfig(),
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Minute))

			By("Verifying message in status")
			exp := &chaosv1alpha1.ChaosExperiment{}
			Eventually(func() string {
				_ = k8sClient.Get(ctx, typeNamespacedName, exp)
				return exp.Status.Message
			}, timeout, interval).Should(ContainSubstring("No eligible pods found"))
		})

		It("Should handle pod-delay action", func() {
			By("Creating a ChaosExperiment with pod-delay action")
			experiment := &chaosv1alpha1.ChaosExperiment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      experimentName,
					Namespace: experimentNamespace,
				},
				Spec: chaosv1alpha1.ChaosExperimentSpec{
					Action:    "pod-delay",
					Namespace: targetNamespace,
					Selector: map[string]string{
						"app": "test",
					},
					Count:    1,
					Duration: "10s",
				},
			}
			Expect(k8sClient.Create(ctx, experiment)).Should(Succeed())

			By("Reconciling the experiment")
			reconciler := &ChaosExperimentReconciler{
				Client:        k8sClient,
				Scheme:        k8sClient.Scheme(),
				Recorder:      record.NewFakeRecorder(100),
				HistoryConfig: DefaultHistoryConfig(),
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Minute))

			By("Verifying status message (no pods to affect in this test)")
			exp := &chaosv1alpha1.ChaosExperiment{}
			Eventually(func() string {
				_ = k8sClient.Get(ctx, typeNamespacedName, exp)
				return exp.Status.Message
			}, timeout, interval).Should(ContainSubstring("No eligible pods found"))
		})

		It("Should requeue after specified time", func() {
			By("Creating a valid ChaosExperiment")
			experiment := &chaosv1alpha1.ChaosExperiment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      experimentName,
					Namespace: experimentNamespace,
				},
				Spec: chaosv1alpha1.ChaosExperimentSpec{
					Action:    "pod-kill",
					Namespace: targetNamespace,
					Selector: map[string]string{
						"app": "test",
					},
					Count: 1,
				},
			}
			Expect(k8sClient.Create(ctx, experiment)).Should(Succeed())

			By("Reconciling and checking requeue time")
			reconciler := &ChaosExperimentReconciler{
				Client:        k8sClient,
				Scheme:        k8sClient.Scheme(),
				Recorder:      record.NewFakeRecorder(100),
				HistoryConfig: DefaultHistoryConfig(),
			}

			result, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Minute))
		})

		It("Should handle paused experiments", func() {
			By("Creating a paused ChaosExperiment")
			experiment := &chaosv1alpha1.ChaosExperiment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      experimentName,
					Namespace: experimentNamespace,
				},
				Spec: chaosv1alpha1.ChaosExperimentSpec{
					Action:    "pod-kill",
					Namespace: targetNamespace,
					Selector: map[string]string{
						"app": "test",
					},
					Count:  1,
					Paused: true,
				},
			}
			Expect(k8sClient.Create(ctx, experiment)).Should(Succeed())

			By("Reconciling the paused experiment")
			reconciler := &ChaosExperimentReconciler{
				Client:        k8sClient,
				Scheme:        k8sClient.Scheme(),
				Recorder:      record.NewFakeRecorder(100),
				HistoryConfig: DefaultHistoryConfig(),
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying status is Paused")
			exp := &chaosv1alpha1.ChaosExperiment{}
			Eventually(func() string {
				_ = k8sClient.Get(ctx, typeNamespacedName, exp)
				return exp.Status.Phase
			}, timeout, interval).Should(Equal("Paused"))
			Expect(exp.Status.Message).To(Equal("Experiment is paused"))

			By("Unpausing the experiment")
			Eventually(func() error {
				if err := k8sClient.Get(ctx, typeNamespacedName, exp); err != nil {
					return err
				}
				exp.Spec.Paused = false
				return k8sClient.Update(ctx, exp)
			}, timeout, interval).Should(Succeed())

			By("Reconciling the unpaused experiment")
			// We trigger reconcile again to process the change
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying status is no longer Paused")
			Eventually(func() string {
				_ = k8sClient.Get(ctx, typeNamespacedName, exp)
				return exp.Status.Phase
			}, timeout, interval).ShouldNot(Equal("Paused"))
		})

		It("Should handle network-partition action", func() {
			// Use a unique namespace for this test to avoid race conditions with BeforeEach/AfterEach cleanup
			uniqueNamespace := "test-ns-" + generateShortUID()

			// Create namespace via unstructured or Typed Client.
			// We have k8sClient.
			realNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: uniqueNamespace}}
			Expect(k8sClient.Create(ctx, realNs)).Should(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, realNs)
			}()

			By("Creating a valid ChaosExperiment with network-partition action")
			experiment := &chaosv1alpha1.ChaosExperiment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      experimentName + "-network", // unique name too
					Namespace: experimentNamespace,
				},
				Spec: chaosv1alpha1.ChaosExperimentSpec{
					Action:    "network-partition",
					Namespace: uniqueNamespace,
					Selector: map[string]string{
						"app": "test",
					},
					Count:     1,
					Duration:  "5s",
					Direction: "ingress",
				},
			}
			Expect(k8sClient.Create(ctx, experiment)).Should(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, experiment)
			}()

			By("Creating a target Pod")
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod-network",
					Namespace: uniqueNamespace,
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test",
							Image: "busybox",
							Command: []string{
								"sh", "-c", "sleep 3600",
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, pod)).Should(Succeed())

			// We need to update pod status to Running so getEligiblePods picks it up
			By("Updating Pod status to Running")
			// Need to get the pod again or ensure we have the latest version before update status?
			// Create returns the updated object.
			pod.Status.Phase = corev1.PodRunning
			pod.Status.Conditions = []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			}
			// Use Status().Update() to update the status subresource
			Expect(k8sClient.Status().Update(ctx, pod)).Should(Succeed())

			By("Reconciling the experiment")
			// Recreate reconciler or reuse? Reuse is fine, but we need to create a new Request
			reconciler := &ChaosExperimentReconciler{
				Client:        k8sClient,
				Scheme:        k8sClient.Scheme(),
				Recorder:      record.NewFakeRecorder(100),
				HistoryConfig: DefaultHistoryConfig(),
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: experiment.Name, Namespace: experiment.Namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying ephemeral container was injected")
			updatedPod := &corev1.Pod{}
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: pod.Name, Namespace: uniqueNamespace}, updatedPod); err != nil {
					return false
				}
				for _, ec := range updatedPod.Spec.EphemeralContainers {
					if len(ec.Name) > 0 { // Check if any ephemeral container exists
						return true
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())

			By("Verifying experiment status")
			exp := &chaosv1alpha1.ChaosExperiment{}
			Eventually(func() string {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: experiment.Name, Namespace: experiment.Namespace}, exp)
				return exp.Status.Message
			}, timeout, interval).Should(ContainSubstring("Successfully injected network partition"))
		})
	})
})
