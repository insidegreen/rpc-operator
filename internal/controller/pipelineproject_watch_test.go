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
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
)

func TestProjectsForPipeline_EnqueuesAllProjectsEvenWhenRefCleared(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := rpcv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	p1 := &rpcv1alpha1.PipelineProject{ObjectMeta: metav1.ObjectMeta{Name: "orders", Namespace: "ns"}}
	p2 := &rpcv1alpha1.PipelineProject{ObjectMeta: metav1.ObjectMeta{Name: "billing", Namespace: "ns"}}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(p1, p2).Build()
	r := &PipelineProjectReconciler{Client: cl, Scheme: scheme}

	// Pipeline whose projectRef was just cleared (nil) — the old mapper would
	// short-circuit here and enqueue nothing, leaving the projects' status stale.
	pipe := &rpcv1alpha1.Pipeline{ObjectMeta: metav1.ObjectMeta{Name: "ingest", Namespace: "ns"}}
	reqs := r.projectsForPipeline(context.Background(), pipe)
	if len(reqs) != 2 {
		t.Fatalf("expected 2 project requests (ref cleared must still trigger), got %d: %v", len(reqs), reqs)
	}
	// Verify both project names are present.
	got := make(map[string]bool, 2)
	for _, req := range reqs {
		got[req.Name] = true
	}
	for _, name := range []string{"orders", "billing"} {
		if !got[name] {
			t.Errorf("expected request for project %q but it was absent; got %v", name, reqs)
		}
	}
}

func TestProjectsForPipeline_DifferentNamespaceYieldsNoRequests(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := rpcv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	// Projects live in "ns"; pipeline is in "other".
	p1 := &rpcv1alpha1.PipelineProject{ObjectMeta: metav1.ObjectMeta{Name: "orders", Namespace: "ns"}}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(p1).Build()
	r := &PipelineProjectReconciler{Client: cl, Scheme: scheme}

	pipe := &rpcv1alpha1.Pipeline{ObjectMeta: metav1.ObjectMeta{Name: "ingest", Namespace: "other"}}
	reqs := r.projectsForPipeline(context.Background(), pipe)
	if len(reqs) != 0 {
		t.Fatalf("expected 0 requests for pipeline in a different namespace, got %d: %v", len(reqs), reqs)
	}
}
