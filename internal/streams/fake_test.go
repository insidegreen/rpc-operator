package streams

import (
	"context"
	"errors"
	"testing"
)

func TestFakeClient_GetStreamStatus(t *testing.T) {
	ctx := context.Background()
	f := NewFakeClient()
	const pod, id = "http://pod-0:4195", "mypipe"

	// Not held → ErrStreamNotFound.
	if _, err := f.GetStreamStatus(ctx, pod, id); !errors.Is(err, ErrStreamNotFound) {
		t.Fatalf("expected ErrStreamNotFound for absent stream, got %v", err)
	}

	// Held → Active:true by default.
	_ = f.EnsureStream(ctx, pod, id, "input: {}\n")
	st, err := f.GetStreamStatus(ctx, pod, id)
	if err != nil || !st.Active {
		t.Fatalf("expected Active=true, got %+v err=%v", st, err)
	}

	// Marked inactive → Active:false.
	f.SetStreamActive(id, false)
	st, _ = f.GetStreamStatus(ctx, pod, id)
	if st.Active {
		t.Fatalf("expected Active=false after SetStreamActive(false), got %+v", st)
	}

	// GetErr wins over everything (simulate transport failure).
	f.GetErr = errors.New("boom")
	if _, err := f.GetStreamStatus(ctx, pod, id); err == nil || errors.Is(err, ErrStreamNotFound) {
		t.Fatalf("expected GetErr to be returned, got %v", err)
	}
}
