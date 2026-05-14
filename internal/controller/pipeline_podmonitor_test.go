package controller

import (
	"context"
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
)

func newPodMonitorScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	sch := runtime.NewScheme()
	_ = rpcv1alpha1.AddToScheme(sch)
	_ = corev1.AddToScheme(sch)
	_ = monitoringv1.AddToScheme(sch)
	return sch
}

// TestReconciler_CreatesPodMonitor verifies that Reconcile creates a PodMonitor
// with the correct selector and endpoint configuration.
func TestReconciler_CreatesPodMonitor(t *testing.T) {
	sch := newPodMonitorScheme(t)
	pipe := &rpcv1alpha1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pipe", Namespace: "default"},
		Spec: rpcv1alpha1.PipelineSpec{
			Input:  rpcv1alpha1.ComponentSpec{Type: "generate"},
			Output: rpcv1alpha1.ComponentSpec{Type: "stdout"},
		},
	}
	c := fake.NewClientBuilder().WithScheme(sch).WithObjects(pipe).
		WithStatusSubresource(pipe).Build()

	r := &PipelineReconciler{Client: c, Scheme: sch}
	// First reconcile adds finalizer and requeues
	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "test-pipe", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("Reconcile (1st) failed: %v", err)
	}

	// Second reconcile creates ConfigMap, Pod, PodMonitor
	_, err = r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "test-pipe", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("Reconcile (2nd) failed: %v", err)
	}

	pm := &monitoringv1.PodMonitor{}
	if err := c.Get(context.Background(), types.NamespacedName{
		Name: "test-pipe", Namespace: "default",
	}, pm); err != nil {
		t.Fatalf("PodMonitor not created: %v", err)
	}
	if pm.Spec.Selector.MatchLabels["rpc.operator.io/pipeline"] != "test-pipe" {
		t.Errorf("wrong selector: %v", pm.Spec.Selector.MatchLabels)
	}
	if len(pm.Spec.PodMetricsEndpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(pm.Spec.PodMetricsEndpoints))
	}
	ep := pm.Spec.PodMetricsEndpoints[0]
	if ep.Port == nil || *ep.Port != "http" {
		t.Errorf("wrong port: %v", ep.Port)
	}
	if ep.Path != "/metrics" {
		t.Errorf("wrong path: %q", ep.Path)
	}
	if string(ep.Interval) != "15s" {
		t.Errorf("wrong interval: %q", ep.Interval)
	}
	if len(pm.OwnerReferences) != 1 || pm.OwnerReferences[0].Name != "test-pipe" {
		t.Errorf("wrong OwnerRef: %v", pm.OwnerReferences)
	}
}

// TestReconciler_PodMonitorCRDMissing verifies graceful degradation when
// the monitoring CRD is not installed (scheme without monitoringv1).
func TestReconciler_PodMonitorCRDMissing(t *testing.T) {
	// Scheme WITHOUT monitoring types → simulates missing CRD
	sch := runtime.NewScheme()
	_ = rpcv1alpha1.AddToScheme(sch)
	_ = corev1.AddToScheme(sch)

	pipe := &rpcv1alpha1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "no-crd-pipe", Namespace: "default"},
		Spec: rpcv1alpha1.PipelineSpec{
			Input:  rpcv1alpha1.ComponentSpec{Type: "generate"},
			Output: rpcv1alpha1.ComponentSpec{Type: "stdout"},
		},
	}
	c := fake.NewClientBuilder().WithScheme(sch).WithObjects(pipe).
		WithStatusSubresource(pipe).Build()

	r := &PipelineReconciler{Client: c, Scheme: sch}
	// First reconcile adds finalizer
	_, _ = r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "no-crd-pipe", Namespace: "default"},
	})
	// Second reconcile should NOT return error despite missing CRD
	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "no-crd-pipe", Namespace: "default"},
	})
	if err != nil {
		t.Errorf("Reconcile should not fail when CRD is missing, got: %v", err)
	}
}
