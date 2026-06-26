package controller

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
)

func ephemeralPipe(result string, completedAgo time.Duration) *rpcv1alpha1.Pipeline {
	t := metav1.NewTime(time.Now().Add(-completedAgo))
	return &rpcv1alpha1.Pipeline{
		Spec: rpcv1alpha1.PipelineSpec{
			Ephemeral: &rpcv1alpha1.EphemeralSpec{
				TTLAfterSuccess: metav1.Duration{Duration: time.Hour},
				TTLAfterFailure: metav1.Duration{Duration: 72 * time.Hour},
			},
		},
		Status: rpcv1alpha1.PipelineStatus{CompletionTime: &t, CompletionResult: result},
	}
}

func TestEphemeralExpiry(t *testing.T) {
	cases := []struct {
		name        string
		result      string
		completedAgo time.Duration
		wantExpired bool
	}{
		{"success not yet expired", completionSucceeded, 30 * time.Minute, false},
		{"success expired", completionSucceeded, 90 * time.Minute, true},
		{"failure not yet expired", completionFailed, 24 * time.Hour, false},
		{"failure expired", completionFailed, 80 * time.Hour, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expired, rest := ephemeralExpiry(ephemeralPipe(tc.result, tc.completedAgo))
			if expired != tc.wantExpired {
				t.Fatalf("expired = %v, want %v (rest=%s)", expired, tc.wantExpired, rest)
			}
			if !expired && rest <= 0 {
				t.Fatalf("not expired but rest %s <= 0", rest)
			}
		})
	}
}

func TestMarkEphemeralCompletion(t *testing.T) {
	// Non-ephemeral: No-op.
	plain := &rpcv1alpha1.Pipeline{}
	markEphemeralCompletion(plain, completionSucceeded)
	if plain.Status.CompletionTime != nil {
		t.Fatal("non-ephemeral pipeline must not get a completion time")
	}

	// Ephemeral: sets once, never overwrites.
	p := &rpcv1alpha1.Pipeline{Spec: rpcv1alpha1.PipelineSpec{Ephemeral: &rpcv1alpha1.EphemeralSpec{}}}
	markEphemeralCompletion(p, completionFailed)
	if p.Status.CompletionTime == nil || p.Status.CompletionResult != completionFailed {
		t.Fatal("expected Failed completion to be recorded")
	}
	first := p.Status.CompletionTime
	markEphemeralCompletion(p, completionSucceeded)
	if p.Status.CompletionTime != first || p.Status.CompletionResult != completionFailed {
		t.Fatal("completion must not be overwritten once set")
	}
}
