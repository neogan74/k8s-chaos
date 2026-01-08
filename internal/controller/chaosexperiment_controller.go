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

	"github.com/robfig/cron/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
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
	Scheme        *runtime.Scheme
	Config        *rest.Config
	Clientset     *kubernetes.Clientset
	Recorder      record.EventRecorder
	HistoryConfig HistoryConfig
}

// +kubebuilder:rbac:groups=chaos.gushchin.dev,resources=chaosexperiments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chaos.gushchin.dev,resources=chaosexperiments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chaos.gushchin.dev,resources=chaosexperiments/finalizers,verbs=update
// +kubebuilder:rbac:groups=chaos.gushchin.dev,resources=chaosexperimenthistories,verbs=create;get;list;watch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;delete;patch
// +kubebuilder:rbac:groups="",resources=pods/exec,verbs=create
// +kubebuilder:rbac:groups="",resources=pods/ephemeralcontainers,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups="",resources=pods/eviction,verbs=create
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

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

	// Check if scheduled experiment should run now
	shouldRun, requeueAfter, err := r.checkSchedule(ctx, &exp)
	if err != nil {
		log.Error(err, "Failed to check schedule")
		exp.Status.Message = fmt.Sprintf("Schedule error: %v", err)
		_ = r.Status().Update(ctx, &exp)
		return ctrl.Result{}, err
	}
	if !shouldRun {
		// Not time to run yet, requeue for the next scheduled time
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	// Check if we're within allowed time windows
	inWindow, requeueAt, err := r.checkTimeWindows(ctx, &exp)
	if err != nil {
		log.Error(err, "Failed to check time windows")
		exp.Status.Message = fmt.Sprintf("Time window error: %v", err)
		_ = r.Status().Update(ctx, &exp)
		return ctrl.Result{}, err
	}
	if !inWindow {
		// Outside time window, requeue for the next window opening
		return ctrl.Result{RequeueAfter: time.Until(requeueAt)}, nil
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
	case "pod-memory-stress":
		return r.handlePodMemoryStress(ctx, &exp)
	case "pod-failure":
		return r.handlePodFailure(ctx, &exp)
	case "pod-restart":
		return r.handlePodRestart(ctx, &exp)
	case "pod-network-loss":
		return r.handlePodNetworkLoss(ctx, &exp)
	case "pod-disk-fill":
		return r.handlePodDiskFill(ctx, &exp)
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

	// Get eligible pods (includes namespace validation and exclusion filtering)
	eligiblePods, err := r.getEligiblePods(ctx, exp)
	if err != nil {
		return r.handleExperimentFailure(ctx, exp, fmt.Sprintf("Failed to get eligible pods: %v", err))
	}

	if len(eligiblePods) == 0 {
		log.Info("No eligible pods found")
		exp.Status.Message = "No eligible pods found matching selector (or all are excluded)"
		_ = r.Status().Update(ctx, exp)
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	// Handle dry-run mode
	if exp.Spec.DryRun {
		return r.handleDryRun(ctx, exp, eligiblePods, "delete")
	}

	// Shuffle the list of eligible pods
	rand.Shuffle(len(eligiblePods), func(i, j int) {
		eligiblePods[i], eligiblePods[j] = eligiblePods[j], eligiblePods[i]
	})

	// Delete the specified number of pods
	killCount := exp.Spec.Count
	if killCount <= 0 {
		killCount = 1 // Default to 1 if not specified or invalid
	}
	if killCount > len(eligiblePods) {
		killCount = len(eligiblePods)
	}

	killedPods := []string{}
	for i := 0; i < killCount; i++ {
		pod := eligiblePods[i]
		log.Info("Deleting pod", "pod", pod.Name, "namespace", pod.Namespace)
		if err := r.Delete(ctx, &pod); err != nil {
			if client.IgnoreNotFound(err) == nil {
				log.Info("Pod already deleted", "pod", pod.Name)
			} else {
				log.Error(err, "Failed to delete pod", "pod", pod.Name)
				r.Recorder.Event(exp, corev1.EventTypeWarning, "PodKillFailed", fmt.Sprintf("Failed to kill pod %s/%s: %v", pod.Namespace, pod.Name, err))
			}
		} else {
			killedPods = append(killedPods, pod.Name)
			r.Recorder.Event(exp, corev1.EventTypeNormal, "PodKilled", fmt.Sprintf("Killed pod %s/%s", pod.Namespace, pod.Name))
			r.Recorder.Event(&pod, corev1.EventTypeWarning, "ChaosKilled", fmt.Sprintf("Pod killed by chaos experiment %s", exp.Name))
		}
	}

	// Check if we killed any pods
	if len(killedPods) == 0 {
		return r.handleExperimentFailure(ctx, exp, "Failed to kill any pods")
	}

	// Update status - success
	now := metav1.Now()
	exp.Status.LastRunTime = &now
	exp.Status.Message = fmt.Sprintf("Successfully killed %d pod(s)", len(killedPods))

	// Reset retry counters on success
	if err := r.handleExperimentSuccess(ctx, exp); err != nil {
		log.Error(err, "Failed to update ChaosExperiment status")
		return ctrl.Result{}, err
	}

	// Record metrics
	duration := time.Since(startTime).Seconds()
	chaosmetrics.ExperimentsTotal.WithLabelValues("pod-kill", exp.Spec.Namespace, statusSuccess).Inc()
	chaosmetrics.ExperimentDuration.WithLabelValues("pod-kill", exp.Spec.Namespace).Observe(duration)
	chaosmetrics.ResourcesAffected.WithLabelValues("pod-kill", exp.Spec.Namespace, exp.Name).Set(float64(len(killedPods)))

	// Create history record
	affectedResources := buildResourceReferences("deleted", exp.Spec.Namespace, killedPods, "Pod")
	if err := r.createHistoryRecord(ctx, exp, statusSuccess, affectedResources, startTime, nil); err != nil {
		log.Error(err, "Failed to create history record")
		// Don't fail the experiment if history recording fails
	}

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

	// Get eligible pods (includes namespace validation and exclusion filtering)
	eligiblePods, err := r.getEligiblePods(ctx, exp)
	if err != nil {
		log.Error(err, "Failed to get eligible pods")
		exp.Status.Message = fmt.Sprintf("Error: Failed to get eligible pods: %v", err)
		_ = r.Status().Update(ctx, exp)
		return ctrl.Result{}, err
	}

	if len(eligiblePods) == 0 {
		log.Info("No eligible pods found")
		exp.Status.Message = "No eligible pods found matching selector (or all are excluded)"
		_ = r.Status().Update(ctx, exp)
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	// Handle dry-run mode
	if exp.Spec.DryRun {
		return r.handleDryRun(ctx, exp, eligiblePods, fmt.Sprintf("add %dms network delay to", delayMs))
	}

	// Shuffle the list of eligible pods
	rand.Shuffle(len(eligiblePods), func(i, j int) {
		eligiblePods[i], eligiblePods[j] = eligiblePods[j], eligiblePods[i]
	})

	// Determine how many pods to affect
	affectCount := exp.Spec.Count
	if affectCount <= 0 {
		affectCount = 1 // Default to 1 if not specified or invalid
	}
	if affectCount > len(eligiblePods) {
		affectCount = len(eligiblePods)
	}

	// Apply network delay to selected pods
	affectedPods := []string{}
	for i := 0; i < affectCount; i++ {
		pod := eligiblePods[i]
		log.Info("Adding network delay to pod", "pod", pod.Name, "namespace", pod.Namespace, "delay", delayMs)

		// Apply delay using tc (traffic control)
		if err := r.applyNetworkDelay(ctx, &pod, delayMs); err != nil {
			log.Error(err, "Failed to apply network delay", "pod", pod.Name)
			r.Recorder.Event(exp, corev1.EventTypeWarning, "PodDelayFailed", fmt.Sprintf("Failed to delay pod %s/%s: %v", pod.Namespace, pod.Name, err))
		} else {
			affectedPods = append(affectedPods, pod.Name)
			r.Recorder.Event(exp, corev1.EventTypeNormal, "PodDelayInjected", fmt.Sprintf("Injected %dms delay into pod %s/%s", delayMs, pod.Namespace, pod.Name))
			r.Recorder.Event(&pod, corev1.EventTypeWarning, "ChaosDelayInjected", fmt.Sprintf("Network delay %dms injected by chaos experiment %s", delayMs, exp.Name))
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

	// Create history record
	affectedResources := buildResourceReferences(fmt.Sprintf("network-delay-%dms", delayMs), exp.Spec.Namespace, affectedPods, "Pod")
	var errorDetails *chaosv1alpha1.ErrorDetails
	if status == statusFailure {
		errorDetails = &chaosv1alpha1.ErrorDetails{
			Message:       exp.Status.Message,
			FailureReason: "ExecutionError",
		}
	}
	if err := r.createHistoryRecord(ctx, exp, status, affectedResources, startTime, errorDetails); err != nil {
		log.Error(err, "Failed to create history record")
		// Don't fail the experiment if history recording fails
	}

	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

// handlePodCPUStress injects ephemeral containers with stress-ng to consume CPU resources
func (r *ChaosExperimentReconciler) handlePodCPUStress(ctx context.Context, exp *chaosv1alpha1.ChaosExperiment) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	startTime := time.Now()

	// Track active experiments
	chaosmetrics.ActiveExperiments.WithLabelValues("pod-cpu-stress").Inc()
	defer chaosmetrics.ActiveExperiments.WithLabelValues("pod-cpu-stress").Dec()

	// Validate namespace
	if exp.Spec.Namespace == "" {
		return r.handleExperimentFailure(ctx, exp, "Namespace not specified")
	}

	// Validate required fields for pod-cpu-stress
	if exp.Spec.CPULoad <= 0 {
		return r.handleExperimentFailure(ctx, exp, "CPULoad must be specified and greater than 0 for pod-cpu-stress")
	}

	if exp.Spec.Duration == "" {
		return r.handleExperimentFailure(ctx, exp, "Duration is required for pod-cpu-stress action")
	}

	// Parse duration for stress-ng timeout
	durationSeconds, err := r.parseDurationToSeconds(exp.Spec.Duration)
	if err != nil {
		return r.handleExperimentFailure(ctx, exp, fmt.Sprintf("Invalid duration format: %s", exp.Spec.Duration))
	}

	// Get eligible pods (includes namespace validation and exclusion filtering)
	eligiblePods, err := r.getEligiblePods(ctx, exp)
	if err != nil {
		return r.handleExperimentFailure(ctx, exp, fmt.Sprintf("Failed to get eligible pods: %v", err))
	}

	if len(eligiblePods) == 0 {
		log.Info("No eligible pods found")
		exp.Status.Message = "No eligible pods found matching selector (or all are excluded)"
		_ = r.Status().Update(ctx, exp)
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	// Handle dry-run mode
	if exp.Spec.DryRun {
		return r.handleDryRun(ctx, exp, eligiblePods, fmt.Sprintf("apply %d%% CPU stress to", exp.Spec.CPULoad))
	}

	// Shuffle the list of eligible pods
	rand.Shuffle(len(eligiblePods), func(i, j int) {
		eligiblePods[i], eligiblePods[j] = eligiblePods[j], eligiblePods[i]
	})

	// Determine how many pods to affect
	affectCount := exp.Spec.Count
	if affectCount <= 0 {
		affectCount = 1 // Default to 1 if not specified or invalid
	}
	if affectCount > len(eligiblePods) {
		affectCount = len(eligiblePods)
	}

	// Set default CPU workers if not specified
	cpuWorkers := exp.Spec.CPUWorkers
	if cpuWorkers <= 0 {
		cpuWorkers = 1
	}

	// Apply CPU stress to selected pods
	affectedPods := []string{}
	for i := 0; i < affectCount; i++ {
		pod := eligiblePods[i]
		log.Info("Injecting CPU stress into pod",
			"pod", pod.Name,
			"namespace", pod.Namespace,
			"cpuLoad", exp.Spec.CPULoad,
			"cpuWorkers", cpuWorkers,
			"duration", durationSeconds)

		// Inject ephemeral container with stress-ng
		containerName, err := r.injectCPUStressContainer(ctx, &pod, exp.Spec.CPULoad, cpuWorkers, durationSeconds)
		if err != nil {
			log.Error(err, "Failed to inject CPU stress container", "pod", pod.Name)
			chaosmetrics.ExperimentErrors.WithLabelValues("pod-cpu-stress", exp.Spec.Namespace).Inc()
			r.Recorder.Event(exp, corev1.EventTypeWarning, "PodCPUStressFailed", fmt.Sprintf("Failed to inject CPU stress into pod %s/%s: %v", pod.Namespace, pod.Name, err))
		} else if containerName != "" {
			// Track the affected pod for cleanup later
			r.trackAffectedPod(exp, pod.Namespace, pod.Name, containerName)
			affectedPods = append(affectedPods, pod.Name)
			r.Recorder.Event(exp, corev1.EventTypeNormal, "PodCPUStressInjected", fmt.Sprintf("Injected %d%% CPU stress into pod %s/%s", exp.Spec.CPULoad, pod.Namespace, pod.Name))
			r.Recorder.Event(&pod, corev1.EventTypeWarning, "ChaosCPUStressInjected", fmt.Sprintf("CPU stress %d%% injected by chaos experiment %s", exp.Spec.CPULoad, exp.Name))
		}
	}

	// Update status
	now := metav1.Now()
	exp.Status.LastRunTime = &now
	status := statusSuccess
	if len(affectedPods) > 0 {
		exp.Status.Message = fmt.Sprintf("Successfully applied %d%% CPU stress to %d pod(s) for %ds",
			exp.Spec.CPULoad, len(affectedPods), durationSeconds)
		// Reset retry count on success
		exp.Status.RetryCount = 0
		exp.Status.LastError = ""
		exp.Status.NextRetryTime = nil
	} else {
		exp.Status.Message = "Failed to apply CPU stress to any pods"
		status = statusFailure
	}
	if err := r.Status().Update(ctx, exp); err != nil {
		log.Error(err, "Failed to update ChaosExperiment status")
		return ctrl.Result{}, err
	}

	// Record metrics
	duration := time.Since(startTime).Seconds()
	chaosmetrics.ExperimentsTotal.WithLabelValues("pod-cpu-stress", exp.Spec.Namespace, status).Inc()
	chaosmetrics.ExperimentDuration.WithLabelValues("pod-cpu-stress", exp.Spec.Namespace).Observe(duration)
	chaosmetrics.ResourcesAffected.WithLabelValues("pod-cpu-stress", exp.Spec.Namespace, exp.Name).Set(float64(len(affectedPods)))

	// Create history record
	affectedResources := buildResourceReferences(fmt.Sprintf("cpu-stress-%d%%", exp.Spec.CPULoad), exp.Spec.Namespace, affectedPods, "Pod")
	var errorDetails *chaosv1alpha1.ErrorDetails
	if status == statusFailure {
		errorDetails = &chaosv1alpha1.ErrorDetails{
			Message:       exp.Status.Message,
			FailureReason: "ExecutionError",
		}
	}
	if err := r.createHistoryRecord(ctx, exp, status, affectedResources, startTime, errorDetails); err != nil {
		log.Error(err, "Failed to create history record")
		// Don't fail the experiment if history recording fails
	}

	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

// injectCPUStressContainer adds an ephemeral container with stress-ng to the pod
// Returns the container name for tracking purposes
func (r *ChaosExperimentReconciler) injectCPUStressContainer(ctx context.Context, pod *corev1.Pod, cpuLoad, cpuWorkers, durationSeconds int) (string, error) {
	log := ctrl.LoggerFrom(ctx)

	// Generate unique container name based on experiment
	containerName := fmt.Sprintf("chaos-cpu-stress-%d", time.Now().Unix())

	// Get the current pod to check container statuses
	currentPod := &corev1.Pod{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(pod), currentPod); err != nil {
		return "", fmt.Errorf("failed to get current pod state: %w", err)
	}

	// Check if pod is terminating
	if currentPod.DeletionTimestamp != nil {
		return "", fmt.Errorf("pod is terminating")
	}

	// Check if a chaos-cpu-stress ephemeral container is still running
	// We only want to prevent injection if there's an actively running stress container
	for _, ec := range currentPod.Spec.EphemeralContainers {
		if strings.HasPrefix(ec.Name, "chaos-cpu-stress") {
			// Check if this container is still running
			if isEphemeralContainerRunning(currentPod, ec.Name) {
				log.Info("Chaos CPU stress container is already running, skipping injection",
					"pod", pod.Name,
					"container", ec.Name)
				return "", nil // Return empty name to indicate skipped
			}
			// Container exists but has completed, we can inject a new one
			log.Info("Found completed chaos CPU stress container, will inject new one",
				"pod", pod.Name,
				"oldContainer", ec.Name,
				"newContainer", containerName)
		}
	}

	// Create ephemeral container spec with stress-ng
	ephemeralContainer := corev1.EphemeralContainer{
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:  containerName,
			Image: "alexeiled/stress-ng:latest-alpine",
			Command: []string{
				"stress-ng",
				"--cpu", fmt.Sprintf("%d", cpuWorkers),
				"--cpu-load", fmt.Sprintf("%d", cpuLoad),
				"--timeout", fmt.Sprintf("%ds", durationSeconds),
				"--metrics-brief",
			},
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse(fmt.Sprintf("%d", cpuWorkers)),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("100m"),
				},
			},
		},
	}

	// Append the ephemeral container
	currentPod.Spec.EphemeralContainers = append(currentPod.Spec.EphemeralContainers, ephemeralContainer)

	// Update the pod with the ephemeral container
	// Use SubResource to update ephemeralcontainers
	if err := r.Client.SubResource("ephemeralcontainers").Update(ctx, currentPod); err != nil {
		return "", fmt.Errorf("failed to inject ephemeral container: %w", err)
	}

	log.Info("Successfully injected CPU stress ephemeral container",
		"pod", pod.Name,
		"container", containerName,
		"cpuLoad", cpuLoad,
		"cpuWorkers", cpuWorkers,
		"duration", durationSeconds)

	return containerName, nil
}

// parseDurationToSeconds converts duration string to seconds
func (r *ChaosExperimentReconciler) parseDurationToSeconds(durationStr string) (int, error) {
	duration, err := r.parseDuration(durationStr)
	if err != nil {
		return 0, err
	}
	return int(duration.Seconds()), nil
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

	// Check if pod is terminating
	currentPod := &corev1.Pod{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(pod), currentPod); err != nil {
		return fmt.Errorf("failed to get current pod state: %w", err)
	}
	if currentPod.DeletionTimestamp != nil {
		return fmt.Errorf("pod is terminating")
	}

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

	// Handle dry-run mode for nodes
	if exp.Spec.DryRun {
		count := exp.Spec.Count
		if count <= 0 {
			count = 1
		}
		if count > len(nodeList.Items) {
			count = len(nodeList.Items)
		}

		nodeNames := []string{}
		for i := 0; i < count && i < len(nodeList.Items); i++ {
			nodeNames = append(nodeNames, nodeList.Items[i].Name)
		}

		now := metav1.Now()
		exp.Status.LastRunTime = &now
		exp.Status.Message = fmt.Sprintf("DRY RUN: Would cordon and drain %d node(s): %v", count, nodeNames)
		exp.Status.Phase = "Completed"

		if err := r.Status().Update(ctx, exp); err != nil {
			log.Error(err, "Failed to update ChaosExperiment status")
			return ctrl.Result{}, err
		}

		log.Info("Dry run completed", "action", "node-drain", "wouldAffect", count, "nodes", nodeNames)
		return ctrl.Result{}, nil
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
	newlyCordonedNodes := []string{}
	for i := 0; i < drainCount; i++ {
		node := &nodeList.Items[i]
		log.Info("Cordoning and draining node", "node", node.Name)

		// Cordon the node (mark as unschedulable)
		wasAlreadyCordoned, err := r.cordonNode(ctx, node)
		if err != nil {
			log.Error(err, "Failed to cordon node", "node", node.Name)
			r.Recorder.Event(exp, corev1.EventTypeWarning, "NodeCordonFailed", fmt.Sprintf("Failed to cordon node %s: %v", node.Name, err))
			continue
		}

		// Track nodes that we cordoned (not ones that were already cordoned)
		if !wasAlreadyCordoned {
			newlyCordonedNodes = append(newlyCordonedNodes, node.Name)
			r.Recorder.Event(node, corev1.EventTypeWarning, "ChaosCordoned", fmt.Sprintf("Node cordoned by chaos experiment %s", exp.Name))
		}

		// Drain the node (evict pods)
		if err := r.drainNode(ctx, node); err != nil {
			log.Error(err, "Failed to drain node", "node", node.Name)
			r.Recorder.Event(exp, corev1.EventTypeWarning, "NodeDrainFailed", fmt.Sprintf("Failed to drain node %s: %v", node.Name, err))
			continue
		}

		drainedNodes = append(drainedNodes, node.Name)
		r.Recorder.Event(exp, corev1.EventTypeNormal, "NodeDrained", fmt.Sprintf("Drained node %s", node.Name))
		r.Recorder.Event(node, corev1.EventTypeWarning, "ChaosDrained", fmt.Sprintf("Node drained by chaos experiment %s", exp.Name))
	}

	// Update status to track newly cordoned nodes for later uncordon
	if len(newlyCordonedNodes) > 0 {
		// Append newly cordoned nodes to the existing list (avoid duplicates)
		existingNodes := make(map[string]bool)
		for _, nodeName := range exp.Status.CordonedNodes {
			existingNodes[nodeName] = true
		}
		for _, nodeName := range newlyCordonedNodes {
			if !existingNodes[nodeName] {
				exp.Status.CordonedNodes = append(exp.Status.CordonedNodes, nodeName)
			}
		}
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

	// Create history record
	affectedResources := buildResourceReferences("drained", "", drainedNodes, "Node")
	var errorDetails *chaosv1alpha1.ErrorDetails
	if status == statusFailure {
		errorDetails = &chaosv1alpha1.ErrorDetails{
			Message:       exp.Status.Message,
			FailureReason: "ExecutionError",
		}
	}
	if err := r.createHistoryRecord(ctx, exp, status, affectedResources, startTime, errorDetails); err != nil {
		log.Error(err, "Failed to create history record")
		// Don't fail the experiment if history recording fails
	}

	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

// cordonNode marks a node as unschedulable
// Returns (wasAlreadyCordoned bool, error)
func (r *ChaosExperimentReconciler) cordonNode(ctx context.Context, node *corev1.Node) (bool, error) {
	log := ctrl.LoggerFrom(ctx)

	// Check if already cordoned
	if node.Spec.Unschedulable {
		log.Info("Node is already cordoned", "node", node.Name)
		return true, nil
	}

	// Mark as unschedulable
	node.Spec.Unschedulable = true
	if err := r.Update(ctx, node); err != nil {
		return false, fmt.Errorf("failed to cordon node: %w", err)
	}

	log.Info("Successfully cordoned node", "node", node.Name)
	return false, nil
}

// uncordonNode marks a node as schedulable
func (r *ChaosExperimentReconciler) uncordonNode(ctx context.Context, nodeName string) error {
	log := ctrl.LoggerFrom(ctx)

	// Get the node
	node := &corev1.Node{}
	if err := r.Get(ctx, client.ObjectKey{Name: nodeName}, node); err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	// Check if already uncordoned
	if !node.Spec.Unschedulable {
		log.Info("Node is already uncordoned", "node", nodeName)
		return nil
	}

	// Mark as schedulable
	node.Spec.Unschedulable = false
	if err := r.Update(ctx, node); err != nil {
		return fmt.Errorf("failed to uncordon node: %w", err)
	}

	log.Info("Successfully uncordoned node", "node", nodeName)
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
			if client.IgnoreNotFound(err) != nil {
				log.Error(err, "Failed to evict pod", "pod", pod.Name, "namespace", pod.Namespace)
				continue
			}
			// If not found, it's already gone, so we count it as evicted
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

	// Update error information and last run time
	now := metav1.Now()
	exp.Status.LastRunTime = &now
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

		// Emit event for retry
		r.Recorder.Event(exp, corev1.EventTypeWarning, "ExperimentRetrying",
			fmt.Sprintf("Experiment failed, will retry %d/%d in %s: %s",
				exp.Status.RetryCount, exp.Spec.MaxRetries, retryDelay, errorMsg))

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

	// Emit event for permanent failure
	r.Recorder.Event(exp, corev1.EventTypeWarning, "ExperimentFailed",
		fmt.Sprintf("Experiment failed after %d retries: %s", exp.Status.RetryCount, errorMsg))

	// Don't requeue, experiment has permanently failed
	return ctrl.Result{}, nil
}

// handleExperimentSuccess resets retry counters after a successful experiment execution
func (r *ChaosExperimentReconciler) handleExperimentSuccess(ctx context.Context, exp *chaosv1alpha1.ChaosExperiment) error {
	// Reset retry count and error information on success
	exp.Status.RetryCount = 0
	exp.Status.LastError = ""
	exp.Status.NextRetryTime = nil
	exp.Status.Phase = phaseCompleted

	// Update status
	if err := r.Status().Update(ctx, exp); err != nil {
		return fmt.Errorf("failed to update status after success: %w", err)
	}

	// Emit event for successful experiment
	r.Recorder.Event(exp, corev1.EventTypeNormal, "ExperimentSucceeded",
		fmt.Sprintf("Chaos experiment completed successfully: %s", exp.Status.Message))

	return nil
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

	// Track dry-run execution in metrics
	chaosmetrics.SafetyDryRunExecutions.WithLabelValues(exp.Spec.Action, exp.Spec.Namespace).Inc()

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

		// Emit event for experiment start
		r.Recorder.Event(exp, corev1.EventTypeNormal, "ExperimentStarted",
			fmt.Sprintf("Chaos experiment started: action=%s, namespace=%s, count=%d",
				exp.Spec.Action, exp.Spec.Namespace, exp.Spec.Count))
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

		// Uncordon nodes that were cordoned by this experiment (for node-drain action)
		if exp.Spec.Action == "node-drain" && len(exp.Status.CordonedNodes) > 0 {
			log.Info("Uncordoning nodes that were cordoned by this experiment",
				"nodes", exp.Status.CordonedNodes)
			for _, nodeName := range exp.Status.CordonedNodes {
				if err := r.uncordonNode(ctx, nodeName); err != nil {
					log.Error(err, "Failed to uncordon node", "node", nodeName)
					// Continue with other nodes even if one fails
				}
			}
			// Clear the list after uncordoning
			exp.Status.CordonedNodes = nil
		}

		// Cleanup ephemeral containers for experiments using them (pod-cpu-stress, pod-memory-stress, pod-network-loss, pod-disk-fill)
		if (exp.Spec.Action == "pod-cpu-stress" || exp.Spec.Action == "pod-memory-stress" || exp.Spec.Action == "pod-network-loss" || exp.Spec.Action == "pod-disk-fill") && len(exp.Status.AffectedPods) > 0 {
			log.Info("Cleaning up ephemeral containers injected by this experiment",
				"affectedPods", len(exp.Status.AffectedPods))
			if err := r.cleanupEphemeralContainers(ctx, exp); err != nil {
				log.Error(err, "Failed to cleanup ephemeral containers")
				// Continue with completion even if cleanup fails
			}
		}

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

// getEligiblePods returns pods that match the selector and are not excluded
func (r *ChaosExperimentReconciler) getEligiblePods(ctx context.Context, exp *chaosv1alpha1.ChaosExperiment) ([]corev1.Pod, error) {
	log := ctrl.LoggerFrom(ctx)

	// Validate namespace
	if exp.Spec.Namespace == "" {
		return nil, fmt.Errorf("namespace not specified")
	}

	// Choose Pods by selector
	podList := &corev1.PodList{}
	selector := labels.SelectorFromSet(exp.Spec.Selector)
	if err := r.List(ctx, podList, client.InNamespace(exp.Spec.Namespace),
		client.MatchingLabelsSelector{Selector: selector}); err != nil {
		log.Error(err, "Failed to list pods")
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	// Get namespace to check for exclusion annotation
	ns := &corev1.Namespace{}
	namespaceExcluded := false
	if err := r.Get(ctx, client.ObjectKey{Name: exp.Spec.Namespace}, ns); err == nil {
		if val, exists := ns.Annotations[chaosv1alpha1.ExclusionLabel]; exists && val == "true" {
			namespaceExcluded = true
		}
	}

	// Filter out excluded pods, terminating pods, and track exclusions in metrics
	eligiblePods := []corev1.Pod{}
	excludedByNamespace := 0
	excludedByLabel := 0
	excludedByTerminating := 0

	for _, pod := range podList.Items {
		// Skip if namespace is excluded
		if namespaceExcluded {
			excludedByNamespace++
			continue
		}

		// Skip if pod has exclusion label
		if val, exists := pod.Labels[chaosv1alpha1.ExclusionLabel]; exists && val == "true" {
			log.Info("Skipping excluded pod", "pod", pod.Name, "namespace", pod.Namespace)
			excludedByLabel++
			continue
		}

		// Skip if pod is terminating (has DeletionTimestamp set)
		if pod.DeletionTimestamp != nil {
			log.Info("Skipping terminating pod", "pod", pod.Name, "namespace", pod.Namespace, "deletionTimestamp", pod.DeletionTimestamp)
			excludedByTerminating++
			continue
		}

		eligiblePods = append(eligiblePods, pod)
	}

	// Track excluded resources in metrics
	if excludedByNamespace > 0 {
		chaosmetrics.SafetyExcludedResources.WithLabelValues(
			exp.Spec.Action,
			exp.Spec.Namespace,
			"namespace",
		).Add(float64(excludedByNamespace))
	}
	if excludedByLabel > 0 {
		chaosmetrics.SafetyExcludedResources.WithLabelValues(
			exp.Spec.Action,
			exp.Spec.Namespace,
			"pod",
		).Add(float64(excludedByLabel))
	}
	if excludedByTerminating > 0 {
		chaosmetrics.SafetyExcludedResources.WithLabelValues(
			exp.Spec.Action,
			exp.Spec.Namespace,
			"terminating",
		).Add(float64(excludedByTerminating))
	}

	return eligiblePods, nil
}

// handlePodMemoryStress injects ephemeral containers with stress-ng to stress memory
func (r *ChaosExperimentReconciler) handlePodMemoryStress(ctx context.Context, exp *chaosv1alpha1.ChaosExperiment) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	startTime := time.Now()

	// Track active experiments
	chaosmetrics.ActiveExperiments.WithLabelValues("pod-memory-stress").Inc()
	defer chaosmetrics.ActiveExperiments.WithLabelValues("pod-memory-stress").Dec()

	// Validate required fields
	if exp.Spec.Duration == "" {
		return r.handleExperimentFailure(ctx, exp, "Duration is required for pod-memory-stress action")
	}
	if exp.Spec.MemorySize == "" {
		return r.handleExperimentFailure(ctx, exp, "MemorySize must be specified for pod-memory-stress action")
	}

	// Parse duration to seconds for stress-ng timeout
	duration, err := r.parseDuration(exp.Spec.Duration)
	if err != nil {
		return r.handleExperimentFailure(ctx, exp, fmt.Sprintf("Invalid duration format: %v", err))
	}
	timeoutSeconds := int(duration.Seconds())

	// Get eligible pods
	eligiblePods, err := r.getEligiblePods(ctx, exp)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(eligiblePods) == 0 {
		log.Info("No eligible pods found for selector", "selector", exp.Spec.Selector)
		exp.Status.Message = "No eligible pods found matching selector"
		_ = r.Status().Update(ctx, exp)
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	// Handle dry-run mode
	if exp.Spec.DryRun {
		return r.handleDryRun(ctx, exp, eligiblePods, "pod-memory-stress")
	}

	// Shuffle the list of pods
	rand.Shuffle(len(eligiblePods), func(i, j int) {
		eligiblePods[i], eligiblePods[j] = eligiblePods[j], eligiblePods[i]
	})

	// Determine how many pods to stress
	stressCount := exp.Spec.Count
	if stressCount <= 0 {
		stressCount = 1
	}
	if stressCount > len(eligiblePods) {
		stressCount = len(eligiblePods)
	}

	// Set default memory workers if not specified
	memoryWorkers := exp.Spec.MemoryWorkers
	if memoryWorkers <= 0 {
		memoryWorkers = 1
	}

	// Inject ephemeral containers to stress memory
	stressedPods := []string{}
	for i := 0; i < stressCount; i++ {
		pod := eligiblePods[i]
		log.Info("Injecting memory stress into pod", "pod", pod.Name, "namespace", pod.Namespace)

		containerName, err := r.injectMemoryStressContainer(ctx, &pod, memoryWorkers, exp.Spec.MemorySize, timeoutSeconds)
		if err != nil {
			log.Error(err, "Failed to inject memory stress container", "pod", pod.Name)
			r.Recorder.Event(exp, corev1.EventTypeWarning, "PodMemoryStressFailed", fmt.Sprintf("Failed to inject memory stress into pod %s/%s: %v", pod.Namespace, pod.Name, err))
			continue
		}

		// Track the affected pod for cleanup later
		r.trackAffectedPod(exp, pod.Namespace, pod.Name, containerName)
		stressedPods = append(stressedPods, pod.Name)
		r.Recorder.Event(exp, corev1.EventTypeNormal, "PodMemoryStressInjected", fmt.Sprintf("Injected %s memory stress into pod %s/%s", exp.Spec.MemorySize, pod.Namespace, pod.Name))
		r.Recorder.Event(&pod, corev1.EventTypeWarning, "ChaosMemoryStressInjected", fmt.Sprintf("Memory stress %s injected by chaos experiment %s", exp.Spec.MemorySize, exp.Name))
	}

	// Update status
	now := metav1.Now()
	exp.Status.LastRunTime = &now
	status := statusSuccess
	if len(stressedPods) > 0 {
		exp.Status.Message = fmt.Sprintf("Successfully injected memory stress into %d pod(s) for %s", len(stressedPods), exp.Spec.Duration)
	} else {
		exp.Status.Message = "Failed to stress any pods"
		status = statusFailure
	}
	if err := r.Status().Update(ctx, exp); err != nil {
		log.Error(err, "Failed to update ChaosExperiment status")
		return ctrl.Result{}, err
	}

	// Record metrics
	duration = time.Since(startTime)
	chaosmetrics.ExperimentsTotal.WithLabelValues("pod-memory-stress", exp.Spec.Namespace, status).Inc()
	chaosmetrics.ExperimentDuration.WithLabelValues("pod-memory-stress", exp.Spec.Namespace).Observe(duration.Seconds())
	chaosmetrics.ResourcesAffected.WithLabelValues("pod-memory-stress", exp.Spec.Namespace, exp.Name).Set(float64(len(stressedPods)))

	// Create history record
	affectedResources := buildResourceReferences("memory-stress", exp.Spec.Namespace, stressedPods, "Pod")
	var errorDetails *chaosv1alpha1.ErrorDetails
	if status == statusFailure {
		errorDetails = &chaosv1alpha1.ErrorDetails{
			Message:       exp.Status.Message,
			FailureReason: "ExecutionError",
		}
	}
	if err := r.createHistoryRecord(ctx, exp, status, affectedResources, startTime, errorDetails); err != nil {
		log.Error(err, "Failed to create history record")
		// Don't fail the experiment if history recording fails
	}

	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

// injectMemoryStressContainer injects an ephemeral container that stresses memory
// Returns the container name for tracking purposes
func (r *ChaosExperimentReconciler) injectMemoryStressContainer(ctx context.Context, pod *corev1.Pod, workers int, memorySize string, timeoutSeconds int) (string, error) {
	log := ctrl.LoggerFrom(ctx)

	// Build stress-ng command
	stressCmd := fmt.Sprintf("stress-ng --vm %d --vm-bytes %s --timeout %ds --metrics-brief", workers, memorySize, timeoutSeconds)

	// Generate unique container name
	containerName := fmt.Sprintf("memory-stress-%d", time.Now().Unix())

	// Create ephemeral container with resource limits
	ephemeralContainer := corev1.EphemeralContainer{
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:    containerName,
			Image:   "ghcr.io/neogan74/stress-ng:latest",
			Command: []string{"/bin/sh", "-c", stressCmd},
		},
	}

	// Get the latest pod version
	currentPod := &corev1.Pod{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: pod.Namespace, Name: pod.Name}, currentPod); err != nil {
		return "", fmt.Errorf("failed to get current pod: %w", err)
	}

	// Check if pod is terminating
	if currentPod.DeletionTimestamp != nil {
		return "", fmt.Errorf("pod is terminating")
	}

	// Add ephemeral container
	currentPod.Spec.EphemeralContainers = append(currentPod.Spec.EphemeralContainers, ephemeralContainer)

	// Update pod with ephemeral container
	if err := r.Client.SubResource("ephemeralcontainers").Update(ctx, currentPod); err != nil {
		return "", fmt.Errorf("failed to inject ephemeral container: %w", err)
	}

	log.Info("Successfully injected memory stress ephemeral container", "pod", pod.Name, "container", containerName)
	return containerName, nil
}

// handlePodFailure kills the main process in pods to cause container crashes and restarts
func (r *ChaosExperimentReconciler) handlePodFailure(ctx context.Context, exp *chaosv1alpha1.ChaosExperiment) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	startTime := time.Now()

	// Track active experiments
	chaosmetrics.ActiveExperiments.WithLabelValues("pod-failure").Inc()
	defer chaosmetrics.ActiveExperiments.WithLabelValues("pod-failure").Dec()

	// Get eligible pods (includes namespace validation and exclusion filtering)
	eligiblePods, err := r.getEligiblePods(ctx, exp)
	if err != nil {
		return r.handleExperimentFailure(ctx, exp, fmt.Sprintf("Failed to get eligible pods: %v", err))
	}

	if len(eligiblePods) == 0 {
		log.Info("No eligible pods found")
		exp.Status.Message = "No eligible pods found matching selector (or all are excluded)"
		_ = r.Status().Update(ctx, exp)
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	// Handle dry-run mode
	if exp.Spec.DryRun {
		return r.handleDryRun(ctx, exp, eligiblePods, "cause container failure in")
	}

	// Shuffle the list of eligible pods
	rand.Shuffle(len(eligiblePods), func(i, j int) {
		eligiblePods[i], eligiblePods[j] = eligiblePods[j], eligiblePods[i]
	})

	// Determine how many pods to affect
	affectCount := exp.Spec.Count
	if affectCount <= 0 {
		affectCount = 1 // Default to 1 if not specified or invalid
	}
	if affectCount > len(eligiblePods) {
		affectCount = len(eligiblePods)
	}

	// Kill main process in selected pods to cause container crashes
	failedPods := []string{}
	for i := 0; i < affectCount; i++ {
		pod := eligiblePods[i]
		log.Info("Causing container failure in pod", "pod", pod.Name, "namespace", pod.Namespace)

		// Kill the main process (PID 1) in the first container
		if err := r.killContainerProcess(ctx, &pod); err != nil {
			log.Error(err, "Failed to kill container process", "pod", pod.Name)
			chaosmetrics.ExperimentErrors.WithLabelValues("pod-failure", exp.Spec.Namespace).Inc()
			r.Recorder.Event(exp, corev1.EventTypeWarning, "PodFailureFailed", fmt.Sprintf("Failed to cause failure in pod %s/%s: %v", pod.Namespace, pod.Name, err))
		} else {
			failedPods = append(failedPods, pod.Name)
			r.Recorder.Event(exp, corev1.EventTypeNormal, "PodFailureInjected", fmt.Sprintf("Caused failure in pod %s/%s", pod.Namespace, pod.Name))
			r.Recorder.Event(&pod, corev1.EventTypeWarning, "ChaosFailureInjected", fmt.Sprintf("Process kill injected by chaos experiment %s", exp.Name))
		}
	}

	// Check if we failed any pods
	if len(failedPods) == 0 {
		return r.handleExperimentFailure(ctx, exp, "Failed to cause container failure in any pods")
	}

	// Update status - success
	now := metav1.Now()
	exp.Status.LastRunTime = &now
	exp.Status.Message = fmt.Sprintf("Successfully caused container failure in %d pod(s)", len(failedPods))

	// Reset retry counters on success
	if err := r.handleExperimentSuccess(ctx, exp); err != nil {
		log.Error(err, "Failed to update ChaosExperiment status")
		return ctrl.Result{}, err
	}

	// Record metrics
	duration := time.Since(startTime).Seconds()
	chaosmetrics.ExperimentsTotal.WithLabelValues("pod-failure", exp.Spec.Namespace, statusSuccess).Inc()
	chaosmetrics.ExperimentDuration.WithLabelValues("pod-failure", exp.Spec.Namespace).Observe(duration)
	chaosmetrics.ResourcesAffected.WithLabelValues("pod-failure", exp.Spec.Namespace, exp.Name).Set(float64(len(failedPods)))

	// Create history record
	affectedResources := buildResourceReferences("process-killed", exp.Spec.Namespace, failedPods, "Pod")
	if err := r.createHistoryRecord(ctx, exp, statusSuccess, affectedResources, startTime, nil); err != nil {
		log.Error(err, "Failed to create history record")
		// Don't fail the experiment if history recording fails
	}

	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

// handlePodRestart gracefully restarts containers by sending SIGTERM to PID 1
func (r *ChaosExperimentReconciler) handlePodRestart(ctx context.Context, exp *chaosv1alpha1.ChaosExperiment) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	startTime := time.Now()

	// Track active experiments
	chaosmetrics.ActiveExperiments.WithLabelValues("pod-restart").Inc()
	defer chaosmetrics.ActiveExperiments.WithLabelValues("pod-restart").Dec()

	// Get eligible pods (includes namespace validation and exclusion filtering)
	eligiblePods, err := r.getEligiblePods(ctx, exp)
	if err != nil {
		return r.handleExperimentFailure(ctx, exp, fmt.Sprintf("Failed to get eligible pods: %v", err))
	}

	if len(eligiblePods) == 0 {
		log.Info("No eligible pods found")
		exp.Status.Message = "No eligible pods found matching selector (or all are excluded)"
		_ = r.Status().Update(ctx, exp)
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	// Handle dry-run mode
	if exp.Spec.DryRun {
		return r.handleDryRun(ctx, exp, eligiblePods, "gracefully restart")
	}

	// Parse restart interval if provided
	var restartInterval time.Duration
	if exp.Spec.RestartInterval != "" {
		interval, err := r.parseDuration(exp.Spec.RestartInterval)
		if err != nil {
			return r.handleExperimentFailure(ctx, exp, fmt.Sprintf("Invalid restartInterval: %v", err))
		}
		restartInterval = interval
		log.Info("Using restart interval", "interval", restartInterval)
	}

	// Shuffle the list of eligible pods
	rand.Shuffle(len(eligiblePods), func(i, j int) {
		eligiblePods[i], eligiblePods[j] = eligiblePods[j], eligiblePods[i]
	})

	// Determine how many pods to affect
	affectCount := exp.Spec.Count
	if affectCount <= 0 {
		affectCount = 1 // Default to 1 if not specified or invalid
	}
	if affectCount > len(eligiblePods) {
		affectCount = len(eligiblePods)
	}

	// Gracefully restart containers in selected pods
	restartedPods := []string{}
	for i := 0; i < affectCount; i++ {
		// Apply delay between restarts (except first)
		if i > 0 && restartInterval > 0 {
			log.Info("Waiting before next restart", "interval", restartInterval)
			time.Sleep(restartInterval)
		}

		pod := eligiblePods[i]
		log.Info("Gracefully restarting pod", "pod", pod.Name, "namespace", pod.Namespace)

		// Send SIGTERM to gracefully restart the container
		if err := r.gracefullyRestartContainer(ctx, &pod); err != nil {
			log.Error(err, "Failed to restart pod", "pod", pod.Name)
			chaosmetrics.ExperimentErrors.WithLabelValues("pod-restart", exp.Spec.Namespace).Inc()
			r.Recorder.Event(exp, corev1.EventTypeWarning, "PodRestartFailed", fmt.Sprintf("Failed to restart pod %s/%s: %v", pod.Namespace, pod.Name, err))
			// Continue with other pods even if one fails
		} else {
			restartedPods = append(restartedPods, pod.Name)
			r.Recorder.Event(exp, corev1.EventTypeNormal, "PodRestarted", fmt.Sprintf("Restarted pod %s/%s", pod.Namespace, pod.Name))
			r.Recorder.Event(&pod, corev1.EventTypeWarning, "ChaosRestarted", fmt.Sprintf("Pod restarted by chaos experiment %s", exp.Name))
		}
	}

	// Check if we restarted any pods
	if len(restartedPods) == 0 {
		return r.handleExperimentFailure(ctx, exp, "Failed to restart any pods")
	}

	// Update status - success
	now := metav1.Now()
	exp.Status.LastRunTime = &now
	exp.Status.Message = fmt.Sprintf("Successfully restarted %d pod(s)", len(restartedPods))

	// Reset retry counters on success
	if err := r.handleExperimentSuccess(ctx, exp); err != nil {
		log.Error(err, "Failed to update ChaosExperiment status")
		return ctrl.Result{}, err
	}

	// Record metrics
	duration := time.Since(startTime).Seconds()
	chaosmetrics.ExperimentsTotal.WithLabelValues("pod-restart", exp.Spec.Namespace, statusSuccess).Inc()
	chaosmetrics.ExperimentDuration.WithLabelValues("pod-restart", exp.Spec.Namespace).Observe(duration)
	chaosmetrics.ResourcesAffected.WithLabelValues("pod-restart", exp.Spec.Namespace, exp.Name).Set(float64(len(restartedPods)))

	// Create history record
	affectedResources := buildResourceReferences("container-restarted", exp.Spec.Namespace, restartedPods, "Pod")
	if err := r.createHistoryRecord(ctx, exp, statusSuccess, affectedResources, startTime, nil); err != nil {
		log.Error(err, "Failed to create history record")
		// Don't fail the experiment if history recording fails
	}

	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

// handlePodNetworkLoss injects packet loss into pods using tc netem via ephemeral containers
func (r *ChaosExperimentReconciler) handlePodNetworkLoss(ctx context.Context, exp *chaosv1alpha1.ChaosExperiment) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	startTime := time.Now()

	// Track active experiments
	chaosmetrics.ActiveExperiments.WithLabelValues("pod-network-loss").Inc()
	defer chaosmetrics.ActiveExperiments.WithLabelValues("pod-network-loss").Dec()

	// Validate required fields
	if exp.Spec.Duration == "" {
		return r.handleExperimentFailure(ctx, exp, "Duration is required for pod-network-loss action")
	}
	if exp.Spec.LossPercentage <= 0 {
		return r.handleExperimentFailure(ctx, exp, "LossPercentage must be specified and greater than 0 for pod-network-loss action")
	}

	// Parse duration to seconds for tc timeout
	duration, err := r.parseDuration(exp.Spec.Duration)
	if err != nil {
		return r.handleExperimentFailure(ctx, exp, fmt.Sprintf("Invalid duration format: %v", err))
	}
	timeoutSeconds := int(duration.Seconds())

	// Get eligible pods
	eligiblePods, err := r.getEligiblePods(ctx, exp)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(eligiblePods) == 0 {
		log.Info("No eligible pods found for selector", "selector", exp.Spec.Selector)
		exp.Status.Message = "No eligible pods found matching selector"
		_ = r.Status().Update(ctx, exp)
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	// Handle dry-run mode
	if exp.Spec.DryRun {
		return r.handleDryRun(ctx, exp, eligiblePods, "pod-network-loss")
	}

	// Shuffle the list of pods
	rand.Shuffle(len(eligiblePods), func(i, j int) {
		eligiblePods[i], eligiblePods[j] = eligiblePods[j], eligiblePods[i]
	})

	// Determine how many pods to affect
	affectCount := exp.Spec.Count
	if affectCount <= 0 {
		affectCount = 1
	}
	if affectCount > len(eligiblePods) {
		affectCount = len(eligiblePods)
	}

	// Inject ephemeral containers to apply packet loss
	affectedPods := []string{}
	for i := 0; i < affectCount; i++ {
		pod := eligiblePods[i]
		log.Info("Injecting network loss into pod",
			"pod", pod.Name,
			"namespace", pod.Namespace,
			"lossPercentage", exp.Spec.LossPercentage,
			"correlation", exp.Spec.LossCorrelation)

		containerName, err := r.injectNetworkLossContainer(ctx, &pod, exp.Spec.LossPercentage, exp.Spec.LossCorrelation, timeoutSeconds)
		if err != nil {
			log.Error(err, "Failed to inject network loss container", "pod", pod.Name)
			r.Recorder.Event(exp, corev1.EventTypeWarning, "PodNetworkLossFailed", fmt.Sprintf("Failed to inject network loss into pod %s/%s: %v", pod.Namespace, pod.Name, err))
			continue
		}

		// Track the affected pod for cleanup later
		r.trackAffectedPod(exp, pod.Namespace, pod.Name, containerName)
		affectedPods = append(affectedPods, pod.Name)
		r.Recorder.Event(exp, corev1.EventTypeNormal, "PodNetworkLossInjected", fmt.Sprintf("Injected %d%% network loss into pod %s/%s", exp.Spec.LossPercentage, pod.Namespace, pod.Name))
		r.Recorder.Event(&pod, corev1.EventTypeWarning, "ChaosNetworkLossInjected", fmt.Sprintf("Network loss %d%% injected by chaos experiment %s", exp.Spec.LossPercentage, exp.Name))
	}

	// Update status
	now := metav1.Now()
	exp.Status.LastRunTime = &now
	status := statusSuccess
	if len(affectedPods) > 0 {
		exp.Status.Message = fmt.Sprintf("Successfully injected %d%% packet loss into %d pod(s) for %s",
			exp.Spec.LossPercentage, len(affectedPods), exp.Spec.Duration)
	} else {
		exp.Status.Message = "Failed to inject network loss into any pods"
		status = statusFailure
	}
	if err := r.Status().Update(ctx, exp); err != nil {
		log.Error(err, "Failed to update ChaosExperiment status")
		return ctrl.Result{}, err
	}

	// Record metrics
	elapsed := time.Since(startTime)
	chaosmetrics.ExperimentsTotal.WithLabelValues("pod-network-loss", exp.Spec.Namespace, status).Inc()
	chaosmetrics.ExperimentDuration.WithLabelValues("pod-network-loss", exp.Spec.Namespace).Observe(elapsed.Seconds())
	chaosmetrics.ResourcesAffected.WithLabelValues("pod-network-loss", exp.Spec.Namespace, exp.Name).Set(float64(len(affectedPods)))

	// Create history record
	affectedResources := buildResourceReferences("network-loss", exp.Spec.Namespace, affectedPods, "Pod")
	if err := r.createHistoryRecord(ctx, exp, status, affectedResources, startTime, nil); err != nil {
		log.Error(err, "Failed to create history record")
		// Don't fail the experiment if history recording fails
	}

	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

// handlePodDiskFill injects disk usage into pods using an ephemeral container
func (r *ChaosExperimentReconciler) handlePodDiskFill(ctx context.Context, exp *chaosv1alpha1.ChaosExperiment) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	startTime := time.Now()

	// Track active experiments
	chaosmetrics.ActiveExperiments.WithLabelValues("pod-disk-fill").Inc()
	defer chaosmetrics.ActiveExperiments.WithLabelValues("pod-disk-fill").Dec()

	// Validate required fields
	if exp.Spec.Duration == "" {
		return r.handleExperimentFailure(ctx, exp, "Duration is required for pod-disk-fill action")
	}

	fillPercentage := exp.Spec.FillPercentage
	if fillPercentage <= 0 {
		fillPercentage = 80
	}

	// Parse duration to seconds for sleep timeout
	duration, err := r.parseDuration(exp.Spec.Duration)
	if err != nil {
		return r.handleExperimentFailure(ctx, exp, fmt.Sprintf("Invalid duration format: %v", err))
	}
	timeoutSeconds := int(duration.Seconds())

	// Get eligible pods
	eligiblePods, err := r.getEligiblePods(ctx, exp)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(eligiblePods) == 0 {
		log.Info("No eligible pods found for selector", "selector", exp.Spec.Selector)
		exp.Status.Message = "No eligible pods found matching selector"
		_ = r.Status().Update(ctx, exp)
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	// Handle dry-run mode
	if exp.Spec.DryRun {
		return r.handleDryRun(ctx, exp, eligiblePods, "pod-disk-fill")
	}

	// Shuffle the list of pods
	rand.Shuffle(len(eligiblePods), func(i, j int) {
		eligiblePods[i], eligiblePods[j] = eligiblePods[j], eligiblePods[i]
	})

	// Determine how many pods to affect
	affectCount := exp.Spec.Count
	if affectCount <= 0 {
		affectCount = 1
	}
	if affectCount > len(eligiblePods) {
		affectCount = len(eligiblePods)
	}

	// Fill disk on selected pods
	affectedPods := []string{}
	for i := 0; i < affectCount; i++ {
		pod := eligiblePods[i]

		targetPath, err := resolveDiskFillTarget(&pod, exp.Spec.VolumeName, exp.Spec.TargetPath)
		if err != nil {
			log.Error(err, "Failed to resolve disk fill target", "pod", pod.Name, "namespace", pod.Namespace)
			chaosmetrics.ExperimentErrors.WithLabelValues("pod-disk-fill", exp.Spec.Namespace).Inc()
			continue
		}

		log.Info("Injecting disk fill into pod",
			"pod", pod.Name,
			"namespace", pod.Namespace,
			"fillPercentage", fillPercentage,
			"targetPath", targetPath,
			"duration", timeoutSeconds)

		containerName, err := r.injectDiskFillContainer(ctx, &pod, fillPercentage, targetPath, timeoutSeconds)
		if err != nil {
			log.Error(err, "Failed to inject disk fill container", "pod", pod.Name)
			chaosmetrics.ExperimentErrors.WithLabelValues("pod-disk-fill", exp.Spec.Namespace).Inc()
			r.Recorder.Event(exp, corev1.EventTypeWarning, "PodDiskFillFailed", fmt.Sprintf("Failed to inject disk fill into pod %s/%s: %v", pod.Namespace, pod.Name, err))
			continue
		}
		if containerName == "" {
			continue
		}

		// Track the affected pod for cleanup later
		r.trackAffectedPod(exp, pod.Namespace, pod.Name, containerName)
		affectedPods = append(affectedPods, pod.Name)
		r.Recorder.Event(exp, corev1.EventTypeNormal, "PodDiskFillInjected", fmt.Sprintf("Injected %d%% disk fill into pod %s/%s", fillPercentage, pod.Namespace, pod.Name))
		r.Recorder.Event(&pod, corev1.EventTypeWarning, "ChaosDiskFillInjected", fmt.Sprintf("Disk fill %d%% injected by chaos experiment %s", fillPercentage, exp.Name))
	}

	// Update status
	now := metav1.Now()
	exp.Status.LastRunTime = &now
	status := statusSuccess
	if len(affectedPods) > 0 {
		exp.Status.Message = fmt.Sprintf("Successfully filled disk to %d%% on %d pod(s) for %s",
			fillPercentage, len(affectedPods), exp.Spec.Duration)
	} else {
		exp.Status.Message = "Failed to fill disk on any pods"
		status = statusFailure
	}
	if err := r.Status().Update(ctx, exp); err != nil {
		log.Error(err, "Failed to update ChaosExperiment status")
		return ctrl.Result{}, err
	}

	// Record metrics
	elapsed := time.Since(startTime)
	chaosmetrics.ExperimentsTotal.WithLabelValues("pod-disk-fill", exp.Spec.Namespace, status).Inc()
	chaosmetrics.ExperimentDuration.WithLabelValues("pod-disk-fill", exp.Spec.Namespace).Observe(elapsed.Seconds())
	chaosmetrics.ResourcesAffected.WithLabelValues("pod-disk-fill", exp.Spec.Namespace, exp.Name).Set(float64(len(affectedPods)))

	// Create history record
	affectedResources := buildResourceReferences(fmt.Sprintf("disk-fill-%d%%", fillPercentage), exp.Spec.Namespace, affectedPods, "Pod")
	if err := r.createHistoryRecord(ctx, exp, status, affectedResources, startTime, nil); err != nil {
		log.Error(err, "Failed to create history record")
		// Don't fail the experiment if history recording fails
	}

	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

// injectNetworkLossContainer injects an ephemeral container that applies packet loss using tc netem
func (r *ChaosExperimentReconciler) injectNetworkLossContainer(ctx context.Context, pod *corev1.Pod, lossPercentage, correlation, timeoutSeconds int) (string, error) {
	log := ctrl.LoggerFrom(ctx)

	// Build tc command with correlation if specified
	var tcCmd string
	if correlation > 0 {
		tcCmd = fmt.Sprintf("tc qdisc add dev eth0 root netem loss %d%% %d%% && sleep %d && tc qdisc del dev eth0 root",
			lossPercentage, correlation, timeoutSeconds)
	} else {
		tcCmd = fmt.Sprintf("tc qdisc add dev eth0 root netem loss %d%% && sleep %d && tc qdisc del dev eth0 root",
			lossPercentage, timeoutSeconds)
	}

	// Generate unique container name
	containerName := fmt.Sprintf("network-loss-%d", time.Now().Unix())

	// Create ephemeral container with NET_ADMIN capability
	ephemeralContainer := corev1.EphemeralContainer{
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:    containerName,
			Image:   "ghcr.io/neogan74/iproute2:latest",
			Command: []string{"/bin/sh", "-c", tcCmd},
			SecurityContext: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Add: []corev1.Capability{"NET_ADMIN"},
				},
			},
		},
	}

	// Get the latest pod version
	currentPod := &corev1.Pod{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: pod.Namespace, Name: pod.Name}, currentPod); err != nil {
		return "", fmt.Errorf("failed to get current pod: %w", err)
	}

	// Check if pod is terminating
	if currentPod.DeletionTimestamp != nil {
		return "", fmt.Errorf("pod is terminating")
	}

	// Add ephemeral container
	currentPod.Spec.EphemeralContainers = append(currentPod.Spec.EphemeralContainers, ephemeralContainer)

	// Update pod with ephemeral container
	if err := r.Client.SubResource("ephemeralcontainers").Update(ctx, currentPod); err != nil {
		return "", fmt.Errorf("failed to inject ephemeral container: %w", err)
	}

	log.Info("Successfully injected network loss ephemeral container",
		"pod", pod.Name,
		"container", containerName,
		"lossPercentage", lossPercentage,
		"correlation", correlation,
		"duration", timeoutSeconds)

	return containerName, nil
}

// injectDiskFillContainer injects an ephemeral container that fills disk space
// Returns the container name for tracking purposes
func (r *ChaosExperimentReconciler) injectDiskFillContainer(ctx context.Context, pod *corev1.Pod, fillPercentage int, targetPath string, timeoutSeconds int) (string, error) {
	log := ctrl.LoggerFrom(ctx)

	// Generate unique container name
	containerName := fmt.Sprintf("disk-fill-%d", time.Now().Unix())

	// Get the current pod to check container statuses
	currentPod := &corev1.Pod{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(pod), currentPod); err != nil {
		return "", fmt.Errorf("failed to get current pod state: %w", err)
	}

	// Check if pod is terminating
	if currentPod.DeletionTimestamp != nil {
		return "", fmt.Errorf("pod is terminating")
	}

	// Check if a disk-fill ephemeral container is still running
	for _, ec := range currentPod.Spec.EphemeralContainers {
		if strings.HasPrefix(ec.Name, "disk-fill") {
			if isEphemeralContainerRunning(currentPod, ec.Name) {
				log.Info("Disk fill container is already running, skipping injection",
					"pod", pod.Name,
					"container", ec.Name)
				return "", nil
			}
		}
	}

	diskFillCmd := fmt.Sprintf(`set -e
TARGET=%q
FILE="$TARGET/chaos-disk-fill.img"
PERCENT=%d
DURATION=%d

mkdir -p "$TARGET"
df_out=$(df -Pk "$TARGET" | tail -1)
total_kb=$(echo "$df_out" | awk '{print $2}')
used_kb=$(echo "$df_out" | awk '{print $3}')
if [ -z "$total_kb" ] || [ -z "$used_kb" ]; then
  echo "failed to read disk usage"
  exit 1
fi
target_kb=$((total_kb * PERCENT / 100))
fill_kb=$((target_kb - used_kb))
if [ "$fill_kb" -le 0 ]; then
  echo "disk already above target"
  sleep "$DURATION"
  exit 0
fi
fallocate_failed=0
if command -v fallocate >/dev/null 2>&1; then
  if ! fallocate -l "${fill_kb}K" "$FILE"; then
    fallocate_failed=1
  fi
else
  fallocate_failed=1
fi
if [ "$fallocate_failed" -ne 0 ]; then
  count=$((fill_kb / 1024))
  if [ "$count" -le 0 ]; then
    count=1
  fi
  dd if=/dev/zero of="$FILE" bs=1M count="$count" conv=fsync 2>/dev/null
fi
sleep "$DURATION"
rm -f "$FILE"
`, targetPath, fillPercentage, timeoutSeconds)

	ephemeralContainer := corev1.EphemeralContainer{
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:    containerName,
			Image:   "busybox:1.36",
			Command: []string{"/bin/sh", "-c", diskFillCmd},
		},
	}

	currentPod.Spec.EphemeralContainers = append(currentPod.Spec.EphemeralContainers, ephemeralContainer)

	if err := r.Client.SubResource("ephemeralcontainers").Update(ctx, currentPod); err != nil {
		return "", fmt.Errorf("failed to inject ephemeral container: %w", err)
	}

	log.Info("Successfully injected disk fill ephemeral container",
		"pod", pod.Name,
		"container", containerName,
		"fillPercentage", fillPercentage,
		"targetPath", targetPath,
		"duration", timeoutSeconds)

	return containerName, nil
}

func resolveDiskFillTarget(pod *corev1.Pod, volumeName, targetPath string) (string, error) {
	if volumeName == "" {
		if targetPath == "" {
			return "", fmt.Errorf("target path is empty")
		}
		return targetPath, nil
	}

	for _, container := range pod.Spec.Containers {
		for _, mount := range container.VolumeMounts {
			if mount.Name == volumeName {
				return mount.MountPath, nil
			}
		}
	}

	for _, container := range pod.Spec.InitContainers {
		for _, mount := range container.VolumeMounts {
			if mount.Name == volumeName {
				return mount.MountPath, nil
			}
		}
	}

	return "", fmt.Errorf("volume %q not found in pod %s/%s", volumeName, pod.Namespace, pod.Name)
}

// killContainerProcess kills the main process (PID 1) in the pod's first container to cause a crash
func (r *ChaosExperimentReconciler) killContainerProcess(ctx context.Context, pod *corev1.Pod) error {
	log := ctrl.LoggerFrom(ctx)

	// Check if pod is terminating
	currentPod := &corev1.Pod{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(pod), currentPod); err != nil {
		return fmt.Errorf("failed to get current pod state: %w", err)
	}
	if currentPod.DeletionTimestamp != nil {
		return fmt.Errorf("pod is terminating")
	}

	// Find the first container
	if len(pod.Spec.Containers) == 0 {
		return fmt.Errorf("no containers found in pod")
	}
	containerName := pod.Spec.Containers[0].Name

	// Kill PID 1 (main process) to cause container crash
	command := []string{"kill", "-9", "1"}

	stdout, stderr, err := r.execInPod(ctx, pod.Namespace, pod.Name, containerName, command)
	if err != nil {
		log.Error(err, "Failed to kill main process in container",
			"pod", pod.Name,
			"container", containerName,
			"stdout", stdout,
			"stderr", stderr)
		return err
	}

	log.Info("Successfully killed main process in container",
		"pod", pod.Name,
		"container", containerName,
		"stdout", stdout,
		"stderr", stderr)

	return nil
}

// gracefullyRestartContainer sends SIGTERM to the main process (PID 1) to trigger graceful shutdown
func (r *ChaosExperimentReconciler) gracefullyRestartContainer(ctx context.Context, pod *corev1.Pod) error {
	log := ctrl.LoggerFrom(ctx)

	// Check if pod is terminating
	currentPod := &corev1.Pod{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(pod), currentPod); err != nil {
		return fmt.Errorf("failed to get current pod state: %w", err)
	}
	if currentPod.DeletionTimestamp != nil {
		return fmt.Errorf("pod is terminating")
	}

	// Find the first container (main application container)
	if len(pod.Spec.Containers) == 0 {
		return fmt.Errorf("no containers found in pod")
	}
	containerName := pod.Spec.Containers[0].Name

	// Send SIGTERM (signal 15) to PID 1 for graceful shutdown
	// Using fallback command to handle different environments
	command := []string{"/bin/sh", "-c", "kill -15 1 || kill -TERM 1"}

	stdout, stderr, err := r.execInPod(ctx, pod.Namespace, pod.Name, containerName, command)
	if err != nil {
		log.Error(err, "Failed to send SIGTERM to main process",
			"pod", pod.Name,
			"container", containerName,
			"stdout", stdout,
			"stderr", stderr)
		return err
	}

	log.Info("Successfully sent SIGTERM to main process for graceful restart",
		"pod", pod.Name,
		"container", containerName,
		"stdout", stdout,
		"stderr", stderr)

	return nil
}

// checkSchedule determines if a scheduled experiment should run now
// Returns: shouldRun (bool), requeueAfter (time.Duration), error
func (r *ChaosExperimentReconciler) checkSchedule(ctx context.Context, exp *chaosv1alpha1.ChaosExperiment) (bool, time.Duration, error) {
	log := ctrl.LoggerFrom(ctx)

	// If no schedule is defined, always run (immediate execution)
	if exp.Spec.Schedule == "" {
		return true, time.Minute, nil // Requeue after 1 minute for continuous experiments
	}

	// Parse the cron schedule
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	schedule, err := parser.Parse(exp.Spec.Schedule)
	if err != nil {
		log.Error(err, "Failed to parse cron schedule", "schedule", exp.Spec.Schedule)
		return false, 0, fmt.Errorf("invalid schedule: %w", err)
	}

	now := time.Now()

	// Calculate when the experiment should next run
	nextScheduledTime := schedule.Next(now)

	// Determine the reference time for checking if we should run
	// Use LastScheduledTime if set, otherwise use StartTime or creation time
	var lastScheduledTime time.Time
	if exp.Status.LastScheduledTime != nil {
		lastScheduledTime = exp.Status.LastScheduledTime.Time
	} else if exp.Status.StartTime != nil {
		lastScheduledTime = exp.Status.StartTime.Time
	} else {
		lastScheduledTime = exp.CreationTimestamp.Time
	}

	// Get the last time the schedule should have fired
	lastScheduleShouldHaveFired := schedule.Next(lastScheduledTime.Add(-time.Second))

	// Check if we've missed a scheduled run
	// We should run if: the last time the schedule should have fired is in the past and
	// we haven't run since then
	shouldRun := !lastScheduleShouldHaveFired.After(now) && lastScheduleShouldHaveFired.After(lastScheduledTime)

	// Update NextScheduledTime in status
	nextTime := metav1.NewTime(nextScheduledTime)
	if exp.Status.NextScheduledTime == nil || !exp.Status.NextScheduledTime.Equal(&nextTime) {
		exp.Status.NextScheduledTime = &nextTime
		if err := r.Status().Update(ctx, exp); err != nil {
			log.Error(err, "Failed to update next scheduled time")
			// Don't fail the reconciliation for this
		}
	}

	if shouldRun {
		log.Info("Scheduled experiment should run now",
			"schedule", exp.Spec.Schedule,
			"lastScheduledTime", lastScheduledTime,
			"nextScheduledTime", nextScheduledTime)

		// Update LastScheduledTime
		nowTime := metav1.Now()
		exp.Status.LastScheduledTime = &nowTime
		if err := r.Status().Update(ctx, exp); err != nil {
			log.Error(err, "Failed to update last scheduled time")
			return false, 0, err
		}

		return true, 0, nil
	}

	// Calculate how long until the next scheduled run
	untilNext := time.Until(nextScheduledTime)
	log.Info("Scheduled experiment not due yet, requeuing",
		"schedule", exp.Spec.Schedule,
		"nextRun", nextScheduledTime,
		"requeueAfter", untilNext)

	return false, untilNext, nil
}

// checkTimeWindows determines if the current time is within allowed time windows.
// Returns: inWindow (bool), requeueAt (time.Time), error
func (r *ChaosExperimentReconciler) checkTimeWindows(ctx context.Context, exp *chaosv1alpha1.ChaosExperiment) (bool, time.Time, error) {
	log := ctrl.LoggerFrom(ctx)

	// If no time windows configured, always allowed
	if len(exp.Spec.TimeWindows) == 0 {
		return true, time.Time{}, nil
	}

	now := time.Now()

	// Check if we're within any time window
	inWindow := chaosv1alpha1.IsWithinTimeWindows(exp.Spec.TimeWindows, now)

	if inWindow {
		// We're in a window, clear the blocked condition if it exists
		r.clearBlockedByTimeWindowCondition(ctx, exp)
		log.V(1).Info("Experiment is within time window, proceeding")
		return true, time.Time{}, nil
	}

	// We're outside all windows, calculate next opening
	nextBoundary, willBeOpen := chaosv1alpha1.NextTimeWindowBoundary(exp.Spec.TimeWindows, now)

	if nextBoundary.IsZero() {
		// No future windows (e.g., absolute window in the past)
		log.Info("No future time windows available for experiment")
		r.setBlockedByTimeWindowCondition(ctx, exp, "No future time windows available", time.Time{})
		return false, now.Add(24 * time.Hour), nil // Requeue in 24 hours
	}

	log.Info("Experiment blocked by time window",
		"currentTime", now.Format(time.RFC3339),
		"nextBoundary", nextBoundary.Format(time.RFC3339),
		"boundaryOpens", willBeOpen)

	// Set condition indicating we're blocked
	r.setBlockedByTimeWindowCondition(ctx, exp,
		fmt.Sprintf("Outside allowed time window. Next window %s at %s",
			map[bool]string{true: "opens", false: "closes"}[willBeOpen],
			nextBoundary.Format(time.RFC3339)),
		nextBoundary)

	return false, nextBoundary, nil
}

// setBlockedByTimeWindowCondition sets a condition indicating the experiment is blocked by time windows
func (r *ChaosExperimentReconciler) setBlockedByTimeWindowCondition(ctx context.Context, exp *chaosv1alpha1.ChaosExperiment, message string, nextBoundary time.Time) {
	condition := metav1.Condition{
		Type:               "BlockedByTimeWindow",
		Status:             metav1.ConditionTrue,
		ObservedGeneration: exp.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             "OutsideTimeWindow",
		Message:            message,
	}

	// Update or add the condition
	updated := false
	for i, existingCondition := range exp.Status.Conditions {
		if existingCondition.Type == "BlockedByTimeWindow" {
			exp.Status.Conditions[i] = condition
			updated = true
			break
		}
	}
	if !updated {
		exp.Status.Conditions = append(exp.Status.Conditions, condition)
	}

	// Update status
	if err := r.Status().Update(ctx, exp); err != nil {
		log := ctrl.LoggerFrom(ctx)
		log.Error(err, "Failed to update BlockedByTimeWindow condition")
	}
}

// clearBlockedByTimeWindowCondition removes the BlockedByTimeWindow condition
func (r *ChaosExperimentReconciler) clearBlockedByTimeWindowCondition(ctx context.Context, exp *chaosv1alpha1.ChaosExperiment) {
	// Find and remove the condition
	for i, condition := range exp.Status.Conditions {
		if condition.Type == "BlockedByTimeWindow" {
			// Remove condition by slicing
			exp.Status.Conditions = append(exp.Status.Conditions[:i], exp.Status.Conditions[i+1:]...)

			// Update status
			if err := r.Status().Update(ctx, exp); err != nil {
				log := ctrl.LoggerFrom(ctx)
				log.Error(err, "Failed to clear BlockedByTimeWindow condition")
			}
			break
		}
	}
}

// isEphemeralContainerRunning checks if an ephemeral container is currently running in a pod
func isEphemeralContainerRunning(pod *corev1.Pod, containerName string) bool {
	// Check the pod's container statuses for the ephemeral container
	for _, status := range pod.Status.EphemeralContainerStatuses {
		if status.Name == containerName {
			// Container is running if State.Running is not nil
			if status.State.Running != nil {
				return true
			}
			// Container is not running if it's terminated or waiting
			return false
		}
	}
	// Container status not found yet (might be starting), consider it as running to be safe
	return true
}

// cleanupEphemeralContainers cleans up ephemeral containers that were injected by this experiment
// Note: Kubernetes doesn't support removing ephemeral containers directly, but we can track them
// and log their completion status. The containers will remain in the pod spec but stop consuming resources.
func (r *ChaosExperimentReconciler) cleanupEphemeralContainers(ctx context.Context, exp *chaosv1alpha1.ChaosExperiment) error {
	log := ctrl.LoggerFrom(ctx)

	if len(exp.Status.AffectedPods) == 0 {
		log.Info("No affected pods to clean up")
		return nil
	}

	log.Info("Cleaning up ephemeral containers", "affectedPods", len(exp.Status.AffectedPods))

	cleanedUp := 0
	stillRunning := 0
	errors := 0

	for _, podRef := range exp.Status.AffectedPods {
		// Parse the pod reference format: "namespace/podName:containerName"
		parts := strings.SplitN(podRef, ":", 2)
		if len(parts) != 2 {
			log.Error(nil, "Invalid pod reference format", "ref", podRef)
			errors++
			continue
		}

		podKey := parts[0]
		containerName := parts[1]

		nsPod := strings.SplitN(podKey, "/", 2)
		if len(nsPod) != 2 {
			log.Error(nil, "Invalid pod key format", "key", podKey)
			errors++
			continue
		}

		namespace := nsPod[0]
		podName := nsPod[1]

		// Get the pod
		pod := &corev1.Pod{}
		if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: podName}, pod); err != nil {
			if client.IgnoreNotFound(err) != nil {
				log.Error(err, "Failed to get pod for cleanup", "pod", podName, "namespace", namespace)
				errors++
			} else {
				// Pod was deleted, consider it cleaned up
				log.Info("Pod no longer exists, cleanup not needed", "pod", podName, "namespace", namespace)
				cleanedUp++
			}
			continue
		}

		// Check if the ephemeral container has terminated
		containerTerminated := false
		for _, status := range pod.Status.EphemeralContainerStatuses {
			if status.Name == containerName {
				if status.State.Terminated != nil {
					containerTerminated = true
					log.Info("Ephemeral container has terminated",
						"pod", podName,
						"namespace", namespace,
						"container", containerName,
						"exitCode", status.State.Terminated.ExitCode,
						"reason", status.State.Terminated.Reason)
					cleanedUp++
				} else if status.State.Running != nil {
					log.Info("Ephemeral container is still running",
						"pod", podName,
						"namespace", namespace,
						"container", containerName)
					stillRunning++
				}
				break
			}
		}

		if !containerTerminated && stillRunning == 0 {
			// Container status not found, might be starting or already cleaned up
			log.Info("Ephemeral container status not found",
				"pod", podName,
				"namespace", namespace,
				"container", containerName)
			cleanedUp++
		}
	}

	log.Info("Ephemeral container cleanup summary",
		"cleanedUp", cleanedUp,
		"stillRunning", stillRunning,
		"errors", errors,
		"total", len(exp.Status.AffectedPods))

	// Clear the affected pods list after cleanup attempt
	exp.Status.AffectedPods = nil

	return nil
}

// trackAffectedPod adds a pod to the affected pods list in the experiment status
func (r *ChaosExperimentReconciler) trackAffectedPod(exp *chaosv1alpha1.ChaosExperiment, namespace, podName, containerName string) {
	podRef := fmt.Sprintf("%s/%s:%s", namespace, podName, containerName)

	// Check if already tracked (avoid duplicates)
	for _, existing := range exp.Status.AffectedPods {
		if existing == podRef {
			return
		}
	}

	exp.Status.AffectedPods = append(exp.Status.AffectedPods, podRef)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChaosExperimentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1alpha1.ChaosExperiment{}).
		Named("chaosexperiment").
		Complete(r)
}
