package controller

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
)

func podWithWaitReason(reason string) *corev1.Pod {
	if reason == "" {
		return &corev1.Pod{}
	}
	return &corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{
		{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: reason}}},
	}}}
}

func TestContainerWaitReason(t *testing.T) {
	cases := []struct {
		name string
		pod  *corev1.Pod
		want string
	}{
		{"no containers", &corev1.Pod{}, ""},
		{"running container", &corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{
			{State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
		}}}, ""},
		{"ImagePullBackOff", podWithWaitReason("ImagePullBackOff"), "ImagePullBackOff"},
		{"CrashLoopBackOff", podWithWaitReason("CrashLoopBackOff"), "CrashLoopBackOff"},
		{"ErrImagePull", podWithWaitReason("ErrImagePull"), "ErrImagePull"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := containerWaitReason(tc.pod); got != tc.want {
				t.Errorf("containerWaitReason() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDeriveCondition(t *testing.T) {
	cases := []struct {
		name       string
		pod        *corev1.Pod
		phase      rpcv1alpha1.PipelinePhase
		wantStatus metav1.ConditionStatus
		wantReason string
	}{
		{"running", podWithWaitReason(""), rpcv1alpha1.PhaseRunning, metav1.ConditionTrue, "Running"},
		{"pending", podWithWaitReason(""), rpcv1alpha1.PhasePending, metav1.ConditionUnknown, "Pending"},
		{"failed", podWithWaitReason(""), rpcv1alpha1.PhaseFailed, metav1.ConditionFalse, "PodFailed"},
		{"stopped", podWithWaitReason(""), rpcv1alpha1.PhaseStopped, metav1.ConditionFalse, "Completed"},
		{"ImagePullBackOff", podWithWaitReason("ImagePullBackOff"), rpcv1alpha1.PhasePending, metav1.ConditionFalse, "ImagePullBackOff"},
		{"ErrImagePull normalized", podWithWaitReason("ErrImagePull"), rpcv1alpha1.PhasePending, metav1.ConditionFalse, "ImagePullBackOff"},
		{"CrashLoopBackOff", podWithWaitReason("CrashLoopBackOff"), rpcv1alpha1.PhasePending, metav1.ConditionFalse, "CrashLoopBackOff"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := deriveCondition(tc.pod, tc.phase)
			if c.Type != conditionTypeReady {
				t.Errorf("Type = %q, want Ready", c.Type)
			}
			if c.Status != tc.wantStatus {
				t.Errorf("Status = %q, want %q", c.Status, tc.wantStatus)
			}
			if c.Reason != tc.wantReason {
				t.Errorf("Reason = %q, want %q", c.Reason, tc.wantReason)
			}
			if c.Message == "" {
				t.Error("Message must not be empty")
			}
		})
	}
}
