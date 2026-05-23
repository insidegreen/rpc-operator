package controller

import (
	"context"
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
)

func newClusterMonitorScheme(t *testing.T, withMonitoring bool) *runtime.Scheme {
	t.Helper()
	sch := runtime.NewScheme()
	_ = rpcv1alpha1.AddToScheme(sch)
	_ = corev1.AddToScheme(sch)
	_ = appsv1.AddToScheme(sch)
	if withMonitoring {
		_ = monitoringv1.AddToScheme(sch)
	}
	return sch
}

func TestClusterReconciler_CreatesPodMonitor(t *testing.T) {
	sch := newClusterMonitorScheme(t, true)
	cl := &rpcv1alpha1.PipelineCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "etl", Namespace: "default"},
		Spec:       rpcv1alpha1.PipelineClusterSpec{Replicas: 1},
	}
	c := fake.NewClientBuilder().WithScheme(sch).WithObjects(cl).
		WithStatusSubresource(cl).Build()

	r := &PipelineClusterReconciler{Client: c, Scheme: sch}
	if _, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "etl", Namespace: "default"},
	}); err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	pm := &monitoringv1.PodMonitor{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "etl", Namespace: "default"}, pm); err != nil {
		t.Fatalf("PodMonitor not created: %v", err)
	}
	if pm.Spec.Selector.MatchLabels[clusterLabelKey] != "etl" {
		t.Errorf("wrong selector: %v", pm.Spec.Selector.MatchLabels)
	}
	if len(pm.Spec.PodMetricsEndpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(pm.Spec.PodMetricsEndpoints))
	}
	ep := pm.Spec.PodMetricsEndpoints[0]
	if ep.Port == nil || *ep.Port != "http" || ep.Path != "/metrics" || string(ep.Interval) != "15s" {
		t.Errorf("wrong endpoint: port=%v path=%q interval=%q", ep.Port, ep.Path, ep.Interval)
	}
	if len(pm.OwnerReferences) != 1 || pm.OwnerReferences[0].Name != "etl" {
		t.Errorf("wrong OwnerRef: %v", pm.OwnerReferences)
	}
}

func TestClusterReconciler_PodMonitorCRDMissing(t *testing.T) {
	sch := newClusterMonitorScheme(t, false) // no monitoringv1 → simulates missing CRD
	cl := &rpcv1alpha1.PipelineCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "no-crd", Namespace: "default"},
		Spec:       rpcv1alpha1.PipelineClusterSpec{Replicas: 1},
	}
	c := fake.NewClientBuilder().WithScheme(sch).WithObjects(cl).
		WithStatusSubresource(cl).Build()

	r := &PipelineClusterReconciler{Client: c, Scheme: sch}
	if _, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "no-crd", Namespace: "default"},
	}); err != nil {
		t.Errorf("Reconcile should not fail when monitoring CRD is missing, got: %v", err)
	}
}
