package v1alpha1

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Ein ephemeral Pipeline muss tief kopiert werden (Pointer-Felder dürfen nicht
// geteilt werden), sonst leakt ein Mutieren der Kopie in das Original.
func TestPipelineDeepCopyEphemeral(t *testing.T) {
	now := metav1.Now()
	orig := &Pipeline{
		Spec: PipelineSpec{
			Ephemeral: &EphemeralSpec{
				TTLAfterSuccess: metav1.Duration{Duration: time.Hour},
				TTLAfterFailure: metav1.Duration{Duration: 72 * time.Hour},
			},
		},
		Status: PipelineStatus{
			CompletionTime:   &now,
			CompletionResult: "Succeeded",
		},
	}

	cp := orig.DeepCopy()

	if cp.Spec.Ephemeral == orig.Spec.Ephemeral {
		t.Fatal("Ephemeral pointer was shared, expected a deep copy")
	}
	cp.Spec.Ephemeral.TTLAfterSuccess = metav1.Duration{Duration: 5 * time.Minute}
	if orig.Spec.Ephemeral.TTLAfterSuccess.Duration != time.Hour {
		t.Fatal("mutating the copy leaked into the original (shallow copy)")
	}
	if cp.Status.CompletionTime == orig.Status.CompletionTime {
		t.Fatal("CompletionTime pointer was shared, expected a deep copy")
	}
}
