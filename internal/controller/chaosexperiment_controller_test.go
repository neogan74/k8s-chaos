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
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
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
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
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
			}, timeout, interval).Should(ContainSubstring("No pods found"))
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
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Minute))

			By("Verifying status message indicates not implemented")
			exp := &chaosv1alpha1.ChaosExperiment{}
			Eventually(func() string {
				_ = k8sClient.Get(ctx, typeNamespacedName, exp)
				return exp.Status.Message
			}, timeout, interval).Should(ContainSubstring("not yet implemented"))
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
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			result, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Minute))
		})
	})
})
