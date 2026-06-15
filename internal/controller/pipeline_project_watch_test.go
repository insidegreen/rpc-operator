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

func memberPipeline(name, ns, projectRef string) *rpcv1alpha1.Pipeline {
	p := &rpcv1alpha1.Pipeline{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}
	if projectRef != "" {
		p.Spec.ProjectRef = &rpcv1alpha1.ProjectRef{Name: projectRef}
	}
	return p
}

func TestPipelinesForProject_EnqueuesMembersByProjectRef(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := rpcv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	m1 := memberPipeline("calc-price-infos", "ns", "pv-automation")
	m2 := memberPipeline("sensor-store", "ns", "pv-automation")
	other := memberPipeline("invoice", "ns", "billing")
	bare := memberPipeline("standalone", "ns", "")
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(m1, m2, other, bare).Build()
	r := &PipelineReconciler{Client: cl, Scheme: scheme}

	project := &rpcv1alpha1.PipelineProject{ObjectMeta: metav1.ObjectMeta{Name: "pv-automation", Namespace: "ns"}}
	reqs := r.pipelinesForProject(context.Background(), project)

	if len(reqs) != 2 {
		t.Fatalf("expected 2 member requests, got %d: %v", len(reqs), reqs)
	}
	got := make(map[string]bool, len(reqs))
	for _, req := range reqs {
		got[req.Name] = true
	}
	for _, name := range []string{"calc-price-infos", "sensor-store"} {
		if !got[name] {
			t.Errorf("expected request for member %q, got %v", name, reqs)
		}
	}
	if got["invoice"] || got["standalone"] {
		t.Errorf("must not enqueue non-members, got %v", reqs)
	}
}

func TestProjectChangeRedeploysMembers(t *testing.T) {
	base := func() *rpcv1alpha1.PipelineProject {
		return &rpcv1alpha1.PipelineProject{
			ObjectMeta: metav1.ObjectMeta{Name: "pv", Namespace: "ns", Generation: 5},
			Status: rpcv1alpha1.PipelineProjectStatus{
				CacheResources: []rpcv1alpha1.ProjectCacheResourceStatus{
					{Name: "sensor-cache", Phase: "Failed"},
				},
			},
		}
	}

	t.Run("cache readiness change triggers", func(t *testing.T) {
		oldP, newP := base(), base()
		newP.Status.CacheResources[0].Phase = "Ready" // cache became available
		if !projectChangeRedeploysMembers(oldP, newP) {
			t.Error("a cacheResources status change must re-enqueue members")
		}
	})

	t.Run("generation change triggers", func(t *testing.T) {
		oldP, newP := base(), base()
		newP.Generation = 6 // spec edit (routes/caches/cluster)
		if !projectChangeRedeploysMembers(oldP, newP) {
			t.Error("a generation change must re-enqueue members")
		}
	})

	t.Run("status-only churn does not trigger", func(t *testing.T) {
		oldP, newP := base(), base()
		newP.Status.Phase = rpcv1alpha1.ProjectPhaseDegraded // unrelated status field
		newP.Status.Conditions = []metav1.Condition{{Type: "Ready", Reason: "x"}}
		if projectChangeRedeploysMembers(oldP, newP) {
			t.Error("unrelated status churn must NOT re-enqueue members (avoids reconcile storm)")
		}
	})
}

func TestPipelinesForProject_DifferentNamespaceYieldsNoRequests(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := rpcv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	// Member references "pv-automation" but lives in another namespace.
	m1 := memberPipeline("sensor-store", "other", "pv-automation")
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(m1).Build()
	r := &PipelineReconciler{Client: cl, Scheme: scheme}

	project := &rpcv1alpha1.PipelineProject{ObjectMeta: metav1.ObjectMeta{Name: "pv-automation", Namespace: "ns"}}
	reqs := r.pipelinesForProject(context.Background(), project)
	if len(reqs) != 0 {
		t.Fatalf("expected 0 requests across namespaces, got %d: %v", len(reqs), reqs)
	}
}
