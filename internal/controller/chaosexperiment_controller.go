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
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	chaosv1alpha1 "github.com/neogan74/k8s-chaos/api/v1alpha1"
	chaosmetrics "github.com/neogan74/k8s-chaos/internal/metrics"
)

const (
	// Status constants for experiment execution
	statusSuccess = "success"
	statusFailure = "failure"

	// Phase constants for experiment lifecycle
	phaseRunning   = "Running"
	phaseCompleted = "Completed"
	phasePending   = "Pending"
	phaseFailed    = "Failed"

	// Default retry configuration
	defaultMaxRetries   = 3
	defaultRetryDelay   = 30 * time.Second
	defaultRetryBackoff = "exponential"
)

// ChaosExperimentReconciler reconciles a ChaosExperiment object
type ChaosExperimentReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Config    *rest.Config
	Clientset *kubernetes.Clientset
}

// +kubebuilder:rbac:groups=chaos.gushchin.dev,resources=chaosexperiments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chaos.gushchin.dev,resources=chaosexperiments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chaos.gushchin.dev,resources=chaosexperiments/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;delete;patch
// +kubebuilder:rbac:groups="",resources=pods/exec,verbs=create
// +kubebuilder:rbac:groups="",resources=pods/ephemeralcontainers,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups="",resources=pods/eviction,verbs=create
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ChaosExperiment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *ChaosExperimentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	var exp chaosv1alpha1.ChaosExperiment
	if err := r.Get(ctx, req.NamespacedName, &exp); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if exp.Spec.Action == "" {
		log.Error(nil, "Action not specified")
		exp.Status.Message = "Error: Action not specified"
		_ = r.Status().Update(ctx, &exp)
		return ctrl.Result{}, nil
	}

	// Check experiment lifecycle (duration-based auto-stop)
	shouldContinue, err := r.checkExperimentLifecycle(ctx, &exp)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !shouldContinue {
		// Experiment has completed its duration or is already completed
		return ctrl.Result{}, nil
	}

	switch exp.Spec.Action {
	case "pod-kill":
		return r.handlePodKill(ctx, &exp)
	case "pod-delay":
		return r.handlePodDelay(ctx, &exp)
	case "node-drain":
		return r.handleNodeDrain(ctx, &exp)
	case "pod-cpu-stress":
		return r.handlePodCPUStress(ctx, &exp)
	default:
		log.Info("Unsupported action", "action", exp.Spec.Action)
		exp.Status.Message = "Error: Unsupported action: " + exp.Spec.Action
		_ = r.Status().Update(ctx, &exp)
		return ctrl.Result{}, nil
	}
}

