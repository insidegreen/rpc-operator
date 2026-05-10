/*
Copyright 2026.

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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
)

// helloWorldSpec returns a valid PipelineSpec mirroring the sample CR.
func helloWorldSpec() rpcv1alpha1.PipelineSpec {
	return rpcv1alpha1.PipelineSpec{
		Input: rpcv1alpha1.ComponentSpec{
			Type: "generate",
			Config: runtime.RawExtension{Raw: []byte(
				`{"mapping":"root = \"hello world\"","interval":"1s","count":5}`,
			)},
		},
		Processors: []rpcv1alpha1.ComponentSpec{{
			Type: "mapping",
			Config: runtime.RawExtension{Raw: []byte(
				`{"mapping":"root = content().uppercase()"}`,
			)},
		}},
		Output: rpcv1alpha1.ComponentSpec{
			Type:   "stdout",
			Config: runtime.RawExtension{Raw: []byte(`{}`)},
		},
	}
}

var _ = Describe("Pipeline Controller", func() {
	const (
		resourceName = "test-pipeline"
		namespace    = "default"
	)

	var (
		ctx                  = context.Background()
		nn                   = types.NamespacedName{Name: resourceName, Namespace: namespace}
		controllerReconciler *PipelineReconciler
	)

	BeforeEach(func() {
		controllerReconciler = &PipelineReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}

		By("creating the Pipeline CR")
		pipe := &rpcv1alpha1.Pipeline{
			ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: namespace},
			Spec:       helloWorldSpec(),
		}
		Expect(k8sClient.Create(ctx, pipe)).To(Succeed())
	})

	AfterEach(func() {
		By("removing leftover Pipeline + child resources")
		pipe := &rpcv1alpha1.Pipeline{}
		if err := k8sClient.Get(ctx, nn, pipe); err == nil {
			// Force-clear finalizer in case the test left one behind.
			pipe.Finalizers = nil
			_ = k8sClient.Update(ctx, pipe)
			_ = k8sClient.Delete(ctx, pipe)
		}
		_ = k8sClient.Delete(ctx, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: namespace},
		})
		_ = k8sClient.Delete(ctx, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: resourceName + "-config", Namespace: namespace},
		})
	})

	It("creates a ConfigMap and Pod with owner references", func() {
		// First reconcile adds the finalizer, then requeues.
		_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
		Expect(err).NotTo(HaveOccurred())

		// Second reconcile creates the children.
		_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
		Expect(err).NotTo(HaveOccurred())

		By("having a ConfigMap with the rendered pipeline.yaml")
		cm := &corev1.ConfigMap{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name: resourceName + "-config", Namespace: namespace,
		}, cm)).To(Succeed())
		Expect(cm.Data).To(HaveKey("pipeline.yaml"))
		Expect(cm.Data["pipeline.yaml"]).To(ContainSubstring("generate:"))
		Expect(cm.Data["pipeline.yaml"]).To(ContainSubstring("uppercase"))
		Expect(cm.OwnerReferences).To(HaveLen(1))
		Expect(*cm.OwnerReferences[0].Controller).To(BeTrue())
		Expect(cm.OwnerReferences[0].Kind).To(Equal("Pipeline"))

		By("having a Pod referencing the ConfigMap")
		pod := &corev1.Pod{}
		Expect(k8sClient.Get(ctx, nn, pod)).To(Succeed())
		Expect(pod.Spec.Containers).To(HaveLen(1))
		Expect(pod.Spec.Containers[0].Image).To(Equal(defaultImage))
		Expect(pod.OwnerReferences).To(HaveLen(1))
		Expect(*pod.OwnerReferences[0].Controller).To(BeTrue())
		Expect(pod.Labels).To(HaveKeyWithValue("rpc.operator.io/pipeline", resourceName))

		By("setting status.podName on the Pipeline")
		pipe := &rpcv1alpha1.Pipeline{}
		Expect(k8sClient.Get(ctx, nn, pipe)).To(Succeed())
		Expect(pipe.Status.PodName).To(Equal(resourceName))
		Expect(pipe.Status.ObservedGeneration).To(Equal(pipe.Generation))
	})

	It("removes the finalizer on delete and the CR disappears", func() {
		// Reconcile to add finalizer + create children.
		_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
		Expect(err).NotTo(HaveOccurred())
		_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
		Expect(err).NotTo(HaveOccurred())

		// Confirm finalizer present.
		pipe := &rpcv1alpha1.Pipeline{}
		Expect(k8sClient.Get(ctx, nn, pipe)).To(Succeed())
		Expect(pipe.Finalizers).To(ContainElement(finalizerName))

		// Delete the CR — API server keeps it around (with DeletionTimestamp) until
		// the finalizer is removed.
		Expect(k8sClient.Delete(ctx, pipe)).To(Succeed())

		// Reconcile observes the deletion timestamp and removes the finalizer.
		_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
		Expect(err).NotTo(HaveOccurred())

		// CR should now be gone.
		Eventually(func() bool {
			err := k8sClient.Get(ctx, nn, &rpcv1alpha1.Pipeline{})
			return apierrors.IsNotFound(err)
		}).Should(BeTrue())
	})
})
