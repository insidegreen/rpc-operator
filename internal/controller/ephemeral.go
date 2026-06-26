/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package controller

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
)

// completionSucceeded / completionFailed are the two terminal outcomes recorded
// in status.completionResult; they select which TTL applies. F53.
const (
	completionSucceeded = "Succeeded"
	completionFailed    = "Failed"
)

// ephemeralExpiry reports whether an ephemeral pipeline's retention TTL has
// elapsed since completion, and—when not—how long remains (for RequeueAfter).
// Always recomputed from status.completionTime, so it is drift-free and survives
// operator restarts. Returns (false, 0) if the pipeline is non-ephemeral or has
// not completed.
func ephemeralExpiry(pipe *rpcv1alpha1.Pipeline) (expired bool, requeueAfter time.Duration) {
	if pipe.Spec.Ephemeral == nil || pipe.Status.CompletionTime == nil {
		return false, 0
	}
	ttl := pipe.Spec.Ephemeral.TTLAfterSuccess.Duration
	if pipe.Status.CompletionResult == completionFailed {
		ttl = pipe.Spec.Ephemeral.TTLAfterFailure.Duration
	}
	deadline := pipe.Status.CompletionTime.Add(ttl)
	rest := time.Until(deadline)
	return rest <= 0, rest
}

// markEphemeralCompletion records a one-shot run's terminal outcome on the status,
// starting the TTL clock. No-op for non-ephemeral pipelines or once already set
// (the first observed outcome wins; the pod/stream then rests until deletion).
func markEphemeralCompletion(pipe *rpcv1alpha1.Pipeline, result string) {
	if pipe.Spec.Ephemeral == nil || pipe.Status.CompletionTime != nil {
		return
	}
	now := metav1.Now()
	pipe.Status.CompletionTime = &now
	pipe.Status.CompletionResult = result
}
