package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
)

// projectObj builds a Ready PipelineProject named "orders" in namespace "default".
func projectObj() *rpcv1alpha1.PipelineProject {
	return &rpcv1alpha1.PipelineProject{
		ObjectMeta: metav1.ObjectMeta{Name: "orders", Namespace: "default"},
		Spec: rpcv1alpha1.PipelineProjectSpec{
			Description: "routed orders",
			Routes: []rpcv1alpha1.ProjectRoute{
				{Name: "ingest", From: "order-ingest", To: []rpcv1alpha1.ProjectRouteTarget{{Pipeline: "warehouse"}}},
			},
		},
		Status: rpcv1alpha1.PipelineProjectStatus{
			Phase:   rpcv1alpha1.ProjectPhaseReady,
			Cluster: rpcv1alpha1.ProjectChildStatus{Name: "orders-cluster", Ready: 1, Total: 1},
			NATS:    rpcv1alpha1.ProjectChildStatus{Name: "orders-nats", Ready: 1, Total: 1},
		},
	}
}

func TestHandlerListNamespacedProjects(t *testing.T) {
	ts := newTestServer(t, projectObj())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/namespaces/default/pipelineprojects")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body struct {
		Items []rpcv1alpha1.PipelineProject `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].Name != "orders" {
		t.Fatalf("expected [orders], got %+v", body.Items)
	}
}

func TestHandlerGetProject(t *testing.T) {
	ts := newTestServer(t, projectObj())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/namespaces/default/pipelineprojects/orders")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var got rpcv1alpha1.PipelineProject
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Name != "orders" || got.Status.Cluster.Name != "orders-cluster" {
		t.Fatalf("unexpected project: %+v", got)
	}
}

func TestHandlerGetProjectNotFound(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/namespaces/default/pipelineprojects/missing")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestHandlerCreateProject(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	payload := `{"apiVersion":"rpc.operator.io/v1alpha1","kind":"PipelineProject",` +
		`"metadata":{"name":"neo","namespace":"default"},` +
		`"spec":{"description":"new project"}}`
	resp, err := http.Post(ts.URL+"/api/v1/namespaces/default/pipelineprojects",
		"application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var got rpcv1alpha1.PipelineProject
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Name != "neo" || got.Namespace != "default" {
		t.Fatalf("unexpected created project: %+v", got)
	}
}

func TestHandlerUpdateProjectReplacesSpec(t *testing.T) {
	ts := newTestServer(t, projectObj())
	defer ts.Close()

	// Replace routes with an empty table (pure grouping).
	payload := `{"apiVersion":"rpc.operator.io/v1alpha1","kind":"PipelineProject",` +
		`"metadata":{"name":"orders","namespace":"default"},` +
		`"spec":{"description":"degrouped"}}`
	req, _ := http.NewRequest(http.MethodPut,
		ts.URL+"/api/v1/namespaces/default/pipelineprojects/orders",
		strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var got rpcv1alpha1.PipelineProject
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Spec.Routes) != 0 || got.Spec.Description != "degrouped" {
		t.Fatalf("spec not replaced: %+v", got.Spec)
	}
}

func TestHandlerDeleteProject(t *testing.T) {
	ts := newTestServer(t, projectObj())
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodDelete,
		ts.URL+"/api/v1/namespaces/default/pipelineprojects/orders", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