func (r *ChaosExperimentReconciler) handlePodKill(ctx context.Context, exp *chaosv1alpha1.ChaosExperiment) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	startTime := time.Now()

	// Track active experiments
	chaosmetrics.ActiveExperiments.WithLabelValues("pod-kill").Inc()
	defer chaosmetrics.ActiveExperiments.WithLabelValues("pod-kill").Dec()

	// Validate namespace
	if exp.Spec.Namespace == "" {
		return r.handleExperimentFailure(ctx, exp, "Namespace not specified")
	}

	// Choose Pods by selector
	podList := &corev1.PodList{}
	selector := labels.SelectorFromSet(exp.Spec.Selector)
	if err := r.List(ctx, podList, client.InNamespace(exp.Spec.Namespace),
		client.MatchingLabelsSelector{Selector: selector}); err != nil {
		log.Error(err, "Failed to list pods")
		exp.Status.Message = "Error: Failed to list pods"
		_ = r.Status().Update(ctx, exp)
		return ctrl.Result{}, err
	}

	if len(podList.Items) == 0 {
		log.Info("No pods found for selector", "selector", exp.Spec.Selector)
		exp.Status.Message = "No pods found matching selector"
		_ = r.Status().Update(ctx, exp)
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	// Shuffle the list of pods
	rand.Shuffle(len(podList.Items), func(i, j int) {
		podList.Items[i], podList.Items[j] = podList.Items[j], podList.Items[i]
	})

	// Delete the specified number of pods
	killCount := exp.Spec.Count
	if killCount <= 0 {
		killCount = 1 // Default to 1 if not specified or invalid
	}
	if killCount > len(podList.Items) {
		killCount = len(podList.Items)
	}

	killedPods := []string{}
	for i := 0; i < killCount; i++ {
		pod := podList.Items[i]
		log.Info("Deleting pod", "pod", pod.Name, "namespace", pod.Namespace)
		if err := r.Delete(ctx, &pod); err != nil {
			log.Error(err, "Failed to delete pod", "pod", pod.Name)
		} else {
			killedPods = append(killedPods, pod.Name)
		}
	}

	// Update status
	now := metav1.Now()
	exp.Status.LastRunTime = &now
	status := statusSuccess
	if len(killedPods) > 0 {
		exp.Status.Message = fmt.Sprintf("Successfully killed %d pod(s)", len(killedPods))
	} else {
		exp.Status.Message = "Failed to kill any pods"
		status = statusFailure
	}
	if err := r.Status().Update(ctx, exp); err != nil {
		log.Error(err, "Failed to update ChaosExperiment status")
		return ctrl.Result{}, err
	}

	// Record metrics
	duration := time.Since(startTime).Seconds()
	chaosmetrics.ExperimentsTotal.WithLabelValues("pod-kill", exp.Spec.Namespace, status).Inc()
	chaosmetrics.ExperimentDuration.WithLabelValues("pod-kill", exp.Spec.Namespace).Observe(duration)
	chaosmetrics.ResourcesAffected.WithLabelValues("pod-kill", exp.Spec.Namespace, exp.Name).Set(float64(len(killedPods)))

	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

func (r *ChaosExperimentReconciler) handlePodDelay(ctx context.Context, exp *chaosv1alpha1.ChaosExperiment) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	startTime := time.Now()

	// Track active experiments
	chaosmetrics.ActiveExperiments.WithLabelValues("pod-delay").Inc()
	defer chaosmetrics.ActiveExperiments.WithLabelValues("pod-delay").Dec()

	// Validate namespace
	if exp.Spec.Namespace == "" {
		log.Error(nil, "Namespace not specified")
		exp.Status.Message = "Error: Namespace not specified"
		_ = r.Status().Update(ctx, exp)
		return ctrl.Result{}, nil
	}

	// Validate duration is specified for pod-delay
	if exp.Spec.Duration == "" {
		log.Error(nil, "Duration not specified for pod-delay action")
		exp.Status.Message = "Error: Duration is required for pod-delay action"
		_ = r.Status().Update(ctx, exp)
		return ctrl.Result{}, nil
	}

	// Parse duration
	delayMs, err := r.parseDurationToMs(exp.Spec.Duration)
	if err != nil {
		log.Error(err, "Failed to parse duration", "duration", exp.Spec.Duration)
		exp.Status.Message = fmt.Sprintf("Error: Invalid duration format: %s", exp.Spec.Duration)
		_ = r.Status().Update(ctx, exp)
		return ctrl.Result{}, nil
	}

	// List pods by selector
	podList := &corev1.PodList{}
	selector := labels.SelectorFromSet(exp.Spec.Selector)
	if err := r.List(ctx, podList, client.InNamespace(exp.Spec.Namespace),
		client.MatchingLabelsSelector{Selector: selector}); err != nil {
		log.Error(err, "Failed to list pods")
		exp.Status.Message = "Error: Failed to list pods"
		_ = r.Status().Update(ctx, exp)
		return ctrl.Result{}, err
	}

	if len(podList.Items) == 0 {
		log.Info("No pods found for selector", "selector", exp.Spec.Selector)
		exp.Status.Message = "No pods found matching selector"
		_ = r.Status().Update(ctx, exp)
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	// Shuffle the list of pods
	rand.Shuffle(len(podList.Items), func(i, j int) {
		podList.Items[i], podList.Items[j] = podList.Items[j], podList.Items[i]
	})

	// Determine how many pods to affect
	affectCount := exp.Spec.Count
	if affectCount <= 0 {
		affectCount = 1 // Default to 1 if not specified or invalid
	}
	if affectCount > len(podList.Items) {
		affectCount = len(podList.Items)
	}

	// Apply network delay to selected pods
	affectedPods := []string{}
	for i := 0; i < affectCount; i++ {
		pod := podList.Items[i]
		log.Info("Adding network delay to pod", "pod", pod.Name, "namespace", pod.Namespace, "delay", delayMs)

		// Apply delay using tc (traffic control)
		if err := r.applyNetworkDelay(ctx, &pod, delayMs); err != nil {
			log.Error(err, "Failed to apply network delay", "pod", pod.Name)
		} else {
			affectedPods = append(affectedPods, pod.Name)
		}
	}

	// Update status
	now := metav1.Now()
	exp.Status.LastRunTime = &now
	status := statusSuccess
	if len(affectedPods) > 0 {
		exp.Status.Message = fmt.Sprintf("Successfully added %dms delay to %d pod(s)", delayMs, len(affectedPods))
	} else {
		exp.Status.Message = "Failed to add delay to any pods"
		status = statusFailure
	}
	if err := r.Status().Update(ctx, exp); err != nil {
		log.Error(err, "Failed to update ChaosExperiment status")
		return ctrl.Result{}, err
	}

	// Record metrics
	duration := time.Since(startTime).Seconds()
	chaosmetrics.ExperimentsTotal.WithLabelValues("pod-delay", exp.Spec.Namespace, status).Inc()
	chaosmetrics.ExperimentDuration.WithLabelValues("pod-delay", exp.Spec.Namespace).Observe(duration)
	chaosmetrics.ResourcesAffected.WithLabelValues("pod-delay", exp.Spec.Namespace, exp.Name).Set(float64(len(affectedPods)))

	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

// parseDurationToMs parses a duration string (e.g., "30s", "5m", "1h") and returns milliseconds
func (r *ChaosExperimentReconciler) parseDurationToMs(durationStr string) (int, error) {
	// Pattern: ^([0-9]+(s|m|h))+$
	re := regexp.MustCompile(`(\d+)([smh])`)
	matches := re.FindAllStringSubmatch(durationStr, -1)

	if len(matches) == 0 {
		return 0, fmt.Errorf("invalid duration format")
	}

	totalMs := 0
	for _, match := range matches {
		value, _ := strconv.Atoi(match[1])
		unit := match[2]

		switch unit {
		case "s":
			totalMs += value * 1000
		case "m":
			totalMs += value * 60 * 1000
		case "h":
			totalMs += value * 60 * 60 * 1000
		}
	}

	return totalMs, nil
}

// applyNetworkDelay adds network latency to a pod using tc (traffic control)
func (r *ChaosExperimentReconciler) applyNetworkDelay(ctx context.Context, pod *corev1.Pod, delayMs int) error {
	log := ctrl.LoggerFrom(ctx)

	// Find the first container (we'll apply delay to the pod network namespace)
	if len(pod.Spec.Containers) == 0 {
		return fmt.Errorf("no containers found in pod")
	}
	containerName := pod.Spec.Containers[0].Name

	// Commands to apply network delay using tc
	commands := [][]string{
		// First, try to delete any existing qdisc (ignore errors)
		{"tc", "qdisc", "del", "dev", "eth0", "root"},
		// Add delay using netem
		{"tc", "qdisc", "add", "dev", "eth0", "root", "netem", "delay", fmt.Sprintf("%dms", delayMs)},
	}

	for i, command := range commands {
		stdout, stderr, err := r.execInPod(ctx, pod.Namespace, pod.Name, containerName, command)
		if err != nil && i > 0 { // Ignore error for delete command (first command)
			log.Error(err, "Failed to execute command in pod",
				"pod", pod.Name,
				"command", strings.Join(command, " "),
				"stdout", stdout,
				"stderr", stderr)
			return err
		}
		log.Info("Executed command in pod",
			"pod", pod.Name,
			"command", strings.Join(command, " "),
			"stdout", stdout,
			"stderr", stderr)
	}

	return nil
}

// execInPod executes a command in a pod and returns stdout, stderr, and error
func (r *ChaosExperimentReconciler) execInPod(ctx context.Context, namespace, podName, containerName string, command []string) (string, string, error) {
	req := r.Clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   command,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(r.Config, "POST", req.URL())
	if err != nil {
		return "", "", fmt.Errorf("failed to create executor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})

	return stdout.String(), stderr.String(), err
}

// handleNodeDrain cordons and drains nodes matching the selector
func (r *ChaosExperimentReconciler) handleNodeDrain(ctx context.Context, exp *chaosv1alpha1.ChaosExperiment) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	startTime := time.Now()

	// Track active experiments
	chaosmetrics.ActiveExperiments.WithLabelValues("node-drain").Inc()
	defer chaosmetrics.ActiveExperiments.WithLabelValues("node-drain").Dec()

	// List nodes by selector
	nodeList := &corev1.NodeList{}
	selector := labels.SelectorFromSet(exp.Spec.Selector)
	if err := r.List(ctx, nodeList, client.MatchingLabelsSelector{Selector: selector}); err != nil {
		log.Error(err, "Failed to list nodes")
		exp.Status.Message = "Error: Failed to list nodes"
		_ = r.Status().Update(ctx, exp)
		return ctrl.Result{}, err
	}

	if len(nodeList.Items) == 0 {
		log.Info("No nodes found for selector", "selector", exp.Spec.Selector)
		exp.Status.Message = "No nodes found matching selector"
		_ = r.Status().Update(ctx, exp)
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	// Shuffle the list of nodes
	rand.Shuffle(len(nodeList.Items), func(i, j int) {
		nodeList.Items[i], nodeList.Items[j] = nodeList.Items[j], nodeList.Items[i]
	})

	// Determine how many nodes to drain
	drainCount := exp.Spec.Count
	if drainCount <= 0 {
		drainCount = 1 // Default to 1 if not specified or invalid
	}
	if drainCount > len(nodeList.Items) {
		drainCount = len(nodeList.Items)
	}

	// Cordon and drain selected nodes
	drainedNodes := []string{}
	for i := 0; i < drainCount; i++ {
		node := &nodeList.Items[i]
		log.Info("Cordoning and draining node", "node", node.Name)

		// Cordon the node (mark as unschedulable)
		if err := r.cordonNode(ctx, node); err != nil {
			log.Error(err, "Failed to cordon node", "node", node.Name)
			continue
		}

		// Drain the node (evict pods)
		if err := r.drainNode(ctx, node); err != nil {
			log.Error(err, "Failed to drain node", "node", node.Name)
			continue
		}

		drainedNodes = append(drainedNodes, node.Name)
	}

	// Update status
	now := metav1.Now()
	exp.Status.LastRunTime = &now
	status := statusSuccess
	if len(drainedNodes) > 0 {
		exp.Status.Message = fmt.Sprintf("Successfully drained %d node(s): %v", len(drainedNodes), drainedNodes)
	} else {
		exp.Status.Message = "Failed to drain any nodes"
		status = statusFailure
	}
	if err := r.Status().Update(ctx, exp); err != nil {
		log.Error(err, "Failed to update ChaosExperiment status")
		return ctrl.Result{}, err
	}

	// Record metrics
	duration := time.Since(startTime).Seconds()
	chaosmetrics.ExperimentsTotal.WithLabelValues("node-drain", exp.Spec.Namespace, status).Inc()
	chaosmetrics.ExperimentDuration.WithLabelValues("node-drain", exp.Spec.Namespace).Observe(duration)
	chaosmetrics.ResourcesAffected.WithLabelValues("node-drain", exp.Spec.Namespace, exp.Name).Set(float64(len(drainedNodes)))

	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

// cordonNode marks a node as unschedulable
func (r *ChaosExperimentReconciler) cordonNode(ctx context.Context, node *corev1.Node) error {
	log := ctrl.LoggerFrom(ctx)

	// Check if already cordoned
	if node.Spec.Unschedulable {
		log.Info("Node is already cordoned", "node", node.Name)
		return nil
	}

	// Mark as unschedulable
	node.Spec.Unschedulable = true
	if err := r.Update(ctx, node); err != nil {
		return fmt.Errorf("failed to cordon node: %w", err)
	}

	log.Info("Successfully cordoned node", "node", node.Name)
	return nil
}

// drainNode evicts all pods from a node
func (r *ChaosExperimentReconciler) drainNode(ctx context.Context, node *corev1.Node) error {
	log := ctrl.LoggerFrom(ctx)

	// List all pods on this node
	podList := &corev1.PodList{}
	if err := r.List(ctx, podList, client.MatchingFields{"spec.nodeName": node.Name}); err != nil {
		return fmt.Errorf("failed to list pods on node: %w", err)
	}

	log.Info("Found pods on node", "node", node.Name, "count", len(podList.Items))

	// Evict each pod
	evictedCount := 0
	for _, pod := range podList.Items {
		// Skip pods that are already terminating or in a final state
		if pod.DeletionTimestamp != nil {
			continue
		}
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			continue
		}

		// Skip DaemonSet pods (they can't be evicted and will be recreated anyway)
		if isDaemonSetPod(&pod) {
			log.Info("Skipping DaemonSet pod", "pod", pod.Name, "namespace", pod.Namespace)
			continue
		}

		// Skip static pods (managed by kubelet)
		if isStaticPod(&pod) {
			log.Info("Skipping static pod", "pod", pod.Name, "namespace", pod.Namespace)
			continue
		}

		log.Info("Evicting pod from node", "pod", pod.Name, "namespace", pod.Namespace, "node", node.Name)

		// Try to delete the pod gracefully
		if err := r.Delete(ctx, &pod, client.GracePeriodSeconds(30)); err != nil {
			log.Error(err, "Failed to evict pod", "pod", pod.Name, "namespace", pod.Namespace)
			continue
		}

		evictedCount++
	}

	log.Info("Evicted pods from node", "node", node.Name, "evicted", evictedCount, "total", len(podList.Items))
	return nil
}

// isDaemonSetPod checks if a pod is managed by a DaemonSet
func isDaemonSetPod(pod *corev1.Pod) bool {
	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "DaemonSet" {
			return true
		}
	}
	return false
}

// isStaticPod checks if a pod is a static pod (managed by kubelet)
func isStaticPod(pod *corev1.Pod) bool {
	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "Node" {
			return true
		}
	}
	// Static pods typically have this annotation
	if _, exists := pod.Annotations["kubernetes.io/config.source"]; exists {
		return true
	}
	return false
}

// calculateRetryDelay calculates the delay before the next retry based on backoff strategy
func (r *ChaosExperimentReconciler) calculateRetryDelay(exp *chaosv1alpha1.ChaosExperiment) time.Duration {
	// Get base delay
	baseDelay := defaultRetryDelay
	if exp.Spec.RetryDelay != "" {
		if parsed, err := r.parseDuration(exp.Spec.RetryDelay); err == nil {
			baseDelay = parsed
		}
	}

	// Apply backoff strategy
	backoffStrategy := exp.Spec.RetryBackoff
	if backoffStrategy == "" {
		backoffStrategy = defaultRetryBackoff
	}

	retryCount := exp.Status.RetryCount
	if backoffStrategy == "exponential" {
		// Exponential backoff: delay * 2^retryCount (capped at 10 minutes)
		delay := baseDelay * time.Duration(1<<uint(retryCount))
		maxDelay := 10 * time.Minute
		if delay > maxDelay {
			delay = maxDelay
		}
		return delay
	}

	// Fixed backoff: always use base delay
	return baseDelay
}

// parseDuration parses a duration string (e.g., "30s", "5m", "1h") and returns time.Duration
func (r *ChaosExperimentReconciler) parseDuration(durationStr string) (time.Duration, error) {
	re := regexp.MustCompile(`(\d+)([smh])`)
	matches := re.FindAllStringSubmatch(durationStr, -1)

	if len(matches) == 0 {
		return 0, fmt.Errorf("invalid duration format")
	}

	var totalDuration time.Duration
	for _, match := range matches {
		value, _ := strconv.Atoi(match[1])
		unit := match[2]

		switch unit {
		case "s":
			totalDuration += time.Duration(value) * time.Second
		case "m":
			totalDuration += time.Duration(value) * time.Minute
		case "h":
			totalDuration += time.Duration(value) * time.Hour
		}
	}

	return totalDuration, nil
}

// shouldRetry determines if the experiment should be retried
func (r *ChaosExperimentReconciler) shouldRetry(exp *chaosv1alpha1.ChaosExperiment) bool {
	maxRetries := exp.Spec.MaxRetries
	if maxRetries == 0 {
		maxRetries = defaultMaxRetries
	}

	return exp.Status.RetryCount < maxRetries
}

// handleExperimentFailure updates status and determines retry behavior
func (r *ChaosExperimentReconciler) handleExperimentFailure(ctx context.Context, exp *chaosv1alpha1.ChaosExperiment, errorMsg string) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Update error information
	exp.Status.LastError = errorMsg
	exp.Status.Message = fmt.Sprintf("Failed: %s", errorMsg)

	// Check if we should retry
	if r.shouldRetry(exp) {
		// Increment retry count
		exp.Status.RetryCount++
		exp.Status.Phase = phasePending

		// Calculate next retry time
		retryDelay := r.calculateRetryDelay(exp)
		nextRetry := metav1.NewTime(time.Now().Add(retryDelay))
		exp.Status.NextRetryTime = &nextRetry

		exp.Status.Message = fmt.Sprintf("Failed: %s (Retry %d/%d in %s)", errorMsg, exp.Status.RetryCount, exp.Spec.MaxRetries, retryDelay)

		log.Info("Experiment failed, scheduling retry",
			"error", errorMsg,
			"retryCount", exp.Status.RetryCount,
			"maxRetries", exp.Spec.MaxRetries,
			"nextRetry", retryDelay)

		if err := r.Status().Update(ctx, exp); err != nil {
			log.Error(err, "Failed to update ChaosExperiment status")
			return ctrl.Result{}, err
		}

		// Requeue after retry delay
		return ctrl.Result{RequeueAfter: retryDelay}, nil
	}

	// Max retries exceeded
	exp.Status.Phase = phaseFailed
	exp.Status.Message = fmt.Sprintf("Failed after %d retries: %s", exp.Status.RetryCount, errorMsg)
	exp.Status.NextRetryTime = nil

	log.Info("Experiment failed, max retries exceeded",
		"error", errorMsg,
		"retryCount", exp.Status.RetryCount)

	if err := r.Status().Update(ctx, exp); err != nil {
		log.Error(err, "Failed to update ChaosExperiment status")
		return ctrl.Result{}, err
	}

	// Don't requeue, experiment has permanently failed
	return ctrl.Result{}, nil
}

// filterExcludedPods removes pods with exclusion labels or in excluded namespaces
func (r *ChaosExperimentReconciler) filterExcludedPods(ctx context.Context, pods []corev1.Pod, namespace string) []corev1.Pod {
	log := ctrl.LoggerFrom(ctx)
	eligible := []corev1.Pod{}

	// Check if namespace has exclusion annotation
	ns := &corev1.Namespace{}
	nsExcluded := false
	if err := r.Get(ctx, client.ObjectKey{Name: namespace}, ns); err == nil {
		if val, exists := ns.Annotations[chaosv1alpha1.ExclusionLabel]; exists && val == "true" {
			nsExcluded = true
			log.Info("Namespace is excluded from chaos experiments", "namespace", namespace)
		}
	}

	// If namespace is excluded, return empty list
	if nsExcluded {
		return eligible
	}

	// Filter pods with exclusion label
	for _, pod := range pods {
		if val, exists := pod.Labels[chaosv1alpha1.ExclusionLabel]; exists && val == "true" {
			log.Info("Pod excluded from chaos experiment", "pod", pod.Name, "namespace", pod.Namespace)
			continue
		}
		eligible = append(eligible, pod)
	}

	return eligible
}

// handleDryRun handles dry-run mode by previewing affected resources without executing chaos
func (r *ChaosExperimentReconciler) handleDryRun(ctx context.Context, exp *chaosv1alpha1.ChaosExperiment, pods []corev1.Pod, actionType string) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	count := exp.Spec.Count
	if count <= 0 {
		count = 1
	}
	if count > len(pods) {
		count = len(pods)
	}

	// Build preview message
	podNames := []string{}
	for i := 0; i < count && i < len(pods); i++ {
		podNames = append(podNames, pods[i].Name)
	}

	now := metav1.Now()
	exp.Status.LastRunTime = &now
	exp.Status.Message = fmt.Sprintf("DRY RUN: Would %s %d pod(s): %v",
		actionType, count, podNames)
	exp.Status.Phase = "Completed"

	if err := r.Status().Update(ctx, exp); err != nil {
		log.Error(err, "Failed to update ChaosExperiment status")
		return ctrl.Result{}, err
	}

	log.Info("Dry run completed", "action", actionType, "wouldAffect", count, "pods", podNames)

	// Don't requeue for dry-run experiments
	return ctrl.Result{}, nil
}

