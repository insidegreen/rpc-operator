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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
)

var _ = Describe("PipelineCluster Controller", func() {
	const (
		resourceName = "test-cluster"
		namespace    = "default"
	)

	var (
		ctx                  = context.Background()
		nn                   = types.NamespacedName{Name: resourceName, Namespace: namespace}
		controllerReconciler *PipelineClusterReconciler
	)

	BeforeEach(func() {
		controllerReconciler = &PipelineClusterReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}

		By("creating the PipelineCluster CR")
		cluster := &rpcv1alpha1.PipelineCluster{
			ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: namespace},
			Spec: rpcv1alpha1.PipelineClusterSpec{
				Replicas:    2,
				JSONLogging: true,
			},
		}
		Expect(k8sClient.Create(ctx, cluster)).To(Succeed())
	})

	AfterEach(func() {
		cluster := &rpcv1alpha1.PipelineCluster{}
		if err := k8sClient.Get(ctx, nn, cluster); err == nil {
			_ = k8sClient.Delete(ctx, cluster)
		}
		_ = k8sClient.Delete(ctx, &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: namespace},
		})
		_ = k8sClient.Delete(ctx, &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: namespace},
		})
		_ = k8sClient.Delete(ctx, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: resourceName + "-config", Namespace: namespace},
		})
	})

	It("creates a ConfigMap, headless Service, and StatefulSet with owner references", func() {
		_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
		Expect(err).NotTo(HaveOccurred())

		By("having a ConfigMap with the connect.yaml main config")
		cm := &corev1.ConfigMap{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name: resourceName + "-config", Namespace: namespace,
		}, cm)).To(Succeed())
		Expect(cm.Data).To(HaveKey("connect.yaml"))
		Expect(cm.Data["connect.yaml"]).To(ContainSubstring("format: json"))
		Expect(cm.OwnerReferences).To(HaveLen(1))
		Expect(cm.OwnerReferences[0].Kind).To(Equal("PipelineCluster"))

		By("having a headless Service")
		svc := &corev1.Service{}
		Expect(k8sClient.Get(ctx, nn, svc)).To(Succeed())
		Expect(svc.Spec.ClusterIP).To(Equal("None"))
		Expect(svc.OwnerReferences).To(HaveLen(1))
		Expect(svc.OwnerReferences[0].Kind).To(Equal("PipelineCluster"))

		By("having a StatefulSet with the requested replicas")
		ss := &appsv1.StatefulSet{}
		Expect(k8sClient.Get(ctx, nn, ss)).To(Succeed())
		Expect(*ss.Spec.Replicas).To(Equal(int32(2)))
		Expect(ss.OwnerReferences).To(HaveLen(1))
	})

	It("sets phase Pending when ready replicas are below desired", func() {
		_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
		Expect(err).NotTo(HaveOccurred())

		// envtest has no kubelet, so the StatefulSet never reports ready replicas.
		cluster := &rpcv1alpha1.PipelineCluster{}
		Expect(k8sClient.Get(ctx, nn, cluster)).To(Succeed())
		Expect(cluster.Status.Phase).To(Equal(rpcv1alpha1.ClusterPhasePending))
		Expect(cluster.Status.ObservedGeneration).To(Equal(cluster.Generation))
	})
})
