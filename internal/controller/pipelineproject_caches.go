/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package controller

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
	rpcnats "github.com/insidegreen/rpc-operator-claude/internal/nats"
	"github.com/insidegreen/rpc-operator-claude/internal/projectroute"
	"github.com/insidegreen/rpc-operator-claude/internal/render"
	"github.com/insidegreen/rpc-operator-claude/internal/streams"
)

// reconcileCacheResources provisions managed KV buckets and pushes every cache
// resource to all ready instances of the project's cluster. It deletes managed
// buckets (and removes instance entries via the fake) for resources that were
// applied previously but are no longer in spec. A push rejected as a lint error
// (permanent) is recorded Failed; a transient instance/NATS failure is returned
// so the reconcile requeues. If Instances is nil (older test reconciler), the
// function is a no-op and returns the existing status.
func (r *PipelineProjectReconciler) reconcileCacheResources(
	ctx context.Context, project *rpcv1alpha1.PipelineProject,
) ([]rpcv1alpha1.ProjectCacheResourceStatus, error) {
	if r.Instances == nil {
		return project.Status.CacheResources, nil
	}

	natsURL := projectroute.NATSURL(project.Name, project.Namespace)
	instanceURLs, err := r.readyClusterInstanceURLs(ctx, projectChildClusterName(project.Name), project.Namespace)
	if err != nil {
		return nil, fmt.Errorf("list cluster instances: %w", err)
	}

	desired := map[string]bool{}
	statuses := make([]rpcv1alpha1.ProjectCacheResourceStatus, 0, len(project.Spec.CacheResources))

	for i := range project.Spec.CacheResources {
		cr := &project.Spec.CacheResources[i]
		desired[cr.Name] = true
		st := rpcv1alpha1.ProjectCacheResourceStatus{Name: cr.Name, Phase: "Ready"}

		bucket := ""
		if cr.NatsKV != nil {
			bucket = projectroute.CacheBucket(project.Name, cr.Name)
			st.Bucket = bucket
			if err := r.ensureKVBucket(ctx, natsURL, bucket, cr.NatsKV); err != nil {
				statuses = append(statuses, failedCacheStatus(st, "BucketFailed", err))
				continue
			}
		}

		body, err := render.BuildCacheResourceConfig(*cr, natsURL, bucket)
		if err != nil {
			statuses = append(statuses, failedCacheStatus(st, "RenderFailed", err))
			continue
		}

		pushErr := r.pushCacheToInstances(ctx, instanceURLs, cr.Name, body)
		if pushErr != nil {
			var rej *streams.ConfigRejectedError
			if errors.As(pushErr, &rej) {
				statuses = append(statuses, failedCacheStatus(st, "PushFailed", pushErr))
				continue
			}
			return statuses, fmt.Errorf("push cache resource %s: %w", cr.Name, pushErr)
		}
		statuses = append(statuses, st)
	}

	// Teardown: resources previously applied but no longer in spec.
	for _, old := range project.Status.CacheResources {
		if desired[old.Name] {
			continue
		}
		for _, url := range instanceURLs {
			if err := r.Instances.DeleteCacheResource(ctx, url, old.Name); err != nil {
				return statuses, fmt.Errorf("delete cache resource %s from %s: %w", old.Name, url, err)
			}
		}
		if old.Bucket != "" {
			if err := r.Streams.DeleteKV(ctx, natsURL, old.Bucket); err != nil {
				return statuses, fmt.Errorf("delete kv bucket %s: %w", old.Bucket, err)
			}
		}
	}

	return statuses, nil
}

// ensureKVBucket maps the spec KV sizing to nats.KVConfig and upserts the bucket.
func (r *PipelineProjectReconciler) ensureKVBucket(
	ctx context.Context, natsURL, bucket string, kv *rpcv1alpha1.ProjectNATSKVCache,
) error {
	var cfg rpcnats.KVConfig
	if kv.TTL != nil {
		cfg.TTL = kv.TTL.Duration
	}
	if kv.History != nil {
		cfg.History = uint8(*kv.History)
	}
	if kv.MaxBytes != nil {
		cfg.MaxBytes = kv.MaxBytes.Value()
	}
	return r.Streams.EnsureKV(ctx, natsURL, bucket, cfg)
}

// pushCacheToInstances POSTs the resource config to every instance. The first
// error (transient or ConfigRejectedError) stops and is returned to the caller.
func (r *PipelineProjectReconciler) pushCacheToInstances(
	ctx context.Context, instanceURLs []string, label, body string,
) error {
	for _, url := range instanceURLs {
		if err := r.Instances.EnsureCacheResource(ctx, url, label, body); err != nil {
			return err
		}
	}
	return nil
}

// readyClusterInstanceURLs returns the streams-API base URL of each Ready pod of
// the named cluster.
func (r *PipelineProjectReconciler) readyClusterInstanceURLs(
	ctx context.Context, clusterName, namespace string,
) ([]string, error) {
	var pods corev1.PodList
	if err := r.List(ctx, &pods,
		client.InNamespace(namespace),
		client.MatchingLabels{clusterLabelKey: clusterName},
	); err != nil {
		return nil, err
	}
	var urls []string
	for i := range pods.Items {
		p := &pods.Items[i]
		o, ok := ordinalFromPodName(p.Name, clusterName)
		if !ok || !isPodReady(p) {
			continue
		}
		urls = append(urls, clusterPodURL(clusterName, namespace, o))
	}
	return urls, nil
}

// failedCacheStatus stamps a Failed phase + condition on a cache resource status.
func failedCacheStatus(
	st rpcv1alpha1.ProjectCacheResourceStatus, reason string, err error,
) rpcv1alpha1.ProjectCacheResourceStatus {
	st.Phase = "Failed"
	st.Conditions = []metav1.Condition{{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            err.Error(),
		LastTransitionTime: metav1.Now(),
	}}
	return st
}
