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
	"fmt"
	"regexp"
	"strings"

	chaosv1alpha1 "github.com/neogan74/k8s-chaos/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// ErrorType categorizes different types of errors
type ErrorType string

const (
	// ErrorTypePermission indicates RBAC or authentication failures
	ErrorTypePermission ErrorType = "permission"
	// ErrorTypeExecution indicates runtime errors during chaos injection
	ErrorTypeExecution ErrorType = "execution"
	// ErrorTypeValidation indicates invalid experiment configuration
	ErrorTypeValidation ErrorType = "validation"
	// ErrorTypeTimeout indicates operation timeouts
	ErrorTypeTimeout ErrorType = "timeout"
	// ErrorTypeUnknown indicates uncategorized errors
	ErrorTypeUnknown ErrorType = "unknown"
)

// ChaosError wraps K8s API errors with additional context and categorization
type ChaosError struct {
	// Original is the underlying error
	Original error
	// Type categorizes the error
	Type ErrorType
	// Resource is the K8s resource type (e.g., "pods", "nodes")
	Resource string
	// Verb is the operation attempted (e.g., "list", "delete", "update")
	Verb string
	// Namespace is the namespace context (empty for cluster-scoped resources)
	Namespace string
	// APIGroup is the API group ("core" for core resources)
	APIGroup string
	// Subresource is the subresource if applicable (e.g., "ephemeralcontainers", "eviction")
	Subresource string
	// Operation is a human-readable description of what was attempted
	Operation string
}

// Error implements the error interface
func (ce *ChaosError) Error() string {
	if ce.Type == ErrorTypePermission {
		return FormatErrorMessage(ce)
	}
	return ce.Original.Error()
}

// Unwrap implements error unwrapping for errors.Is/As
func (ce *ChaosError) Unwrap() error {
	return ce.Original
}

// ClassifyError analyzes a K8s API error and returns structured error information
func ClassifyError(err error) *ChaosError {
	if err == nil {
		return nil
	}

	ce := &ChaosError{
		Original: err,
		Type:     ErrorTypeUnknown,
	}

	// Check for permission errors (403 Forbidden or 401 Unauthorized)
	if apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err) {
		ce.Type = ErrorTypePermission
		ce.Resource, ce.Verb, ce.Namespace, ce.APIGroup, ce.Subresource = extractPermissionDetails(err)
		return ce
	}

	// Check for timeout errors
	if apierrors.IsTimeout(err) {
		ce.Type = ErrorTypeTimeout
		return ce
	}

	// Check for validation errors
	if apierrors.IsInvalid(err) || apierrors.IsBadRequest(err) {
		ce.Type = ErrorTypeValidation
		return ce
	}

	// Default to execution error
	ce.Type = ErrorTypeExecution
	return ce
}

// extractPermissionDetails parses K8s error messages to extract RBAC details
// It handles various K8s error message formats:
// - 'pods is forbidden: User "..." cannot list resource "pods" in API group "" in namespace "default"'
// - 'nodes "worker-1" is forbidden: User "..." cannot update resource "nodes" in API group ""'
// - 'pods/ephemeralcontainers "pod-name" is forbidden: ...'
func extractPermissionDetails(err error) (resource, verb, namespace, apiGroup, subresource string) {
	if err == nil {
		return
	}

	errMsg := err.Error()

	// Extract verb (list, get, delete, update, create, patch)
	verbRe := regexp.MustCompile(`cannot (list|get|delete|update|create|patch)`)
	if matches := verbRe.FindStringSubmatch(errMsg); len(matches) > 1 {
		verb = matches[1]
	}

	// Extract resource name and subresource
	// Handles both "pods" and "pods/ephemeralcontainers"
	resourceRe := regexp.MustCompile(`resource "([^"]+)"`)
	if matches := resourceRe.FindStringSubmatch(errMsg); len(matches) > 1 {
		fullResource := matches[1]
		if parts := strings.Split(fullResource, "/"); len(parts) == 2 {
			resource = parts[0]
			subresource = parts[1]
		} else {
			resource = fullResource
		}
	}

	// Extract namespace (only present for namespaced resources)
	nsRe := regexp.MustCompile(`in namespace "([^"]+)"`)
	if matches := nsRe.FindStringSubmatch(errMsg); len(matches) > 1 {
		namespace = matches[1]
	}

	// Extract API group (empty string means core API group)
	apiGroupRe := regexp.MustCompile(`in API group "([^"]*)"`)
	if matches := apiGroupRe.FindStringSubmatch(errMsg); len(matches) > 1 {
		apiGroup = matches[1]
		if apiGroup == "" {
			apiGroup = "core"
		}
	}

	return
}