// checkExperimentLifecycle manages the experiment lifecycle based on experimentDuration
// Returns (shouldContinue, error)
func (r *ChaosExperimentReconciler) checkExperimentLifecycle(ctx context.Context, exp *chaosv1alpha1.ChaosExperiment) (bool, error) {
	log := ctrl.LoggerFrom(ctx)

	// If experiment is already completed, don't continue
	if exp.Status.Phase == phaseCompleted {
		log.Info("Experiment already completed", "completedAt", exp.Status.CompletedAt)
		return false, nil
	}

	// Initialize StartTime on first run
	if exp.Status.StartTime == nil {
		now := metav1.Now()
		exp.Status.StartTime = &now
		exp.Status.Phase = phaseRunning
		if err := r.Status().Update(ctx, exp); err != nil {
			log.Error(err, "Failed to update experiment start time")
			return false, err
		}
		log.Info("Experiment started", "startTime", now)
	}

	// Check if experimentDuration is set
	if exp.Spec.ExperimentDuration == "" {
		// No duration limit, continue indefinitely
		return true, nil
	}

	// Parse experiment duration
	duration, err := r.parseDuration(exp.Spec.ExperimentDuration)
	if err != nil {
		log.Error(err, "Failed to parse experimentDuration", "duration", exp.Spec.ExperimentDuration)
		return false, err
	}

	// Calculate end time
	endTime := exp.Status.StartTime.Add(duration)
	now := time.Now()

	// Check if duration has been exceeded
	if now.After(endTime) {
		log.Info("Experiment duration exceeded, completing experiment",
			"startTime", exp.Status.StartTime,
			"duration", duration,
			"endTime", endTime)

		// Mark as completed
		completedAt := metav1.Now()
		exp.Status.CompletedAt = &completedAt
		exp.Status.Phase = phaseCompleted
		exp.Status.Message = fmt.Sprintf("Experiment completed after running for %s", duration)

		if err := r.Status().Update(ctx, exp); err != nil {
			log.Error(err, "Failed to update experiment completion status")
			return false, err
		}

		return false, nil
	}

	// Calculate time until experiment should complete
	timeUntilCompletion := endTime.Sub(now)
	log.Info("Experiment still running",
		"timeRemaining", timeUntilCompletion,
		"willCompleteAt", endTime)

	// Continue experiment
	return true, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChaosExperimentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1alpha1.ChaosExperiment{}).
		Named("chaosexperiment").
		Complete(r)
}
