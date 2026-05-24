package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
)

// clusterObj builds a PipelineCluster for seeding the fake client.
func clusterObj(name, ns string, replicas, ready int32, phase rpcv1alpha1.PipelineClusterPhase) *rpcv1alpha1.PipelineCluster {
	return &rpcv1alpha1.PipelineCluster{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec:       rpcv1alpha1.PipelineClusterSpec{Replicas: replicas},
		Status:     rpcv1alpha1.PipelineClusterStatus{Phase: phase, ReadyReplicas: ready},
	}
}

// clusterPod builds an instance pod labelled for the cluster, ready or not.
func clusterPod(name, ns, cluster string, ready bool) *corev1.Pod {
	cond := corev1.ConditionFalse
	if ready {
		cond = corev1.ConditionTrue
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name, Namespace: ns,
			Labels: map[string]string{"rpc.operator.io/cluster": cluster},
		},
		Status: corev1.PodStatus{Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: cond}}},
	}
}

// clusteredPipeline builds a Pipeline assigned (Phase-2 placement) to an instance.
func clusteredPipeline(name, ns, cluster, instance string) *rpcv1alpha1.Pipeline {
	return &rpcv1alpha1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec:       rpcv1alpha1.PipelineSpec{ClusterRef: cluster},
		Status:     rpcv1alpha1.PipelineStatus{AssignedInstance: instance},
	}
}

func TestHandlerListNamespacedClusters(t *testing.T) {
	ts := newTestServer(t, clusterObj("etl", "default", 2, 2, rpcv1alpha1.ClusterPhaseReady))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/namespaces/default/pipelineclusters")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body struct {
		Items []rpcv1alpha1.PipelineCluster `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].Name != "etl" {
		t.Fatalf("expected [etl], got %+v", body.Items)
	}
}

func TestHandlerGetCluster_NotFound(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/namespaces/default/pipelineclusters/missing")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// compile-time keep-alive for helpers used by later tasks (avoids "unused" until then).
var _ = bytes.NewReader
var _ = clusterPod
var _ = clusteredPipeline
