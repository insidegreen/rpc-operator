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
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
	rpcnats "github.com/insidegreen/rpc-operator-claude/internal/nats"
	"github.com/insidegreen/rpc-operator-claude/internal/projectroute"
	"github.com/insidegreen/rpc-operator-claude/internal/streams"
)

// newProjectReconcilerWithFakes builds a PipelineProjectReconciler backed by a
// fake k8s client (with the rpcv1alpha1 scheme) and the supplied fakes.
func newProjectReconcilerWithFakes(t *testing.T, natsMgr rpcnats.StreamManager, inst streams.Client) (*PipelineProjectReconciler, client.Client) {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := rpcv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	r := &PipelineProjectReconciler{
		Client:    cl,
		Scheme:    scheme,
		Streams:   natsMgr,
		Instances: inst,
	}
	return r, cl
}

// mustCreate creates obj via the fake client; fails the test on error.
func mustCreate(t *testing.T, c client.Client, obj client.Object) {
	t.Helper()
	if err := c.Create(context.Background(), obj); err != nil {
		t.Fatalf("create %T %s: %v", obj, obj.GetName(), err)
	}
}

// newReadyClusterPod returns a Ready pod labelled for the project's child cluster.
func newReadyClusterPod(project, ns string, ordinal int) *corev1.Pod {
	clusterName := projectChildClusterName(project)
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", clusterName, ordinal),
			Namespace: ns,
			Labels:    map[string]string{clusterLabelKey: clusterName},
		},
		Status: corev1.PodStatus{Conditions: []corev1.PodCondition{
			{Type: corev1.PodReady, Status: corev1.ConditionTrue},
		}},
	}
}

func TestReconcileCacheResources_ManagedPushedToAllInstances(t *testing.T) {
	const ns = "default"
	fakeNATS := rpcnats.NewFakeManager()
	fakeInst := streams.NewFakeClient()
	r, c := newProjectReconcilerWithFakes(t, fakeNATS, fakeInst)

	project := &rpcv1alpha1.PipelineProject{
		ObjectMeta: metav1.ObjectMeta{Name: "orders", Namespace: ns},
		Spec: rpcv1alpha1.PipelineProjectSpec{
			CacheResources: []rpcv1alpha1.ProjectCacheResource{
				{Name: "shared", NatsKV: &rpcv1alpha1.ProjectNATSKVCache{}},
			},
		},
	}
	mustCreate(t, c, project)
	mustCreate(t, c, newReadyClusterPod("orders", ns, 0))
	mustCreate(t, c, newReadyClusterPod("orders", ns, 1))

	statuses, err := r.reconcileCacheResources(context.Background(), project)
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 || statuses[0].Phase != "Ready" {
		t.Fatalf("status = %+v", statuses)
	}

	natsURL := projectroute.NATSURL("orders", ns)
	if !fakeNATS.HasKV(natsURL, "rpc-orders-shared") {
		t.Fatal("KV bucket not provisioned")
	}
	for _, ord := range []int{0, 1} {
		url := clusterPodURL(projectChildClusterName("orders"), ns, int32(ord))
		if !fakeInst.HasCacheResource(url, "shared") {
			t.Fatalf("resource not pushed to instance %d", ord)
		}
	}
}

func TestReconcileCacheResources_RemovalDeletesBucketFromInstances(t *testing.T) {
	const ns = "default"
	fakeNATS := rpcnats.NewFakeManager()
	fakeInst := streams.NewFakeClient()
	r, c := newProjectReconcilerWithFakes(t, fakeNATS, fakeInst)

	project := &rpcv1alpha1.PipelineProject{
		ObjectMeta: metav1.ObjectMeta{Name: "orders", Namespace: ns},
		// spec has no cacheResources → prior status entry must be torn down
		Status: rpcv1alpha1.PipelineProjectStatus{
			CacheResources: []rpcv1alpha1.ProjectCacheResourceStatus{
				{Name: "shared", Bucket: "rpc-orders-shared", Phase: "Ready"},
			},
		},
	}
	mustCreate(t, c, project)
	mustCreate(t, c, newReadyClusterPod("orders", ns, 0))

	url := clusterPodURL(projectChildClusterName("orders"), ns, 0)
	ctx := context.Background()
	_ = fakeInst.EnsureCacheResource(ctx, url, "shared", "nats_kv: {}\n")
	natsURL := projectroute.NATSURL("orders", ns)
	_ = fakeNATS.EnsureKV(ctx, natsURL, "rpc-orders-shared", rpcnats.KVConfig{})

	if _, err := r.reconcileCacheResources(ctx, project); err != nil {
		t.Fatal(err)
	}
	if fakeInst.HasCacheResource(url, "shared") {
		t.Fatal("resource should be deleted from instance fake")
	}
	if fakeNATS.HasKV(natsURL, "rpc-orders-shared") {
		t.Fatal("bucket should be deleted")
	}
}