// FormatErrorMessage creates a user-friendly error message with remediation steps
// Format:
// Permission denied: cannot {verb} {resource}/{subresource} in namespace {ns}.
// Missing permission: {resource}/{verb}/{subresource}.
// Troubleshooting: {link}
// Check with: kubectl auth can-i ...
// Fix: make manifests && kubectl apply -f config/rbac/
func FormatErrorMessage(ce *ChaosError) string {
	if ce == nil || ce.Type != ErrorTypePermission {
		if ce != nil {
			return ce.Original.Error()
		}
		return ""
	}

	var msg strings.Builder

	// 1. Describe what failed
	msg.WriteString("Permission denied: cannot ")
	if ce.Verb != "" {
		msg.WriteString(ce.Verb)
		msg.WriteString(" ")
	}
	if ce.Subresource != "" {
		msg.WriteString(fmt.Sprintf("%s/%s", ce.Resource, ce.Subresource))
	} else if ce.Resource != "" {
		msg.WriteString(ce.Resource)
	} else {
		msg.WriteString("perform operation")
	}
	if ce.Namespace != "" {
		msg.WriteString(fmt.Sprintf(" in namespace %s", ce.Namespace))
	}
	msg.WriteString(". ")

	// 2. Specific missing permission
	msg.WriteString("Missing permission: ")
	if ce.Resource != "" {
		msg.WriteString(ce.Resource)
		msg.WriteString("/")
	}
	if ce.Verb != "" {
		msg.WriteString(ce.Verb)
	}
	if ce.Subresource != "" {
		msg.WriteString("/")
		msg.WriteString(ce.Subresource)
	}
	msg.WriteString(". ")

	// 3. Troubleshooting link
	msg.WriteString("Troubleshooting: https://github.com/neogan74/k8s-chaos/blob/main/docs/TROUBLESHOOTING.md#permission-issues. ")

	// 4. Check command
	msg.WriteString("Check with: kubectl auth can-i ")
	if ce.Verb != "" {
		msg.WriteString(ce.Verb)
		msg.WriteString(" ")
	}
	if ce.Resource != "" {
		msg.WriteString(ce.Resource)
	}
	if ce.Subresource != "" {
		msg.WriteString("/")
		msg.WriteString(ce.Subresource)
	}
	msg.WriteString(" --as=system:serviceaccount:k8s-chaos-system:k8s-chaos-controller-manager")
	if ce.Namespace != "" {
		msg.WriteString(fmt.Sprintf(" -n %s", ce.Namespace))
	}
	msg.WriteString(". ")

	// 5. Fix suggestion
	msg.WriteString("Fix: make manifests && kubectl apply -f config/rbac/")

	return msg.String()
}

// WrapK8sError wraps a K8s API error with classification and operation context
func WrapK8sError(err error, operation string) *ChaosError {
	if err == nil {
		return nil
	}

	ce := ClassifyError(err)
	ce.Operation = operation
	return ce
}

// chaosErrorToHistoryError converts a ChaosError to ErrorDetails for history records
func chaosErrorToHistoryError(ce *ChaosError) *chaosv1alpha1.ErrorDetails {
	if ce == nil {
		return nil
	}

	failureReason := "Unknown"
	switch ce.Type {
	case ErrorTypePermission:
		failureReason = "PermissionDenied"
	case ErrorTypeExecution:
		failureReason = "ExecutionError"
	case ErrorTypeValidation:
		failureReason = "ValidationError"
	case ErrorTypeTimeout:
		failureReason = "Timeout"
	default:
		failureReason = "Unknown"
	}

	return &chaosv1alpha1.ErrorDetails{
		Message:       ce.Error(),
		LastError:     ce.Original.Error(),
		FailureReason: failureReason,
	}
}
