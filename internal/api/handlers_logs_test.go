package api_test

import (
	"net/http"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
)

func TestHandlerLogStream_PipelineNotFound(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/namespaces/default/pipelines/no-such/logs")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestHandlerLogStream_NoPod(t *testing.T) {
	pipe := &rpcv1alpha1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "idle", Namespace: "default"},
		Spec: rpcv1alpha1.PipelineSpec{
			Input:  rpcv1alpha1.ComponentSpec{Type: "generate"},
			Output: rpcv1alpha1.ComponentSpec{Type: "stdout"},
		},
		// Status.PodName deliberately empty
	}
	ts := newTestServer(t, pipe)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/namespaces/default/pipelines/idle/logs")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}
}

func TestHandlerLogStream_NoClientset(t *testing.T) {
	// Clientset ist nil in Tests — Pipeline hat Pod-Name, aber kein Clientset
	pipe := &rpcv1alpha1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "running", Namespace: "default"},
		Spec: rpcv1alpha1.PipelineSpec{
			Input:  rpcv1alpha1.ComponentSpec{Type: "generate"},
			Output: rpcv1alpha1.ComponentSpec{Type: "stdout"},
		},
		Status: rpcv1alpha1.PipelineStatus{PodName: "running-pod-abc"},
	}
	ts := newTestServer(t, pipe)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/namespaces/default/pipelines/running/logs")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", resp.StatusCode)
	}
}

func TestHandlerLogStream_Cluster_NoClientset(t *testing.T) {
	pipe := &rpcv1alpha1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "streamed", Namespace: "default"},
		Spec: rpcv1alpha1.PipelineSpec{
			ClusterRef: "etl",
			Input:      rpcv1alpha1.ComponentSpec{Type: "generate"},
			Output:     rpcv1alpha1.ComponentSpec{Type: "stdout"},
		},
		Status: rpcv1alpha1.PipelineStatus{
			Phase:            rpcv1alpha1.PhaseRunning,
			AssignedCluster:  "etl",
			AssignedInstance: "etl-0",
			StreamID:         "streamed",
		},
	}
	ts := newTestServer(t, pipe)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/namespaces/default/pipelines/streamed/logs")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 (clientset nil but cluster pod selected), got %d", resp.StatusCode)
	}
}

func TestHandlerLogStream_RouteRegistered(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/namespaces/default/pipelines/nope/logs")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	// Wenn die Route nicht registriert ist, würde der SPA-Catch-all ein HTML-Dokument liefern
	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "text/html") {
		t.Error("route not registered — SPA catch-all intercepted request")
	}
}