func TestReconcileCacheResources_RepushAfterPodRestart(t *testing.T) {
	const ns = "default"
	fakeNATS := rpcnats.NewFakeManager()
	fakeInst := streams.NewFakeClient()
	r, c := newProjectReconcilerWithFakes(t, fakeNATS, fakeInst)

	project := &rpcv1alpha1.PipelineProject{
		ObjectMeta: metav1.ObjectMeta{Name: "orders", Namespace: ns},
		Spec: rpcv1alpha1.PipelineProjectSpec{CacheResources: []rpcv1alpha1.ProjectCacheResource{
			{Name: "shared", NatsKV: &rpcv1alpha1.ProjectNATSKVCache{}},
		}},
	}
	mustCreate(t, c, project)
	mustCreate(t, c, newReadyClusterPod("orders", ns, 0))

	ctx := context.Background()
	if _, err := r.reconcileCacheResources(ctx, project); err != nil {
		t.Fatal(err)
	}
	url := clusterPodURL(projectChildClusterName("orders"), ns, 0)
	fakeInst.DropPod(url)

	if _, err := r.reconcileCacheResources(ctx, project); err != nil {
		t.Fatal(err)
	}
	if !fakeInst.HasCacheResource(url, "shared") {
		t.Fatal("resource should be re-pushed after pod restart")
	}
}

func TestReconcileCacheResources_PushRejectedMarksFailed(t *testing.T) {
	const ns = "default"
	fakeNATS := rpcnats.NewFakeManager()
	fakeInst := streams.NewFakeClient()
	fakeInst.EnsureCacheErr = &streams.ConfigRejectedError{StreamID: "r", Status: 400, Body: "bad cache"}
	r, c := newProjectReconcilerWithFakes(t, fakeNATS, fakeInst)

	project := &rpcv1alpha1.PipelineProject{
		ObjectMeta: metav1.ObjectMeta{Name: "orders", Namespace: ns},
		Spec: rpcv1alpha1.PipelineProjectSpec{CacheResources: []rpcv1alpha1.ProjectCacheResource{
			{Name: "r", Config: runtime.RawExtension{Raw: []byte(`{"redis":{}}`)}},
		}},
	}
	mustCreate(t, c, project)
	mustCreate(t, c, newReadyClusterPod("orders", ns, 0))

	statuses, err := r.reconcileCacheResources(context.Background(), project)
	if err != nil {
		t.Fatalf("rejected push is permanent, must not return error: %v", err)
	}
	if len(statuses) == 0 || statuses[0].Phase != "Failed" {
		t.Fatalf("want Failed, got %+v", statuses)
	}
}

func TestReconcileCacheResources_NoInstancesFieldNoOp(t *testing.T) {
	const ns = "default"
	fakeNATS := rpcnats.NewFakeManager()
	r, c := newProjectReconcilerWithFakes(t, fakeNATS, nil) // no Instances
	r.Instances = nil

	project := &rpcv1alpha1.PipelineProject{
		ObjectMeta: metav1.ObjectMeta{Name: "orders", Namespace: ns},
		Spec: rpcv1alpha1.PipelineProjectSpec{CacheResources: []rpcv1alpha1.ProjectCacheResource{
			{Name: "shared", NatsKV: &rpcv1alpha1.ProjectNATSKVCache{}},
		}},
		Status: rpcv1alpha1.PipelineProjectStatus{
			CacheResources: []rpcv1alpha1.ProjectCacheResourceStatus{{Name: "prior", Phase: "Ready"}},
		},
	}
	mustCreate(t, c, project)

	statuses, err := r.reconcileCacheResources(context.Background(), project)
	if err != nil {
		t.Fatal(err)
	}
	// defensive: no-op returns existing status unchanged
	if len(statuses) != 1 || statuses[0].Name != "prior" {
		t.Fatalf("expected prior status preserved, got %+v", statuses)
	}
}
